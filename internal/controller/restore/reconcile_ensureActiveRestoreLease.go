package restore

import (
	"context"
	"fmt"

	k8sv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	coordinationv1 "k8s.io/api/coordination/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	restoreLeaseName                 = "k8s-backup-operator-restore"
	restoreLeaseHolderNameAnnotation = "k8s.cloudogu.com/restore-lease-holder-name"
)

type restoreLeaseState int

const (
	restoreLeaseActive restoreLeaseState = iota
	restoreLeaseStale
	restoreLeaseInvalid
	restoreLeaseRepaired
)

// ensureActiveRestoreLease serializes the namespace-wide restore workflow. Creating the
// well-known Lease is the atomic fast path. An existing Lease may only be reused by the Restore
// whose UID is stored as holderIdentity, or taken over when that exact Restore no longer exists or
// is terminal. A mere passage of time is deliberately not enough to steal the Lease: preparation
// and provider restore are destructive and may legitimately take longer than an arbitrary timeout.
func (r *restoreReconciler) ensureActiveRestoreLease(ctx context.Context, restore *k8sv1.Restore) (*k8sv1.Restore, stageOutcome) {
	if requiredOperation(restore) != operationCreate {
		return restore, next()
	}

	key := client.ObjectKey{Namespace: r.namespace, Name: restoreLeaseName}
	lease := &coordinationv1.Lease{}
	err := r.k8sClient.Get(ctx, key, lease)

	switch {
	case apierrors.IsNotFound(err):
		lease = newRestoreLease(restore)
		if cerr := r.k8sClient.Create(ctx, lease); cerr != nil {
			// Another Restore can win between Get and Create. This is normal contention, not an
			// operational error; the next reconciliation observes the winner.
			if apierrors.IsAlreadyExists(cerr) {
				return restore, retryAfter(defaultRequeueDelay)
			}

			return restore, retryOnError(fmt.Errorf("failed to create restore lease %s/%s: %w", key.Namespace, key.Name, cerr))
		}
		// Do not run a second mutating stage in this invocation. The persisted Lease is the
		// source of truth and the controlled retry also covers a lost watch event.
		return restore, retryAfter(defaultRequeueDelay)
	case err != nil:
		return restore, retryOnError(fmt.Errorf("failed to get restore lease %s/%s: %w", key.Namespace, key.Name, err))
	}

	// the Lease belongs to the current restore
	// so the restore can start
	if leaseHolderUID(lease) == restore.UID && lease.Annotations[restoreLeaseHolderNameAnnotation] == restore.Name {
		condition := findSuccessfulCondition(restore)
		if condition == nil || (condition.Reason != ReasonWaitingForActiveRestore && condition.Reason != ReasonInvalidRestoreLease) {
			return restore, next()
		}

		updated, err := newConditionUpdater(r.k8sClient).setConditions(ctx, restore, metav1.Condition{
			Type:    k8sv1.ConditionSuccessful,
			Status:  metav1.ConditionUnknown,
			Reason:  ReasonRestoreLeaseAcquired,
			Message: "The restore holds the namespace-wide restore lease and can continue.",
		})
		if err != nil {
			return restore, retryOnError(
				fmt.Errorf("failed to report lease acquisition for restore %s: %w", restore.Name, err),
			)
		}

		return updated, retryAfter(defaultRequeueDelay)
	}

	state, err := r.inspectRestoreLease(ctx, lease)
	if err != nil {
		return restore, retryOnError(err)
	}

	switch state {
	case restoreLeaseStale:
		claimRestoreLease(lease, restore)
		if err := r.k8sClient.Update(ctx, lease); err != nil {
			// resourceVersion makes takeover a compare-and-swap. A conflict means another
			// contender changed the Lease first; observe that result on the next pass.
			if apierrors.IsConflict(err) {
				return restore, retryAfter(defaultRequeueDelay)
			}

			return restore, retryOnError(fmt.Errorf("failed to take over stale restore lease %s/%s: %w", key.Namespace, key.Name, err))
		}

		return restore, retryAfter(defaultRequeueDelay)
	case restoreLeaseRepaired:
		if err := r.k8sClient.Update(ctx, lease); err != nil {
			if apierrors.IsConflict(err) {
				return restore, retryAfter(defaultRequeueDelay)
			}

			return restore, retryOnError(fmt.Errorf("failed to repair restore lease %s/%s: %w", key.Namespace, key.Name, err))
		}

		return restore, retryAfter(defaultRequeueDelay)
	case restoreLeaseInvalid:
		return r.reportInvalidRestoreLease(ctx, restore, lease)
	}

	holderName := lease.Annotations[restoreLeaseHolderNameAnnotation]
	message := "Another restore currently holds the namespace-wide restore lease."
	if holderName != "" {
		message = fmt.Sprintf("Restore %q currently holds the namespace-wide restore lease.", holderName)
	}

	updated, err := newConditionUpdater(r.k8sClient).setConditions(ctx, restore, metav1.Condition{
		Type:    k8sv1.ConditionSuccessful,
		Status:  metav1.ConditionUnknown,
		Reason:  ReasonWaitingForActiveRestore,
		Message: message,
	})
	if err != nil {
		return restore, retryOnError(fmt.Errorf("failed to report that restore %s is waiting for the active restore: %w", restore.Name, err))
	}

	return updated, retryAfter(defaultRequeueDelay)
}

func (r *restoreReconciler) reportInvalidRestoreLease(ctx context.Context, restore *k8sv1.Restore, lease *coordinationv1.Lease) (*k8sv1.Restore, stageOutcome) {
	leaseError := fmt.Errorf("restore is blocked by invalid lease %s/%s without holder identity or name", lease.Namespace, lease.Name)
	updated, err := newConditionUpdater(r.k8sClient).setConditions(ctx, restore, metav1.Condition{
		Type:    k8sv1.ConditionSuccessful,
		Status:  metav1.ConditionUnknown,
		Reason:  ReasonInvalidRestoreLease,
		Message: fmt.Sprintf("%s. Delete the lease after verifying that no restore is active.", leaseError.Error()),
	})
	if err != nil {
		return restore, retryOnError(fmt.Errorf("failed to report invalid restore lease for restore %s: %w", restore.Name, err))
	}

	return updated, retryOnError(leaseError)
}

func newRestoreLease(restore *k8sv1.Restore) *coordinationv1.Lease {
	lease := &coordinationv1.Lease{ObjectMeta: metav1.ObjectMeta{
		Name:      restoreLeaseName,
		Namespace: restore.Namespace,
	}}
	claimRestoreLease(lease, restore)

	return lease
}

func claimRestoreLease(lease *coordinationv1.Lease, restore *k8sv1.Restore) {
	if lease.Annotations == nil {
		lease.Annotations = map[string]string{}
	}

	now := metav1.NowMicro()
	lease.Annotations[restoreLeaseHolderNameAnnotation] = restore.Name
	lease.Spec.HolderIdentity = ptr.To(string(restore.UID))
	lease.Spec.AcquireTime = &now
	lease.Spec.RenewTime = &now
	lease.Spec.LeaseTransitions = ptr.To(ptr.Deref(lease.Spec.LeaseTransitions, 0) + 1)
}

func leaseHolderUID(lease *coordinationv1.Lease) types.UID {
	return types.UID(ptr.Deref(lease.Spec.HolderIdentity, ""))
}

// inspectRestoreLease checks the Kubernetes object identity rather than a timeout. Partial holder
// information is repaired when it identifies an active Restore unambiguously. A Lease without any
// holder information remains blocked because it cannot be taken over safely.
func (r *restoreReconciler) inspectRestoreLease(ctx context.Context, lease *coordinationv1.Lease) (restoreLeaseState, error) {
	holderUID := leaseHolderUID(lease)
	holderName := lease.Annotations[restoreLeaseHolderNameAnnotation]
	if holderUID == "" && holderName == "" {
		return restoreLeaseInvalid, nil
	}

	if holderName == "" {
		holder, err := r.findRestoreByUID(ctx, lease.Namespace, holderUID)
		if err != nil {
			return restoreLeaseActive, fmt.Errorf("failed to verify holder UID %s of restore lease %s/%s: %w", holderUID, lease.Namespace, lease.Name, err)
		}
		if holder == nil || isTerminal(holder) {
			return restoreLeaseStale, nil
		}

		if lease.Annotations == nil {
			lease.Annotations = map[string]string{}
		}
		lease.Annotations[restoreLeaseHolderNameAnnotation] = holder.Name

		return restoreLeaseRepaired, nil
	}

	holder := &k8sv1.Restore{}
	err := r.k8sClient.Get(ctx, client.ObjectKey{Namespace: lease.Namespace, Name: holderName}, holder)
	if apierrors.IsNotFound(err) {
		return restoreLeaseStale, nil
	}
	if err != nil {
		return restoreLeaseActive, fmt.Errorf("failed to verify holder %s of restore lease %s/%s: %w", holderName, lease.Namespace, lease.Name, err)
	}
	if isTerminal(holder) {
		return restoreLeaseStale, nil
	}

	if holderUID == "" {
		lease.Spec.HolderIdentity = ptr.To(string(holder.UID))

		return restoreLeaseRepaired, nil
	}

	if holder.UID != holderUID {
		return restoreLeaseStale, nil
	}

	return restoreLeaseActive, nil
}

func (r *restoreReconciler) findRestoreByUID(ctx context.Context, namespace string, uid types.UID) (*k8sv1.Restore, error) {
	restores := &k8sv1.RestoreList{}
	if err := r.k8sClient.List(ctx, restores, client.InNamespace(namespace)); err != nil {
		return nil, err
	}

	for i := range restores.Items {
		if restores.Items[i].UID == uid {
			return &restores.Items[i], nil
		}
	}

	return nil, nil
}
