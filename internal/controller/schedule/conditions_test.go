package schedule

import (
	"errors"
	"testing"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestMarkAccepted(t *testing.T) {
	manager := defaultConditionManager{}
	schedule := newBackupSchedule()

	manager.markAccepted(schedule)

	condition := getCondition(t, schedule, backupv1.ConditionAccepted)
	assert.Equal(t, metav1.ConditionTrue, condition.Status)
	assert.Equal(t, backupv1.ReasonValidSpec, condition.Reason)
	assert.Equal(t, "Backup schedule accepted", condition.Message)
}

func TestMarkInvalid(t *testing.T) {
	manager := defaultConditionManager{}
	schedule := newBackupSchedule()

	manager.markInvalid(schedule, errors.New("invalid cron expression"))

	condition := getCondition(t, schedule, backupv1.ConditionAccepted)
	assert.Equal(t, metav1.ConditionFalse, condition.Status)
	assert.Equal(t, backupv1.ReasonInvalidSpec, condition.Reason)
	assert.Equal(t, "invalid cron expression", condition.Message)
}

func TestMarkAcceptanceNotEvaluated(t *testing.T) {
	manager := defaultConditionManager{}
	schedule := newBackupSchedule()

	manager.markAcceptanceNotEvaluated(schedule, errors.New("metadata failed"))

	condition := getCondition(t, schedule, backupv1.ConditionAccepted)
	assert.Equal(t, metav1.ConditionUnknown, condition.Status)
	assert.Equal(t, backupv1.ReasonNotEvaluated, condition.Reason)
	assert.Contains(t, condition.Message, "metadata failed")
}

func TestMarkReady(t *testing.T) {
	manager := defaultConditionManager{}
	schedule := newBackupSchedule()

	manager.markReady(schedule)

	condition := getCondition(t, schedule, backupv1.ConditionReady)
	assert.Equal(t, metav1.ConditionTrue, condition.Status)
	assert.Equal(t, backupv1.ReasonReady, condition.Reason)
	assert.Equal(t, "BackupSchedule is ready.", condition.Message)
}

func TestMarkNotReady(t *testing.T) {
	manager := defaultConditionManager{}
	schedule := newBackupSchedule()

	manager.markNotReady(schedule, backupv1.ReasonSyncFailed, "CronJob synchronization failed")

	condition := getCondition(t, schedule, backupv1.ConditionReady)
	assert.Equal(t, metav1.ConditionFalse, condition.Status)
	assert.Equal(t, backupv1.ReasonSyncFailed, condition.Reason)
	assert.Equal(t, "CronJob synchronization failed", condition.Message)
}

func newBackupSchedule() *backupv1.BackupSchedule {
	return &backupv1.BackupSchedule{}
}

func getCondition(t *testing.T, schedule *backupv1.BackupSchedule, conditionName string) *metav1.Condition {
	t.Helper()

	condition := meta.FindStatusCondition(schedule.Status.Conditions, conditionName)
	require.NotNil(t, condition)
	return condition
}
