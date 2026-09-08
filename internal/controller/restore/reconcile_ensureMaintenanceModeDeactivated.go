package restore

import (
	"context"
	"fmt"

	k8sv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/k8s-backup-operator/internal/logging"
	corev1 "k8s.io/api/core/v1"
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
		r.recorder.Eventf(restore, nil, corev1.EventTypeWarning, ReasonMaintenanceModeDeactivated, actionDeactivateMaintenanceMode, "Failed to deactivate maintenance mode after restore")
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
		r.recorder.Eventf(restore, nil, corev1.EventTypeNormal, ReasonMaintenanceModeDeactivated, actionDeactivateMaintenanceMode, "Maintenance mode deactivated")
		return restore, next()

	}

	updated, err := newConditionUpdater(r.k8sClient).setConditions(ctx, restore, metav1.Condition{
		Type:    k8sv1.ConditionWorkloadsRecovered,
		Status:  metav1.ConditionUnknown,
		Reason:  ReasonMaintenanceModeDeactivated,
		Message: "The maintenance mode was deactivated after workload recovery.",
	})
	if err != nil {
		r.recorder.Eventf(restore, nil, corev1.EventTypeWarning, ReasonMaintenanceModeDeactivated, actionDeactivateMaintenanceMode, "Failed to persist maintenance mode deactivation for restore")
		return restore, retryOnError(fmt.Errorf(
			"failed to persist maintenance mode deactivation for restore %s: %w",
			restore.Name,
			err,
		))
	}

	logging.Info(ctx, "deactivated maintenance mode")
	logging.Debug(ctx, "Retrying restore reconciliation", "reason", "the maintenance mode deactivation was persisted")
	return updated, retryAfter(defaultRequeueDelay)
}
