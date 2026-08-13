package schedule

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	batchv1 "k8s.io/api/batch/v1"
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
