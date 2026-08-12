package schedule

import (
	"context"
	"testing"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestCronJobManagerEnsureCreatesCronJob(t *testing.T) {
	t.Run("ensure should create a new cronjob if none exists", func(t *testing.T) {
		scheme := newCronJobManagerTestScheme(t)
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
		manager := defaultCronJobManager{
			Client:           fakeClient,
			scheme:           scheme,
			operatorImage:    "example.com/backup-operator:1.2.3",
			pullPolicy:       corev1.PullAlways,
			imagePullSecrets: []corev1.LocalObjectReference{{Name: "registry-secret"}},
		}
		schedule := newCronJobManagerTestSchedule()

		require.NoError(t, manager.ensure(context.Background(), schedule))

		stored := &batchv1.CronJob{}
		require.NoError(t, fakeClient.Get(context.Background(), client.ObjectKey{
			Name: schedule.CronJobName(), Namespace: schedule.Namespace,
		}, stored))

		assert.Equal(t, schedule.Spec.Schedule, stored.Spec.Schedule)
		require.Len(t, stored.Spec.JobTemplate.Spec.Template.Spec.Containers, 1)
		container := stored.Spec.JobTemplate.Spec.Template.Spec.Containers[0]
		assert.Equal(t, manager.operatorImage, container.Image)
		assert.Equal(t, manager.pullPolicy, container.ImagePullPolicy)
		assert.Equal(t, manager.imagePullSecrets, stored.Spec.JobTemplate.Spec.Template.Spec.ImagePullSecrets)
		assert.Equal(t, defaultLabels, stored.Labels)
		// owner reference is backup schdule
		require.Len(t, stored.OwnerReferences, 1)
		assert.Equal(t, schedule.Name, stored.OwnerReferences[0].Name)
		assert.True(t, *stored.OwnerReferences[0].Controller)
	})
	t.Run("ensure should sync an existing cronjob when backuschedule is changed", func(t *testing.T) {
		scheme := newCronJobManagerTestScheme(t)
		schedule := newCronJobManagerTestSchedule()
		existing := &batchv1.CronJob{
			ObjectMeta: metav1.ObjectMeta{
				Name:      schedule.CronJobName(),
				Namespace: schedule.Namespace,
				Labels:    map[string]string{"custom": "preserved"},
			},
			Spec: batchv1.CronJobSpec{Schedule: "0 0 * * *"},
		}
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(existing).Build()
		manager := defaultCronJobManager{
			Client:        fakeClient,
			scheme:        scheme,
			operatorImage: "example.com/backup-operator:2.0.0",
			pullPolicy:    corev1.PullIfNotPresent,
		}

		require.NoError(t, manager.ensure(context.Background(), schedule))

		stored := &batchv1.CronJob{}
		require.NoError(t, fakeClient.Get(context.Background(), client.ObjectKeyFromObject(existing), stored))
		assert.Equal(t, schedule.Spec.Schedule, stored.Spec.Schedule)
		assert.Equal(t, manager.operatorImage, stored.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image)
		assert.Equal(t, "preserved", stored.Labels["custom"])
		for key, value := range defaultLabels {
			assert.Equal(t, value, stored.Labels[key])
		}
	})

	t.Run("ensure should fail if cronjob can't be created", func(t *testing.T) {
		scheme := newCronJobManagerTestScheme(t)
		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithInterceptorFuncs(interceptor.Funcs{
				Create: func(_ context.Context, _ client.WithWatch, _ client.Object, _ ...client.CreateOption) error {
					return assert.AnError
				},
			}).
			Build()
		manager := defaultCronJobManager{Client: fakeClient, scheme: scheme}

		err := manager.ensure(context.Background(), newCronJobManagerTestSchedule())

		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to create CronJob")
	})

	t.Run("ensure should fail if owner reference can't be set", func(t *testing.T) {
		scheme := runtime.NewScheme()
		require.NoError(t, batchv1.AddToScheme(scheme))
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
		manager := defaultCronJobManager{Client: fakeClient, scheme: scheme}

		err := manager.ensure(context.Background(), newCronJobManagerTestSchedule())

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to create CronJob")
		assert.ErrorContains(t, err, "failed to set BackupSchedule daily as owner")
	})
}

func TestCronJobManagerDelete(t *testing.T) {
	scheme := newCronJobManagerTestScheme(t)
	schedule := newCronJobManagerTestSchedule()
	cronJob := &batchv1.CronJob{ObjectMeta: metav1.ObjectMeta{
		Name: schedule.CronJobName(), Namespace: schedule.Namespace,
	}}
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cronJob).Build()
	manager := defaultCronJobManager{Client: fakeClient, scheme: scheme}

	require.NoError(t, manager.delete(context.Background(), schedule))
	// twice to check for idempotence
	require.NoError(t, manager.delete(context.Background(), schedule))

	stored := &batchv1.CronJob{}
	err := fakeClient.Get(context.Background(), client.ObjectKeyFromObject(cronJob), stored)
	require.Error(t, err)
	assert.True(t, client.IgnoreNotFound(err) == nil)
}

func TestCronJobManagerDeleteReturnsError(t *testing.T) {
	scheme := newCronJobManagerTestScheme(t)
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithInterceptorFuncs(interceptor.Funcs{
			Delete: func(_ context.Context, _ client.WithWatch, _ client.Object, _ ...client.DeleteOption) error {
				return assert.AnError
			},
		}).
		Build()
	manager := defaultCronJobManager{Client: fakeClient, scheme: scheme}

	err := manager.delete(context.Background(), newCronJobManagerTestSchedule())

	require.Error(t, err)
	assert.ErrorIs(t, err, assert.AnError)
	assert.ErrorContains(t, err, "failed to delete CronJob")
}

func newCronJobManagerTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()

	scheme := runtime.NewScheme()
	require.NoError(t, backupv1.AddToScheme(scheme))
	require.NoError(t, batchv1.AddToScheme(scheme))
	return scheme
}

func newCronJobManagerTestSchedule() *backupv1.BackupSchedule {
	return &backupv1.BackupSchedule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "daily",
			Namespace: "default",
			UID:       "schedule-uid",
		},
		Spec: backupv1.BackupScheduleSpec{
			Schedule: "0 2 * * *",
			Provider: backupv1.Provider("velero"),
		},
	}
}
