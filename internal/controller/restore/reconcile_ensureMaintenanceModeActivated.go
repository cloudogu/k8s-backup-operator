package restore

import (
	"context"

	k8sv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/k8s-backup-operator/internal/logging"
	"github.com/cloudogu/k8s-registry-lib/repository"
	corev1 "k8s.io/api/core/v1"
)

// ensureMaintenanceModeActivated tries to activate the maintenance notice before starting the restore
// this is optional, therefore this stage will always complete successfully
func (r *restoreReconciler) ensureMaintenanceModeActivated(
	ctx context.Context,
	restore *k8sv1.Restore,
) (*k8sv1.Restore, stageOutcome) {

	_, isActive, err := r.maintenanceModeSwitch.GetStatus(ctx)
	if err != nil {
		logging.Error(ctx, err, "The maintenance mode status could not be determined.. Continuing anyways...")
		r.recorder.Eventf(restore, corev1.EventTypeNormal, ReasonMaintenanceModeActivated, "Could not get maintenance mode status; continuing restore.")
	}
	if !isActive {
		err = r.maintenanceModeSwitch.Activate(ctx, repository.MaintenanceModeDescription{Title: maintenanceModeTitle, Text: maintenanceModeText}, false)
		if err != nil {
			logging.Error(ctx, err, "The Maintenance mode could not be activated. Continuing anyways...")
			r.recorder.Eventf(restore, corev1.EventTypeNormal, ReasonMaintenanceModeActivated, "Could not activate maintenance mode; continuing restore.")
		} else {
			logging.Info(ctx, "activated maintenance mode")
		}
	}

	return restore, next()
}
