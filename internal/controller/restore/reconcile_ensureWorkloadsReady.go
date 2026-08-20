package restore

import (
	"context"
	"fmt"

	k8sv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/k8s-backup-operator/internal/logging"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ensureWorkloadsReady observes the workloads after scale-up without blocking a controller worker.
// A workload that is still converging causes a controlled retry, while observation errors use the
// controller-runtime backoff. Readiness is checked again even after its reason was persisted, so a
// workload that regressed cannot let the workflow proceed.
func (r *restoreReconciler) ensureWorkloadsReady(
	ctx context.Context,
	restore *k8sv1.Restore,
) (*k8sv1.Restore, stageOutcome) {
	ready, err := r.scaleManager.AreWorkloadsReady(ctx)
	if err != nil {
		return restore, retryOnError(fmt.Errorf("failed to observe workloads after scale-up: %w", err))
	}

	condition := meta.FindStatusCondition(restore.Status.Conditions, k8sv1.ConditionWorkloadsRecovered)
	if !ready {
		if condition != nil &&
			condition.Status == metav1.ConditionUnknown &&
			condition.Reason == ReasonWaitingForWorkloads {
			logging.Debug(ctx, "Retrying restore reconciliation", "reason", "the workloads have not reached their target state yet")
			return restore, retryAfter(defaultRequeueDelay)
		}

		updated, updateErr := newConditionUpdater(r.k8sClient).setConditions(ctx, restore, metav1.Condition{
			Type:    k8sv1.ConditionWorkloadsRecovered,
			Status:  metav1.ConditionUnknown,
			Reason:  ReasonWaitingForWorkloads,
			Message: "The workloads have not reached their target replica count and availability yet.",
		})
		if updateErr != nil {
			return restore, retryOnError(fmt.Errorf(
				"failed to persist workload readiness wait for restore %s: %w",
				restore.Name,
				updateErr,
			))
		}

		logging.Debug(ctx, "Retrying restore reconciliation", "reason", "the workload readiness wait was persisted")
		return updated, retryAfter(defaultRequeueDelay)
	}

	if condition != nil &&
		condition.Status == metav1.ConditionUnknown &&
		(condition.Reason == ReasonWorkloadsReady ||
			condition.Reason == ReasonScaleUpFinalized ||
			condition.Reason == ReasonMaintenanceModeDeactivated) {
		return restore, next()
	}

	updated, err := newConditionUpdater(r.k8sClient).setConditions(ctx, restore, metav1.Condition{
		Type:    k8sv1.ConditionWorkloadsRecovered,
		Status:  metav1.ConditionUnknown,
		Reason:  ReasonWorkloadsReady,
		Message: "All workloads reached their target replica count and availability.",
	})
	if err != nil {
		return restore, retryOnError(fmt.Errorf(
			"failed to persist workload readiness for restore %s: %w",
			restore.Name,
			err,
		))
	}

	logging.Debug(ctx, "Retrying restore reconciliation", "reason", "the workload readiness was persisted")
	return updated, retryAfter(defaultRequeueDelay)
}
