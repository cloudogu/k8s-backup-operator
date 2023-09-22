package backup

import (
	"context"
	"github.com/cloudogu/k8s-backup-operator/pkg/api/ecosystem"
	v1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
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
}
