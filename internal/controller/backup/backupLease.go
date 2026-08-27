package backup

import (
	"context"
	"fmt"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/k8s-backup-operator/internal/leases"
	"github.com/cloudogu/k8s-backup-operator/internal/logging"
	"github.com/cloudogu/k8s-backup-operator/internal/metrics"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const backupLeaseHolderKind = "Backup"

func (c *defaultReconciler) ensureActiveBackupLease(ctx context.Context, backup *backupv1.Backup) (action, error) {
	manager := leases.NewManager(c.client, backup.Namespace, leases.DefaultName, backupHolderResolver{client: c.client})
	result, err := manager.Acquire(ctx, backup, backupLeaseHolderKind)
	return c.backupLeaseAction(ctx, backup, result, err)
}

func (c *defaultReconciler) backupLeaseAction(ctx context.Context, backup *backupv1.Backup, result leases.Result, err error) (action, error) {
	if err != nil {
		return Abort, fmt.Errorf("acquire backup lease for backup %s: %w", backup.Name, err)
	}
	switch result.State {
	case leases.StateHeld:
		logging.Debug(ctx, "ensureActiveBackupLease: backup lease is held -> NEXT")
		return Next, nil
	case leases.StateAcquired:
		logging.Info(ctx, "acquired the backup lease", "lease", leases.DefaultName)
		logging.Debug(ctx, "Retrying backup reconciliation", "reason", "the acquired backup lease must be observed again")
		c.recorder.Event(backup, corev1.EventTypeNormal, reasonBackupStarted, "The backup has started")
		return Retry, nil
	case leases.StateConflict:
		logging.Debug(ctx, "Retrying backup reconciliation", "reason", "the backup lease was modified concurrently")
		return Retry, nil
	case leases.StateWaiting:
		logging.Debug(ctx, "Retrying backup reconciliation", "reason", "another operation still holds the backup lease", "holder", result.HolderName)
		return Retry, nil
	case leases.StateInvalid:
		metrics.UpdateInvalidLeaseTotalMetric(backup.Namespace, backup.Name)
		c.recorder.Event(backup, corev1.EventTypeWarning, reasonBackupLeaseFailed, "Acquiring the backup lease failed")
		return Abort, fmt.Errorf("backup %s is blocked by invalid lease %s/%s without a resolvable holder", backup.Name, backup.Namespace, leases.DefaultName)
	default:
		c.recorder.Event(backup, corev1.EventTypeWarning, reasonBackupLeaseFailed, "Acquiring the backup lease failed with unknown state")
		return Abort, fmt.Errorf("unknown backup lease acquisition state %d", result.State)
	}
}

// holdsBackupLease reports whether this backup currently owns the shared backup lease.
func (c *defaultReconciler) holdsBackupLease(ctx context.Context, backup *backupv1.Backup) (bool, error) {
	manager := leases.NewManager(c.client, backup.Namespace, leases.DefaultName, backupHolderResolver{client: c.client})
	holds, err := manager.Holds(ctx, backup, backupLeaseHolderKind)
	if err != nil {
		return false, fmt.Errorf("check backup lease ownership for backup %s: %w", backup.Name, err)
	}
	return holds, nil
}

func (c *defaultReconciler) ensureBackupLeaseReleased(ctx context.Context, backup *backupv1.Backup) (action, error) {
	// Safety-net, should normally not happen. Retry until provider is finished
	if !isPostProcessing(backup) && backup.DeletionTimestamp.IsZero() {
		logging.Debug(ctx, "ensureBackupLeaseReleased: the backup run is not finished -> RETRY")
		return Retry, nil
	}

	manager := leases.NewManager(c.client, backup.Namespace, leases.DefaultName, backupHolderResolver{client: c.client})
	released, err := manager.Release(ctx, backup, backupLeaseHolderKind)
	if err != nil {
		c.recorder.Event(backup, corev1.EventTypeWarning, reasonBackupLeaseFailed, "Releasing the backup lease failed")
		return Abort, fmt.Errorf("release backup lease for backup %s: %w", backup.Name, err)
	}
	if !released {
		logging.Debug(ctx, "ensureBackupLeaseReleased: backup does not hold the lease -> NEXT")
		return Next, nil
	}

	logging.Info(ctx, "released the backup lease", "lease", leases.DefaultName)
	return Next, nil
}

// backupRunOutcome derives how a finished backup run ended from its conditions.
func backupRunOutcome(backup *backupv1.Backup) string {
	if meta.IsStatusConditionTrue(backup.Status.Conditions, backupv1.ConditionCanceled) {
		return "canceled"
	}
	succeeded := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionSucceeded)
	switch {
	case succeeded == nil:
		return "unknown"
	case succeeded.Status == metav1.ConditionTrue:
		return "succeeded"
	case succeeded.Status == metav1.ConditionFalse:
		return "failed"
	default:
		return "unknown"
	}
}

// backupRunDuration reports how long the backup ran, or "unknown" while a timestamp is missing.
func backupRunDuration(backup *backupv1.Backup) string {
	start := backup.Status.StartTimestamp
	completion := backup.Status.CompletionTimestamp
	if start.IsZero() || completion.IsZero() {
		return "unknown"
	}
	return completion.Sub(start.Time).String()
}

type backupHolderResolver struct {
	client client.Client
}

func (backupHolderResolver) Kind() string { return backupLeaseHolderKind }

func (r backupHolderResolver) Get(ctx context.Context, namespace, name string) (client.Object, error) {
	holder := &backupv1.Backup{}
	if err := r.client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, holder); err != nil {
		return nil, err
	}
	return holder, nil
}

// IsTerminal deliberately keys on the terminal Succeeded condition rather than on isPostProcessing:
// a backup that is still post-processing must release its lease on its own.
func (backupHolderResolver) IsTerminal(holder client.Object) bool {
	backup, ok := holder.(*backupv1.Backup)
	if !ok {
		return false
	}
	succeeded := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionSucceeded)
	return hasBackupSucceededOrFailed(succeeded)
}
