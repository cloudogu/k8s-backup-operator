package schedule

import (
	"errors"
	"testing"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestMarkAccepted(t *testing.T) {
	manager := conditionManager{}
	schedule := newBackupSchedule()

	manager.MarkAccepted(schedule)

	c := getCondition(t, schedule, AcceptedCondition)

	assert.Equal(t, metav1.ConditionTrue, c.Status)
	assert.Equal(t, "Backup schedule accepted", c.Message)
}

func TestMarkInvalid(t *testing.T) {
	manager := conditionManager{}
	schedule := newBackupSchedule()

	manager.MarkInvalid(schedule, errors.New("invalid cron expresssion"))

	c := getCondition(t, schedule, AcceptedCondition)

	assert.Equal(t, metav1.ConditionFalse, c.Status)
	assert.Equal(t, "invalid cron expresssion", c.Message)
}

func TestMarkCronJobSynced(t *testing.T) {
	manager := conditionManager{}
	schedule := newBackupSchedule()

	manager.MarkCronJobSynced(schedule)

	c := getCondition(t, schedule, CronJobSyncedCondition)

	assert.Equal(t, metav1.ConditionTrue, c.Status)
	assert.Equal(t, "CronJob synced", c.Message)
}

func TestMarkCronJobNotSynced(t *testing.T) {
	manager := conditionManager{}
	schedule := newBackupSchedule()

	manager.MarkCronJobNotSynced(schedule, errors.New("could not sync"))

	c := getCondition(t, schedule, CronJobSyncedCondition)

	assert.Equal(t, metav1.ConditionFalse, c.Status)
	assert.Equal(t, "could not sync", c.Message)
}

func TestMarkDeleting(t *testing.T) {
	manager := conditionManager{}
	schedule := newBackupSchedule()

	manager.MarkDeleting(schedule)

	c := getCondition(t, schedule, DeletingCondition)

	assert.Equal(t, metav1.ConditionTrue, c.Status)
	assert.Equal(t, "Backup schedule is being deleted", c.Message)
}

func TestComputeReady(t *testing.T) {
	tests := []struct {
		name           string
		accepted       bool
		synced         bool
		deleting       bool
		expectedReady  metav1.ConditionStatus
		expectedReason string
	}{
		{
			name:          "ready",
			accepted:      true,
			synced:        true,
			expectedReady: metav1.ConditionTrue,
		},
		{
			name:           "invalid",
			synced:         true,
			expectedReady:  metav1.ConditionFalse,
			expectedReason: ReasonInvalidSpec,
		},
		{
			name:           "not synced",
			accepted:       true,
			expectedReady:  metav1.ConditionFalse,
			expectedReason: ReasonSyncFailed,
		},
		{
			name:           "deleting",
			accepted:       true,
			synced:         true,
			deleting:       true,
			expectedReady:  metav1.ConditionFalse,
			expectedReason: ReasonDeleting,
		},
	}

	manager := conditionManager{}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			schedule := newBackupSchedule()

			if tt.accepted {
				manager.MarkAccepted(schedule)
			} else {
				manager.MarkInvalid(schedule, errors.New("boom"))
			}

			if tt.synced {
				manager.MarkCronJobSynced(schedule)
			} else {
				manager.MarkCronJobNotSynced(schedule, errors.New("boom"))
			}

			if tt.deleting {
				manager.MarkDeleting(schedule)
			}

			manager.ComputeReady(schedule)

			ready := getCondition(t, schedule, ReadyCondition)

			assert.Equal(t, tt.expectedReady, ready.Status)
			assert.Equal(t, tt.expectedReason, ready.Reason)
		})
	}
}

// --------------------------- helpers

func newBackupSchedule() *backupv1.BackupSchedule {
	return &backupv1.BackupSchedule{}
}

func getCondition(t *testing.T, schedule *backupv1.BackupSchedule, conditionName string) *metav1.Condition {
	t.Helper()

	condition := meta.FindStatusCondition(schedule.Status.Conditions, conditionName)

	if condition == nil {
		t.Fatalf("condition was nil, expected %s condition", conditionName)
	}

	return condition
}
