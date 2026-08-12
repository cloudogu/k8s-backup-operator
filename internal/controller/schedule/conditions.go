package schedule

import (
	v1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type conditionManager struct{}

func (c conditionManager) MarkAccepted(schedule *v1.BackupSchedule) {
	setCondition(schedule, metav1.Condition{
		Type:    v1.ConditionAccepted,
		Status:  metav1.ConditionTrue,
		Reason:  v1.ReasonValidSpec,
		Message: "Backup schedule accepted",
	})
}

func (c conditionManager) MarkInvalid(schedule *v1.BackupSchedule, err error) {
	setCondition(schedule, metav1.Condition{
		Type:    v1.ConditionAccepted,
		Status:  metav1.ConditionFalse,
		Reason:  v1.ReasonInvalidSpec,
		Message: err.Error(),
	})
}

func (c conditionManager) MarkAcceptanceNotEvaluated(schedule *v1.BackupSchedule, err error) {
	setCondition(schedule, metav1.Condition{
		Type:    v1.ConditionAccepted,
		Status:  metav1.ConditionUnknown,
		Reason:  v1.ReasonNotEvaluated,
		Message: "Required metadata could not be persisted: " + err.Error(),
	})
}

func (c conditionManager) MarkReady(schedule *v1.BackupSchedule) {
	setCondition(schedule, metav1.Condition{
		Type:    v1.ConditionReady,
		Status:  metav1.ConditionTrue,
		Reason:  v1.ReasonReady,
		Message: "BackupSchedule is ready.",
	})
}

func (c conditionManager) MarkNotReady(schedule *v1.BackupSchedule, reason, message string) {
	setCondition(schedule, metav1.Condition{
		Type:    v1.ConditionReady,
		Status:  metav1.ConditionFalse,
		Reason:  reason,
		Message: message,
	})
}

func setCondition(schedule *v1.BackupSchedule, condition metav1.Condition) {
	meta.SetStatusCondition(&schedule.Status.Conditions, condition)
}
