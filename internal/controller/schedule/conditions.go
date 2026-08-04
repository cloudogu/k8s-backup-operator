package schedule

import (
	v1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type conditionManager struct{}

func (c conditionManager) MarkAccepted(schedule *v1.BackupSchedule) {
	setCondition(schedule, metav1.Condition{
		Type:    AcceptedCondition,
		Status:  metav1.ConditionTrue,
		Reason:  ReasonValidSpec,
		Message: "Backup schedule accepted",
	})
}

func (c conditionManager) MarkInvalid(schedule *v1.BackupSchedule, err error) {
	setCondition(schedule, metav1.Condition{
		Type:    AcceptedCondition,
		Status:  metav1.ConditionFalse,
		Reason:  ReasonInvalidSpec,
		Message: err.Error(),
	})
}

func (c conditionManager) MarkAcceptanceNotEvaluated(schedule *v1.BackupSchedule, err error) {
	setCondition(schedule, metav1.Condition{
		Type:    AcceptedCondition,
		Status:  metav1.ConditionUnknown,
		Reason:  ReasonNotEvaluated,
		Message: "Required metadata could not be persisted: " + err.Error(),
	})
}

func (c conditionManager) MarkCronJobSynced(schedule *v1.BackupSchedule) {
	setCondition(schedule, metav1.Condition{
		Type:    CronJobSyncedCondition,
		Status:  metav1.ConditionTrue,
		Reason:  ReasonSynced,
		Message: "CronJob synced",
	})
}

func (c conditionManager) MarkCronJobNotSynced(schedule *v1.BackupSchedule, err error) {
	setCondition(schedule, metav1.Condition{
		Type:    CronJobSyncedCondition,
		Status:  metav1.ConditionFalse,
		Reason:  ReasonSyncFailed,
		Message: err.Error(),
	})
}

func (c conditionManager) MarkDeleting(schedule *v1.BackupSchedule) {
	setCondition(schedule, metav1.Condition{
		Type:    DeletingCondition,
		Status:  metav1.ConditionTrue,
		Reason:  ReasonDeleting,
		Message: "Backup schedule is being deleted",
	})
}

func (c conditionManager) ComputeReady(schedule *v1.BackupSchedule) {
	accepted := meta.FindStatusCondition(schedule.Status.Conditions, AcceptedCondition)
	synced := meta.FindStatusCondition(schedule.Status.Conditions, CronJobSyncedCondition)
	deleting := isConditionTrue(schedule, DeletingCondition)

	switch {
	case deleting:
		setCondition(schedule, metav1.Condition{
			Type:    ReadyCondition,
			Status:  metav1.ConditionFalse,
			Reason:  ReasonDeleting,
			Message: "BackupSchedule is being deleted.",
		})

	case accepted == nil || accepted.Status == metav1.ConditionUnknown:
		setCondition(schedule, metav1.Condition{
			Type: ReadyCondition, Status: metav1.ConditionUnknown, Reason: ReasonNotEvaluated,
			Message: "BackupSchedule spec has not been evaluated.",
		})

	case accepted.Status == metav1.ConditionFalse:
		setCondition(schedule, metav1.Condition{
			Type:    ReadyCondition,
			Status:  metav1.ConditionFalse,
			Reason:  ReasonInvalidSpec,
			Message: "BackupSchedule spec is invalid.",
		})

	case synced == nil || synced.Status == metav1.ConditionUnknown:
		setCondition(schedule, metav1.Condition{
			Type: ReadyCondition, Status: metav1.ConditionUnknown, Reason: ReasonNotEvaluated,
			Message: "CronJob synchronization has not been evaluated.",
		})

	case synced.Status == metav1.ConditionFalse:
		setCondition(schedule, metav1.Condition{
			Type:    ReadyCondition,
			Status:  metav1.ConditionFalse,
			Reason:  ReasonSyncFailed,
			Message: "CronJob is not synchronized.",
		})

	default:
		setCondition(schedule, metav1.Condition{
			Type:    ReadyCondition,
			Status:  metav1.ConditionTrue,
			Reason:  ReasonReady,
			Message: "BackupSchedule is ready.",
		})
	}
}

func setCondition(schedule *v1.BackupSchedule, condition metav1.Condition) {
	meta.SetStatusCondition(&schedule.Status.Conditions, condition)
}

func isConditionTrue(schedule *v1.BackupSchedule, conditionType string) bool {
	condition := meta.FindStatusCondition(schedule.Status.Conditions, conditionType)
	return condition != nil && condition.Status == metav1.ConditionTrue
}
