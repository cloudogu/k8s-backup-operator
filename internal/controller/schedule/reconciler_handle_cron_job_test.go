package schedule

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func Test_reconciler_createCronJob(t *testing.T) {
	ctx := context.Background()
	logger := log.FromContext(ctx)

	t.Run("should create CronJob with schedule based on the BackupSchedule", func(t *testing.T) {
		// given
		schedule := createTestBackupSchedule()
		schedule.Spec.Schedule = "0 0 * * *" // different schedule
		scheme := createScheme(t)
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
		reconciler := &defaultReconciler{client: fakeClient}

		// when
		err := reconciler.createCronJob(ctx, schedule, testNamespace, logger)

		// then
		require.NoError(t, err)

		createdCronJob := &batchv1.CronJob{}
		err = fakeClient.Get(ctx, types.NamespacedName{
			Name:      schedule.CronJobName(),
			Namespace: testNamespace,
		}, createdCronJob)

		require.NoError(t, err)
		assert.Equal(t, "0 0 * * *", createdCronJob.Spec.Schedule)
	})

	t.Run("should return error when CronJob already exists", func(t *testing.T) {
		// given
		schedule := createTestBackupSchedule()
		scheme := createScheme(t)

		// Create existing CronJob
		existingCronJob := createTestCronJob(testScheduleName, testScheduleCron)
		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(existingCronJob).
			Build()
		reconciler := &defaultReconciler{client: fakeClient}

		// when
		err := reconciler.createCronJob(ctx, schedule, testNamespace, logger)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create CronJob")
		assert.True(t, errors.IsAlreadyExists(err))
	})

}
