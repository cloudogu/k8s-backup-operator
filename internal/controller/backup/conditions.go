package backup

import (
	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/k8s-backup-operator/internal/conditions"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// initialBackupConditions are the conditions every running backup carries, in the order a reader of
// the status should see them, with the value they hold before any stage observed them.
//
// The milestones are Unknown because the workflow has not decided them yet. Canceled and Deleting
// are False instead, because these are flags we know to be false at init.
var initialBackupConditions = []metav1.Condition{
	pendingMilestone(backupv1.ConditionSucceeded),
	pendingMilestone(backupv1.ConditionPrepared),
	pendingMilestone(backupv1.ConditionProviderSucceeded),
	{
		Type:    backupv1.ConditionCanceled,
		Status:  metav1.ConditionFalse,
		Reason:  conditions.ReasonPending,
		Message: "The backup has not been canceled.",
	},
	{
		Type:    backupv1.ConditionDeleting,
		Status:  metav1.ConditionFalse,
		Reason:  conditions.ReasonPending,
		Message: "The backup is not being deleted.",
	},
}

// pendingMilestone is a milestone of the backup workflow that has not been reached yet.
func pendingMilestone(conditionType string) metav1.Condition {
	return metav1.Condition{
		Type:    conditionType,
		Status:  metav1.ConditionUnknown,
		Reason:  conditions.ReasonPending,
		Message: "The backup workflow has not reached this milestone yet.",
	}
}
