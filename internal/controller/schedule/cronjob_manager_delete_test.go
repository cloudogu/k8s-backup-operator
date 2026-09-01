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
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestCronJobManagerDelete(t *testing.T) {
	scheme := newCronJobManagerTestScheme(t)
	schedule := newCronJobManagerTestSchedule()
	cronJob := &batchv1.CronJob{ObjectMeta: metav1.ObjectMeta{
		Name: schedule.CronJobName(), Namespace: schedule.Namespace,
	}}
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cronJob).Build()
	recorder := newFakeEventRecorder()
	manager := defaultCronJobManager{Client: fakeClient, recorder: recorder, scheme: scheme}

	require.NoError(t, manager.delete(context.Background(), schedule))
	requireRecordedEvent(t, recorder, schedule, corev1.EventTypeNormal, backupv1.CronJobDeletionRequestedEventReason, actionDeleteCronJob, `Requested deletion of CronJob "backup-schedule-daily".`)
	// twice to check for idempotence
	require.NoError(t, manager.delete(context.Background(), schedule))
	requireNoRecordedEvent(t, recorder)

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
	recorder := newFakeEventRecorder()
	manager := defaultCronJobManager{Client: fakeClient, recorder: recorder, scheme: scheme}
	schedule := newCronJobManagerTestSchedule()

	err := manager.delete(context.Background(), schedule)

	require.Error(t, err)
	assert.ErrorIs(t, err, assert.AnError)
	assert.ErrorContains(t, err, "failed to delete CronJob")
	requireRecordedEventContains(t, recorder, schedule, corev1.EventTypeWarning, backupv1.CronJobDeletionFailedEventReason, actionDeleteCronJob, `Failed to delete CronJob "backup-schedule-daily"`, assert.AnError.Error())
}
