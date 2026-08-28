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
		recorder := newFakeEventRecorder()
		manager := defaultCronJobManager{
			Client:           fakeClient,
			recorder:         recorder,
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
		// owner reference is backup schedule
		require.Len(t, stored.OwnerReferences, 1)
		assert.Equal(t, schedule.Name, stored.OwnerReferences[0].Name)
		assert.True(t, *stored.OwnerReferences[0].Controller)
		requireRecordedEvent(t, recorder, schedule, corev1.EventTypeNormal, backupv1.CronJobCreatedEventReason, `Created CronJob "backup-schedule-daily".`)

		// A converged CronJob must not produce another lifecycle event.
		require.NoError(t, manager.ensure(context.Background(), schedule))
		requireNoRecordedEvent(t, recorder)
	})
	t.Run("ensure should sync an existing cronjob when backupschedule is changed", func(t *testing.T) {
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
		recorder := newFakeEventRecorder()
		manager := defaultCronJobManager{
			Client:        fakeClient,
			recorder:      recorder,
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
		requireRecordedEvent(t, recorder, schedule, corev1.EventTypeNormal, backupv1.CronJobUpdatedEventReason, `Updated CronJob "backup-schedule-daily".`)
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
		recorder := newFakeEventRecorder()
		manager := defaultCronJobManager{Client: fakeClient, recorder: recorder, scheme: scheme}
		schedule := newCronJobManagerTestSchedule()

		err := manager.ensure(context.Background(), schedule)

		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to synchronize CronJob")
		requireRecordedEventContains(t, recorder, schedule, corev1.EventTypeWarning, backupv1.CronJobSynchronizationFailedEventReason, `Failed to synchronize CronJob "backup-schedule-daily"`, assert.AnError.Error())
	})

	t.Run("ensure should fail if owner reference can't be set", func(t *testing.T) {
		scheme := runtime.NewScheme()
		require.NoError(t, batchv1.AddToScheme(scheme))
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
		recorder := newFakeEventRecorder()
		manager := defaultCronJobManager{Client: fakeClient, recorder: recorder, scheme: scheme}
		schedule := newCronJobManagerTestSchedule()

		err := manager.ensure(context.Background(), schedule)

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to synchronize CronJob")
		assert.ErrorContains(t, err, "failed to set BackupSchedule daily as owner")
		requireRecordedEventContains(t, recorder, schedule, corev1.EventTypeWarning, backupv1.CronJobSynchronizationFailedEventReason, `Failed to synchronize CronJob "backup-schedule-daily"`, "failed to set BackupSchedule daily as owner")
	})
}

func TestCronJobManagerEnsureDoesNotUpdateForAPIServerDefaults(t *testing.T) {
	scheme := newCronJobManagerTestScheme(t)
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	recorder := newFakeEventRecorder()
	manager := defaultCronJobManager{
		Client:        fakeClient,
		recorder:      recorder,
		scheme:        scheme,
		operatorImage: "example.com/backup-operator:1.2.3",
		pullPolicy:    corev1.PullAlways,
	}
	schedule := newCronJobManagerTestSchedule()

	require.NoError(t, manager.ensure(context.Background(), schedule))
	requireRecordedEvent(t, recorder, schedule, corev1.EventTypeNormal, backupv1.CronJobCreatedEventReason, `Created CronJob "backup-schedule-daily".`)

	stored := &batchv1.CronJob{}
	require.NoError(t, fakeClient.Get(context.Background(), client.ObjectKey{
		Name: schedule.CronJobName(), Namespace: schedule.Namespace,
	}, stored))
	setAPIServerPodDefaults(&stored.Spec.JobTemplate.Spec.Template)
	require.NoError(t, fakeClient.Update(context.Background(), stored))

	require.NoError(t, manager.ensure(context.Background(), schedule))
	requireNoRecordedEvent(t, recorder)
}

func TestManagedTemplateEqual(t *testing.T) {
	schedule := newCronJobManagerTestSchedule()
	desired := schedule.CronJobPodTemplate("example.com/backup-operator:1.2.3", corev1.PullAlways)
	desired.Spec.ImagePullSecrets = []corev1.LocalObjectReference{{Name: "registry-secret"}}

	tests := []struct {
		name     string
		mutate   func(*corev1.PodTemplateSpec)
		expected bool
	}{
		{
			name:     "identical templates",
			mutate:   func(*corev1.PodTemplateSpec) {},
			expected: true,
		},
		{
			name: "API server defaults are ignored",
			mutate: func(actual *corev1.PodTemplateSpec) {
				setAPIServerPodDefaults(actual)
			},
			expected: true,
		},
		{
			name: "template name differs",
			mutate: func(actual *corev1.PodTemplateSpec) {
				actual.Name = "different"
			},
		},
		{
			name: "template namespace differs",
			mutate: func(actual *corev1.PodTemplateSpec) {
				actual.Namespace = "different"
			},
		},
		{
			name: "template labels differ",
			mutate: func(actual *corev1.PodTemplateSpec) {
				actual.Labels["app"] = "different"
			},
		},
		{
			name: "image pull secrets differ",
			mutate: func(actual *corev1.PodTemplateSpec) {
				actual.Spec.ImagePullSecrets[0].Name = "different"
			},
		},
		{
			name: "restart policy differs",
			mutate: func(actual *corev1.PodTemplateSpec) {
				actual.Spec.RestartPolicy = corev1.RestartPolicyNever
			},
		},
		{
			name: "service account differs",
			mutate: func(actual *corev1.PodTemplateSpec) {
				actual.Spec.ServiceAccountName = "different"
			},
		},
		{
			name: "container count differs",
			mutate: func(actual *corev1.PodTemplateSpec) {
				actual.Spec.Containers = append(actual.Spec.Containers, corev1.Container{Name: "sidecar"})
			},
		},
		{
			name: "container name differs",
			mutate: func(actual *corev1.PodTemplateSpec) {
				actual.Spec.Containers[0].Name = "different"
			},
		},
		{
			name: "container image differs",
			mutate: func(actual *corev1.PodTemplateSpec) {
				actual.Spec.Containers[0].Image = "different:latest"
			},
		},
		{
			name: "container pull policy differs",
			mutate: func(actual *corev1.PodTemplateSpec) {
				actual.Spec.Containers[0].ImagePullPolicy = corev1.PullNever
			},
		},
		{
			name: "container arguments differ",
			mutate: func(actual *corev1.PodTemplateSpec) {
				actual.Spec.Containers[0].Args = []string{"different"}
			},
		},
		{
			name: "container environment differs",
			mutate: func(actual *corev1.PodTemplateSpec) {
				actual.Spec.Containers[0].Env = []corev1.EnvVar{{Name: "DIFFERENT", Value: "true"}}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := desired.DeepCopy()
			tt.mutate(actual)

			assert.Equal(t, tt.expected, managedTemplateEqual(*actual, desired))
		})
	}
}

// this simulates the Kubernetes API setting certain defaults in the
// pod template after it has been created. This must not lead to a
// second reconcilation or we could get stuck in a loop
func setAPIServerPodDefaults(template *corev1.PodTemplateSpec) {
	terminationGracePeriodSeconds := int64(30)
	enableServiceLinks := true
	template.Spec.DNSPolicy = corev1.DNSClusterFirst
	template.Spec.SchedulerName = corev1.DefaultSchedulerName
	template.Spec.SecurityContext = &corev1.PodSecurityContext{}
	template.Spec.TerminationGracePeriodSeconds = &terminationGracePeriodSeconds
	template.Spec.EnableServiceLinks = &enableServiceLinks
	for i := range template.Spec.Containers {
		template.Spec.Containers[i].TerminationMessagePath = corev1.TerminationMessagePathDefault
		template.Spec.Containers[i].TerminationMessagePolicy = corev1.TerminationMessageReadFile
	}
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
