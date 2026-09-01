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

// ensureScaleUpFinalized removes the temporary replica labels after workload readiness was
// observed.
// FinalizeScaleUp runs before the condition check so retries finish removing
// replica labels before the finalized state is persisted in the Restore status.
func (r *restoreReconciler) ensureScaleUpFinalized(
	ctx context.Context,
	restore *k8sv1.Restore,
) (*k8sv1.Restore, stageOutcome) {
	if err := r.scaleManager.FinalizeScaleUp(ctx); err != nil {
		r.recorder.Eventf(restore, nil, corev1.EventTypeWarning, ReasonWorkloadRecoveryFailed, actionRecoverWorkloads, "failed to finalize workload scale-up after restore")
		return r.reportUnreachedMilestone(
			ctx,
			restore,
			k8sv1.ConditionWorkloadsRecovered,
			ReasonWorkloadRecoveryFailed,
			fmt.Errorf("failed to finalize workload scale-up after restore: %w", err),
		)
	}

	condition := meta.FindStatusCondition(restore.Status.Conditions, k8sv1.ConditionWorkloadsRecovered)
	if condition != nil &&
		condition.Status == metav1.ConditionUnknown &&
		(condition.Reason == ReasonScaleUpFinalized ||
			condition.Reason == ReasonMaintenanceModeDeactivated) {
		return restore, next()
	}

	updated, err := newConditionUpdater(r.k8sClient).setConditions(ctx, restore, metav1.Condition{
		Type:    k8sv1.ConditionWorkloadsRecovered,
		Status:  metav1.ConditionUnknown,
		Reason:  ReasonScaleUpFinalized,
		Message: "The temporary replica recovery labels were removed from all workloads.",
	})
	if err != nil {
		return restore, retryOnError(fmt.Errorf(
			"failed to persist finalized workload scale-up for restore %s: %w",
			restore.Name,
			err,
		))
	}

	logging.Debug(ctx, "Retrying restore reconciliation", "reason", "the finalized workload scale-up was persisted")
	return updated, retryAfter(defaultRequeueDelay)
}
