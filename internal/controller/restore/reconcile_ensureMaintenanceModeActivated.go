package restore

import (
	"context"

	k8sv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/k8s-registry-lib/repository"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ensureMaintenanceModeActivated tries to activate the maintenance notice before starting the restore
// this is optional, therefore this stage will always complete successfully
func (r *restoreReconciler) ensureMaintenanceModeActivated(
	ctx context.Context,
	restore *k8sv1.Restore,
) (*k8sv1.Restore, stageOutcome) {

	_, status, err := r.maintenanceModeSwitch.GetStatus(ctx)
	if !status {
		err = r.maintenanceModeSwitch.Activate(ctx, repository.MaintenanceModeDescription{Title: maintenanceModeTitle, Text: maintenanceModeText}, false)
		if err != nil {
			logger := log.FromContext(ctx)
			logger.Error(err, "The Maintenance mode could not be activated. Continuing anyways...")
			r.recorder.Eventf(restore, corev1.EventTypeNormal, ReasonMaintenanceModeActivated, "Could not activate maintenance mode; continuing restore.")
		}
	}

	return restore, next()
}
