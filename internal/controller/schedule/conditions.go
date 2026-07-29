package schedule

import (
	v1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type conditionManager struct{}

func (c conditionManager) MarkAccepted(schedule *v1.BackupSchedule) {
	meta.SetStatusCondition(&schedule.Status.Conditions, metav1.Condition{
		Type:    AcceptedCondition,
		Status:  metav1.ConditionTrue,
		Reason:  "",
		Message: "Backup schedule accepted",
	})
}

func (c conditionManager) MarkInvalid(schedule *v1.BackupSchedule, err error) {
	meta.SetStatusCondition(&schedule.Status.Conditions, metav1.Condition{
		Type:    AcceptedCondition,
		Status:  metav1.ConditionFalse,
		Reason:  "",
		Message: err.Error(),
	})
}

func (c conditionManager) MarkCronJobSynced(schedule *v1.BackupSchedule) {
	meta.SetStatusCondition(&schedule.Status.Conditions, metav1.Condition{
		Type:    CronJobSyncedCondition,
		Status:  metav1.ConditionTrue,
		Reason:  "",
		Message: "CronJob synced",
	})
}

func (c conditionManager) MarkCronJobNotSynced(schedule *v1.BackupSchedule, err error) {
	meta.SetStatusCondition(&schedule.Status.Conditions, metav1.Condition{
		Type:    CronJobSyncedCondition,
		Status:  metav1.ConditionFalse,
		Reason:  "",
		Message: err.Error(),
	})
}

func (c conditionManager) MarkDeleting(schedule *v1.BackupSchedule) {
	meta.SetStatusCondition(&schedule.Status.Conditions, metav1.Condition{
		Type:    DeletingCondition,
		Status:  metav1.ConditionTrue,
		Reason:  "",
		Message: "Backup schedule is being deleted",
	})
}

func (c conditionManager) ComputeReady(schedule *v1.BackupSchedule) {
	accepted := isConditionTrue(schedule, AcceptedCondition)
	synced := isConditionTrue(schedule, CronJobSyncedCondition)
	deleting := isConditionTrue(schedule, DeletingCondition)

	switch {
	case deleting:
		meta.SetStatusCondition(&schedule.Status.Conditions, metav1.Condition{
			Type:    ReadyCondition,
			Status:  metav1.ConditionFalse,
			Reason:  ReasonDeleting,
			Message: "BackupSchedule is being deleted.",
		})

	case !accepted:
		meta.SetStatusCondition(&schedule.Status.Conditions, metav1.Condition{
			Type:    ReadyCondition,
			Status:  metav1.ConditionFalse,
			Reason:  ReasonInvalidSpec,
			Message: "BackupSchedule spec is invalid.",
		})

	case !synced:
		meta.SetStatusCondition(&schedule.Status.Conditions, metav1.Condition{
			Type:    ReadyCondition,
			Status:  metav1.ConditionFalse,
			Reason:  ReasonSyncFailed,
			Message: "CronJob is not synchronized.",
		})

	default:
		meta.SetStatusCondition(&schedule.Status.Conditions, metav1.Condition{
			Type:    ReadyCondition,
			Status:  metav1.ConditionTrue,
			Reason:  "",
			Message: "BackupSchedule is ready.",
		})
	}
}

func isConditionTrue(schedule *v1.BackupSchedule, conditionType string) bool {
	condition := meta.FindStatusCondition(schedule.Status.Conditions, conditionType)
	return condition != nil && condition.Status == metav1.ConditionTrue
}
