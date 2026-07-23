package schedule

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func Test_reconciler_checkCronJobSynced(t *testing.T) {
	ctx := context.Background()
	logger := log.Log.WithName("test")

	t.Run("should return true when matching cronjob exists", func(t *testing.T) {
		// given
		backupSchedule := createTestBackupSchedule()
		cronJob := createTestCronJob(testCronJobName, testScheduleCron)

		scheme := createScheme(t)
		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(cronJob).
			Build()

		reconciler := &defaultReconciler{client: fakeClient}

		// when
		isSynced, err := reconciler.checkCronJobSynced(ctx, backupSchedule, testNamespace, logger)

		// then
		require.NoError(t, err)
		assert.True(t, isSynced)
	})

	t.Run("should return false when cronjob with matching name but different schedule exists", func(t *testing.T) {
		// given
		backupSchedule := createTestBackupSchedule()
		cronJob := createTestCronJob(testCronJobName, "0 0 * * *") // same name, different schedule

		scheme := createScheme(t)
		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(cronJob).
			Build()

		reconciler := &defaultReconciler{client: fakeClient}

		// when
		isSynced, err := reconciler.checkCronJobSynced(ctx, backupSchedule, testNamespace, logger)

		// then
		require.Error(t, err)
		assert.False(t, isSynced)
		assert.Contains(t, err.Error(), "no matching cronjob found")
	})

	t.Run("should find matching cronjob if there are multiple cronjobs", func(t *testing.T) {
		// given
		backupSchedule := createTestBackupSchedule()
		cronJob1 := createTestCronJob("other-cronjob-1", "0 0 * * *")
		cronJob2 := createTestCronJob(testCronJobName, testScheduleCron)
		cronJob3 := createTestCronJob("other-cronjob-2", "0 12 * * *")

		scheme := createScheme(t)
		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(cronJob1, cronJob2, cronJob3).
			Build()

		reconciler := &defaultReconciler{client: fakeClient}

		// when
		isSynced, err := reconciler.checkCronJobSynced(ctx, backupSchedule, testNamespace, logger)

		// then
		require.NoError(t, err)
		assert.True(t, isSynced)
	})
}
