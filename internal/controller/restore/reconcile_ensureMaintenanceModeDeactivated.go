package restore

import (
	"context"
	"fmt"

	k8sv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/k8s-backup-operator/internal/logging"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ensureMaintenanceModeDeactivated removes the maintenance notice after every recovered workload
// became ready and its temporary replica label was cleaned up. Deactivate is called before checking
// the condition so a crash before the status update is recovered idempotently.
func (r *restoreReconciler) ensureMaintenanceModeDeactivated(
	ctx context.Context,
	restore *k8sv1.Restore,
) (*k8sv1.Restore, stageOutcome) {
	if err := r.maintenanceModeSwitch.Deactivate(ctx, false); err != nil {
		return restore, retryOnError(fmt.Errorf(
			"failed to deactivate maintenance mode after restore %s: %w",
			restore.Name,
			err,
		))
	}

	condition := meta.FindStatusCondition(restore.Status.Conditions, k8sv1.ConditionWorkloadsRecovered)
	if condition != nil &&
		condition.Status == metav1.ConditionUnknown &&
		condition.Reason == ReasonMaintenanceModeDeactivated {
		return restore, next()
	}

	updated, err := newConditionUpdater(r.k8sClient).setConditions(ctx, restore, metav1.Condition{
		Type:    k8sv1.ConditionWorkloadsRecovered,
		Status:  metav1.ConditionUnknown,
		Reason:  ReasonMaintenanceModeDeactivated,
		Message: "The maintenance mode was deactivated after workload recovery.",
	})
	if err != nil {
		return restore, retryOnError(fmt.Errorf(
			"failed to persist maintenance mode deactivation for restore %s: %w",
			restore.Name,
			err,
		))
	}

	logging.Debug(ctx, "Retrying restore reconciliation", "reason", "the maintenance mode deactivation was persisted")
	return updated, retryAfter(defaultRequeueDelay)
}
