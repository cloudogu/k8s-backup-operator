package restore

import (
	"context"
	"errors"
	"fmt"

	k8sv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/k8s-backup-operator/internal/logging"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ensureScaleUpInitiated idempotently restores the desired replica counts and records that
// workload recovery has started. ScaleUp is deliberately called before inspecting the condition:
// after a crash between a workload update and the status update, the next reconciliation must be
// able to finish a partially initiated scale-up.
func (r *restoreReconciler) ensureScaleUpInitiated(
	ctx context.Context,
	restore *k8sv1.Restore,
) (*k8sv1.Restore, stageOutcome) {
	if err := r.scaleManager.ScaleUp(ctx); err != nil {
		r.recorder.Eventf(restore, nil, corev1.EventTypeWarning, ReasonWorkloadRecoveryFailed, actionRecoverWorkloads, "failed to initiate workload scale-up after restore")
		scaleUpErr := fmt.Errorf("failed to initiate workload scale-up after restore: %w", err)
		r.recorder.Eventf(restore, nil, corev1.EventTypeWarning, k8sv1.ErrorOnCreateEventReason, actionRecoverWorkloads, scaleUpErr.Error())

		updated, updateErr := newConditionUpdater(r.k8sClient).setConditions(ctx, restore, metav1.Condition{
			Type:    k8sv1.ConditionWorkloadsRecovered,
			Status:  metav1.ConditionFalse,
			Reason:  ReasonWorkloadRecoveryFailed,
			Message: fmt.Sprintf("The workload scale-up could not be initiated and will be retried: %v", scaleUpErr),
		})
		if updateErr != nil {
			scaleUpErr = errors.Join(scaleUpErr, fmt.Errorf("failed to report the unsuccessful workload scale-up initiation for restore %s: %w", restore.Name, updateErr))
		}

		return updated, retryOnError(scaleUpErr)
	}

	condition := meta.FindStatusCondition(restore.Status.Conditions, k8sv1.ConditionWorkloadsRecovered)
	if condition != nil &&
		condition.Status == metav1.ConditionUnknown &&
		(condition.Reason == ReasonScaleUpInitiated ||
			condition.Reason == ReasonWaitingForWorkloads ||
			condition.Reason == ReasonWorkloadsReady ||
			condition.Reason == ReasonScaleUpFinalized ||
			condition.Reason == ReasonMaintenanceModeDeactivated) {
		return restore, next()
	}

	updated, err := newConditionUpdater(r.k8sClient).setConditions(ctx, restore, metav1.Condition{
		Type:    k8sv1.ConditionWorkloadsRecovered,
		Status:  metav1.ConditionUnknown,
		Reason:  ReasonScaleUpInitiated,
		Message: "The desired replica counts were restored. The controller is waiting for the workloads to become ready.",
	})
	if err != nil {
		return restore, retryOnError(fmt.Errorf(
			"failed to persist initiated workload scale-up for restore %s: %w",
			restore.Name,
			err,
		))
	}

	// The status write ends this reconciliation. The following reconciliation verifies ScaleUp
	// idempotently before the workflow proceeds to workload observation.
	logging.Info(ctx, "scaling up the workloads")
	logging.Debug(ctx, "Retrying restore reconciliation", "reason", "the initiated workload scale-up was persisted")
	return updated, retryAfter(defaultRequeueDelay)
}
