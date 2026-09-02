package restore

import (
	"context"

	k8sv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/k8s-backup-operator/internal/logging"
	"github.com/cloudogu/k8s-registry-lib/repository"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
)

// ensureMaintenanceModeActivated tries to activate the maintenance notice before starting the restore
// this is optional, therefore this stage will always complete successfully
func (r *restoreReconciler) ensureMaintenanceModeActivated(
	ctx context.Context,
	restore *k8sv1.Restore,
) (*k8sv1.Restore, stageOutcome) {
	if r.hasWorkflowDeactivatedMaintenanceMode(restore) {
		logging.Debug(ctx, "ensureMaintenanceModeActivated: the workflow already deactivated the maintenance mode -> NEXT")
		return restore, next()
	}

	_, isActive, err := r.maintenanceModeSwitch.GetStatus(ctx)
	if err != nil {
		logging.Error(ctx, err, "The maintenance mode status could not be determined.. Continuing anyways...")
		r.recorder.Eventf(restore, nil, corev1.EventTypeNormal, ReasonMaintenanceModeActivated, actionCheckMaintenanceMode, "Could not get maintenance mode status; continuing restore.")
	}
	if !isActive {
		err = r.maintenanceModeSwitch.Activate(ctx, repository.MaintenanceModeDescription{Title: maintenanceModeTitle, Text: maintenanceModeText}, false)
		if err != nil {
			logging.Error(ctx, err, "The Maintenance mode could not be activated. Continuing anyways...")
			r.recorder.Eventf(restore, nil, corev1.EventTypeNormal, ReasonMaintenanceModeActivated, actionActivateMaintenanceMode, "Could not activate maintenance mode; continuing restore.")
		} else {
			logging.Info(ctx, "activated maintenance mode")
			r.recorder.Eventf(restore, nil, corev1.EventTypeNormal, ReasonMaintenanceModeActivated, actionActivateMaintenanceMode, "Maintenance mode activated")
		}
	}

	return restore, next()
}

// hasWorkflowDeactivatedMaintenanceMode reports whether this restore already switched the
// maintenance mode off on purpose, which is the case once the workload recovery either reported the
// deactivation or completed.
func (r *restoreReconciler) hasWorkflowDeactivatedMaintenanceMode(restore *k8sv1.Restore) bool {
	if meta.IsStatusConditionTrue(restore.Status.Conditions, k8sv1.ConditionWorkloadsRecovered) {
		return true
	}

	recovery := meta.FindStatusCondition(restore.Status.Conditions, k8sv1.ConditionWorkloadsRecovered)
	return recovery != nil && recovery.Reason == ReasonMaintenanceModeDeactivated
}
