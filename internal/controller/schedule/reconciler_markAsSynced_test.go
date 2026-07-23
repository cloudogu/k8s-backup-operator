package schedule

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_defaultReconciler_markAsSyncedToCronJob(t *testing.T) {
	t.Run("should set synced condition to true", func(t *testing.T) {
		// given
		backupSchedule := createTestBackupSchedule()
		sut := &defaultReconciler{}

		// when
		err := sut.markAsSyncedToCronJob(backupSchedule)

		// then
		require.NoError(t, err)
		require.Len(t, backupSchedule.Status.Conditions, 1)

		condition := backupSchedule.Status.Conditions[0]
		assert.Equal(t, ConditionTypeCronJobSynced, condition.Type)
		assert.Equal(t, metav1.ConditionTrue, condition.Status)
		assert.Equal(t, ReasonCronJobSyncSuccessful, condition.Reason)
		assert.Equal(t, "Cron job synced to backup schedule resource", condition.Message)
	})

	t.Run("should update existing synced condition", func(t *testing.T) {
		// given
		backupSchedule := createTestBackupSchedule()
		// First mark as not synced
		_ = (&defaultReconciler{}).markAsNotSyncedToCronJob(backupSchedule)

		sut := &defaultReconciler{}

		// when
		err := sut.markAsSyncedToCronJob(backupSchedule)

		// then
		require.NoError(t, err)
		require.Len(t, backupSchedule.Status.Conditions, 1)

		condition := backupSchedule.Status.Conditions[0]
		assert.Equal(t, ConditionTypeCronJobSynced, condition.Type)
		assert.Equal(t, metav1.ConditionTrue, condition.Status)
		assert.Equal(t, ReasonCronJobSyncSuccessful, condition.Reason)
	})
}

func Test_defaultReconciler_markAsNotSyncedToCronJob(t *testing.T) {
	t.Run("should set synced condition to false", func(t *testing.T) {
		// given
		backupSchedule := createTestBackupSchedule()
		sut := &defaultReconciler{}

		// when
		err := sut.markAsNotSyncedToCronJob(backupSchedule)

		// then
		require.NoError(t, err)
		require.Len(t, backupSchedule.Status.Conditions, 1)

		condition := backupSchedule.Status.Conditions[0]
		assert.Equal(t, ConditionTypeCronJobSynced, condition.Type)
		assert.Equal(t, metav1.ConditionFalse, condition.Status)
		assert.Equal(t, ReasonCronJobSyncFailed, condition.Reason)
		assert.Equal(t, "Could not sync cronjob to backup schedule", condition.Message)
	})

	t.Run("should update existing synced condition", func(t *testing.T) {
		// given
		backupSchedule := createTestBackupSchedule()
		// First mark as synced
		_ = (&defaultReconciler{}).markAsSyncedToCronJob(backupSchedule)

		sut := &defaultReconciler{}

		// when
		err := sut.markAsNotSyncedToCronJob(backupSchedule)

		// then
		require.NoError(t, err)
		require.Len(t, backupSchedule.Status.Conditions, 1)

		condition := backupSchedule.Status.Conditions[0]
		assert.Equal(t, ConditionTypeCronJobSynced, condition.Type)
		assert.Equal(t, metav1.ConditionFalse, condition.Status)
		assert.Equal(t, ReasonCronJobSyncFailed, condition.Reason)
	})
}
