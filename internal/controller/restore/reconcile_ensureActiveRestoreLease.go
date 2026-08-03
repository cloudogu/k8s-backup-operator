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
	if leaseHolderUID(lease) == restore.UID {
		return restore, next()
	}

	// otherwise the lease is binded to another restore-CR therefor
	// the restore must wati until the lease is freed again

	// check if the current lease ist stale (orphaned lease)
	stale, err := r.isStaleRestoreLease(ctx, lease)
	if err != nil {
		return restore, retryOnError(err)
	}
	if stale {
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

// isStaleRestoreLease checks the Kubernetes object identity rather than a timeout. Unknown legacy
// holders are kept instead of being stolen because safety is more important than automatic progress.
func (r *restoreReconciler) isStaleRestoreLease(ctx context.Context, lease *coordinationv1.Lease) (bool, error) {
	holderUID := leaseHolderUID(lease)
	holderName := lease.Annotations[restoreLeaseHolderNameAnnotation]
	// if the lease does not have valid information, the cr might be corrupted
	// but we can not determine if the lease is really orphaned or in use
	if holderUID == "" || holderName == "" {
		return false, nil
	}

	//
	holder := &k8sv1.Restore{}
	err := r.k8sClient.Get(ctx, client.ObjectKey{Namespace: lease.Namespace, Name: holderName}, holder)
	// no restore for this lease
	if apierrors.IsNotFound(err) {
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to verify holder %s of restore lease %s/%s: %w", holderName, lease.Namespace, lease.Name, err)
	}

	// if the current restore-holder is in terminal state, the lease can be reused
	return holder.UID != holderUID || isTerminal(holder), nil
}
