package restore

import (
	"context"
	"fmt"

	k8sv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/k8s-backup-operator/internal/leases"
	"github.com/cloudogu/k8s-backup-operator/internal/metrics"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	restoreLeaseName                 = leases.DefaultName
	restoreLeaseHolderNameAnnotation = leases.HolderNameAnnotation
	restoreLeaseHolderKind           = "Restore"
)

// ensureActiveRestoreLease remains a Restore workflow stage while delegating the generic lease
// acquisition, holder validation, resolution and takeover mechanics to internal/leases.
func (r *restoreReconciler) ensureActiveRestoreLease(ctx context.Context, restore *k8sv1.Restore) (*k8sv1.Restore, stageOutcome) {
	if requiredOperation(restore) != operationCreate {
		return restore, next()
	}

	manager := leases.NewManager(r.k8sClient, r.namespace, restoreLeaseName, restoreHolderResolver{client: r.k8sClient})
	result, err := manager.Acquire(ctx, restore, restoreLeaseHolderKind)
	if err != nil {
		return restore, retryOnError(fmt.Errorf("failed to acquire restore lease for restore %s: %w", restore.Name, err))
	}

	switch result.State {
	case leases.StateChanged:
		// A Lease write or an optimistic-lock conflict must be observed in a new reconciliation.
		return restore, retryAfter(defaultRequeueDelay)
	case leases.StateInvalid:
		return r.reportInvalidRestoreLease(ctx, restore)
	case leases.StateWaiting:
		return r.reportWaitingForLease(ctx, restore, result.HolderName)
	case leases.StateAcquired:
		return r.continueWithAcquiredLease(ctx, restore)
	default:
		return restore, retryOnError(fmt.Errorf("unknown restore lease acquisition state %d", result.State))
	}
}

func (r *restoreReconciler) continueWithAcquiredLease(ctx context.Context, restore *k8sv1.Restore) (*k8sv1.Restore, stageOutcome) {
	condition := findSuccessfulCondition(restore)
	if condition == nil || (condition.Reason != ReasonWaitingForActiveRestore && condition.Reason != ReasonInvalidRestoreLease) {
		return restore, next()
	}
	updated, err := newConditionUpdater(r.k8sClient).setConditions(ctx, restore, metav1.Condition{
		Type: k8sv1.ConditionSuccessful, Status: metav1.ConditionUnknown,
		Reason:  ReasonRestoreLeaseAcquired,
		Message: "The restore holds the namespace-wide restore lease and can continue.",
	})
	if err != nil {
		return restore, retryOnError(fmt.Errorf("failed to report lease acquisition for restore %s: %w", restore.Name, err))
	}
	return updated, retryAfter(defaultRequeueDelay)
}

func (r *restoreReconciler) reportWaitingForLease(ctx context.Context, restore *k8sv1.Restore, holderName string) (*k8sv1.Restore, stageOutcome) {
	message := "Another operation currently holds the namespace-wide backup/restore lease."
	if holderName != "" {
		message = fmt.Sprintf("Operation %q currently holds the namespace-wide backup/restore lease.", holderName)
	}
	updated, err := newConditionUpdater(r.k8sClient).setConditions(ctx, restore, metav1.Condition{
		Type: k8sv1.ConditionSuccessful, Status: metav1.ConditionUnknown,
		Reason: ReasonWaitingForActiveRestore, Message: message,
	})
	if err != nil {
		return restore, retryOnError(fmt.Errorf("failed to report that restore %s is waiting for the active operation: %w", restore.Name, err))
	}
	return updated, retryAfter(defaultRequeueDelay)
}

func (r *restoreReconciler) reportInvalidRestoreLease(ctx context.Context, restore *k8sv1.Restore) (*k8sv1.Restore, stageOutcome) {
	leaseError := fmt.Errorf("restore is blocked by invalid lease %s/%s without a resolvable holder", r.namespace, restoreLeaseName)
	// use metrics to notify administrators
	metrics.UpdateInvalidLeaseTotalMetric(restore.Namespace, restore.Name)

	updated, err := newConditionUpdater(r.k8sClient).setConditions(ctx, restore, metav1.Condition{
		Type: k8sv1.ConditionSuccessful, Status: metav1.ConditionUnknown,
		Reason:  ReasonInvalidRestoreLease,
		Message: fmt.Sprintf("%s. Delete the lease after verifying that no backup or restore is active.", leaseError.Error()),
	})
	if err != nil {
		return restore, retryOnError(fmt.Errorf("failed to report invalid restore lease for restore %s: %w", restore.Name, err))
	}
	return updated, retryOnError(leaseError)
}

type restoreHolderResolver struct {
	client client.Client
}

func (r restoreHolderResolver) Kind() string { return restoreLeaseHolderKind }

func (r restoreHolderResolver) Get(ctx context.Context, namespace, name string) (client.Object, error) {
	holder := &k8sv1.Restore{}
	if err := r.client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, holder); err != nil {
		return nil, err
	}
	return holder, nil
}

func (restoreHolderResolver) IsTerminal(holder client.Object) bool {
	restore, ok := holder.(*k8sv1.Restore)
	return ok && isTerminal(restore)
}
