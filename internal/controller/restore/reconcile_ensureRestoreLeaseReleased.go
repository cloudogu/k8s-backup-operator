package restore

import (
	"context"
	"fmt"

	k8sv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/k8s-backup-operator/internal/leases"
)

// ensureRestoreLeaseReleased frees the shared operation lease after the Restore became terminal.
// The lease manager verifies holder kind, name and UID, so this stage never releases another
// backup or restore operation.
func (r *restoreReconciler) ensureRestoreLeaseReleased(
	ctx context.Context,
	restore *k8sv1.Restore,
) (*k8sv1.Restore, stageOutcome) {
	manager := leases.NewManager(r.k8sClient, r.namespace, restoreLeaseName, restoreHolderResolver{client: r.k8sClient})
	if _, err := manager.Release(ctx, restore, restoreLeaseHolderKind); err != nil {
		return restore, retryOnError(fmt.Errorf("failed to release restore lease for restore %s: %w", restore.Name, err))
	}
	return restore, next()
}
