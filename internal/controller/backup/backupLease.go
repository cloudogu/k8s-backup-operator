package backup

import (
	"context"
	"fmt"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/k8s-backup-operator/internal/leases"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const backupLeaseHolderKind = "Backup"

func (c *defaultReconciler) ensureActiveBackupLease(ctx context.Context, backup *backupv1.Backup, _ logr.Logger) (action, error) {
	manager := leases.NewManager(c.client, backup.Namespace, leases.DefaultName, backupHolderResolver{client: c.client})
	result, err := manager.Acquire(ctx, backup, backupLeaseHolderKind)
	return backupLeaseAction(backup, result, err)
}

func backupLeaseAction(backup *backupv1.Backup, result leases.Result, err error) (action, error) {
	if err != nil {
		return Abort, fmt.Errorf("acquire backup lease for backup %s: %w", backup.Name, err)
	}
	switch result.State {
	case leases.StateAcquired:
		return Next, nil
	case leases.StateChanged, leases.StateWaiting:
		return Retry, nil
	case leases.StateInvalid:
		return Abort, fmt.Errorf("backup %s is blocked by invalid lease %s/%s without a resolvable holder", backup.Name, backup.Namespace, leases.DefaultName)
	default:
		return Abort, fmt.Errorf("unknown backup lease acquisition state %d", result.State)
	}
}

func (c *defaultReconciler) ensureBackupLeaseReleased(ctx context.Context, backup *backupv1.Backup, _ logr.Logger) (action, error) {
	resolver := backupHolderResolver{client: c.client}
	if !resolver.IsTerminal(backup) && backup.DeletionTimestamp.IsZero() {
		return Next, nil
	}

	manager := leases.NewManager(c.client, backup.Namespace, leases.DefaultName, backupHolderResolver{client: c.client})
	if _, err := manager.Release(ctx, backup, backupLeaseHolderKind); err != nil {
		return Abort, fmt.Errorf("release backup lease for backup %s: %w", backup.Name, err)
	}
	return Next, nil
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

func (backupHolderResolver) IsTerminal(holder client.Object) bool {
	backup, ok := holder.(*backupv1.Backup)
	if !ok {
		return false
	}
	succeeded := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionSucceeded)
	if hasBackupSucceededOrFailed(succeeded) {
		return true
	}
	return meta.IsStatusConditionTrue(backup.Status.Conditions, backupv1.ConditionCanceled)
}
