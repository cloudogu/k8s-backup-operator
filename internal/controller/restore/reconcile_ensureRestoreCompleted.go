package restore

import (
	"context"
	"fmt"

	k8sv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/k8s-backup-operator/pkg/metrics"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ensureRestoreCompleted marks the restore successful only after workload readiness, replica-label
// cleanup and maintenance-mode deactivation were persisted. This terminal stage performs no
// external recovery action itself.
func (r *restoreReconciler) ensureRestoreCompleted(
	ctx context.Context,
	restore *k8sv1.Restore,
) (*k8sv1.Restore, stageOutcome) {
	if meta.IsStatusConditionTrue(restore.Status.Conditions, k8sv1.ConditionWorkloadsRecovered) &&
		meta.IsStatusConditionTrue(restore.Status.Conditions, k8sv1.ConditionSuccessful) {
		return restore, abort()
	}

	recovery := meta.FindStatusCondition(restore.Status.Conditions, k8sv1.ConditionWorkloadsRecovered)
	if recovery == nil ||
		recovery.Status != metav1.ConditionUnknown ||
		recovery.Reason != ReasonMaintenanceModeDeactivated {
		return restore, retryAfter(defaultRequeueDelay)
	}

	updated, err := newConditionUpdater(r.k8sClient).setConditions(ctx, restore,
		reachedMilestone(k8sv1.ConditionWorkloadsRecovered, ReasonWorkloadRecoveryCompleted,
			"The workloads reached their target state, their recovery labels were removed and the maintenance mode was switched off."),
		reachedMilestone(k8sv1.ConditionSuccessful, ReasonRestoreCompleted,
			"The restore workflow finished successfully."),
	)
	if err != nil {
		return restore, retryOnError(fmt.Errorf("failed to persist the completed restore %s: %w", restore.Name, err))
	}

	metrics.UpdateRestoreStatusMetrics(r.namespace, restore.Name, restore.Spec.BackupName, k8sv1.RestoreStatusCompleted) // NOSONAR -- legacy restore status compatibility
	r.recorder.Event(restore, corev1.EventTypeNormal, k8sv1.CreateEventReason, "Restore successful")

	return updated, abort()
}
