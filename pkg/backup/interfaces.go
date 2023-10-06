package backup

import (
	"context"
	"github.com/cloudogu/cesapp-lib/registry"
	"github.com/cloudogu/k8s-backup-operator/pkg/api/ecosystem"
	v1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"time"
)

type ecosystemInterface interface {
	ecosystem.Interface
}

type ecosystemBackupInterface interface {
	ecosystem.BackupInterface
}

type controllerManager interface {
	ctrl.Manager
}

type eventRecorder interface {
	record.EventRecorder
}

type MaintenanceModeSwitch interface {
	ActivateMaintenanceMode(title string, text string) error
	DeactivateMaintenanceMode() error
}

type backupControllerManager interface {
	createManager
	deleteManager
}

type createManager interface {
	create(ctx context.Context, backup *v1.Backup) error
}

type deleteManager interface {
	delete(ctx context.Context, backup *v1.Backup) error
}

// Provider encapsulates different backup provider like velero.
type Provider interface {
	// CreateBackup creates backup according to the backup configuration in v1.Backup.
	CreateBackup(ctx context.Context, backup *v1.Backup) error
	// DeleteBackup deletes backup from the cluster state and the backup storage.
	DeleteBackup(ctx context.Context, backup *v1.Backup) error
	// CheckReady validates if the provider is ready to receive backup requests.
	CheckReady(ctx context.Context) error
}

type requeueHandler interface {
	Handle(ctx context.Context, contextMessage string, backup *v1.Backup, originalErr error, requeueStatus string) (ctrl.Result, error)
}

// requeuableError indicates that the current error requires the operator to requeue the component.
type requeuableError interface {
	error
	// GetRequeueTime returns the time to wait before the next reconciliation.
	GetRequeueTime(requeueTimeNanos time.Duration) time.Duration
}

type etcdRegistry interface {
	registry.Registry
}
