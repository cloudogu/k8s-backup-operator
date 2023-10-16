package velero

import (
	"context"
	"github.com/cloudogu/k8s-backup-operator/pkg/api/ecosystem"
	v1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/record"

	"github.com/vmware-tanzu/velero/pkg/generated/clientset/versioned"
	velerov1 "github.com/vmware-tanzu/velero/pkg/generated/clientset/versioned/typed/velero/v1"
)

type eventRecorder interface {
	record.EventRecorder
}

type ecosystemBackupInterface interface {
	ecosystem.BackupInterface
}

type veleroClientSet interface {
	versioned.Interface
}

// The following interfaces are here to generate mocks.

//nolint:unused
//goland:noinspection GoUnusedType
type veleroInterface interface {
	velerov1.VeleroV1Interface
}

//nolint:unused
//goland:noinspection GoUnusedType
type veleroBackupInterface interface {
	velerov1.BackupInterface
}

//nolint:unused
//goland:noinspection GoUnusedType
type veleroBackupStorageLocationInterface interface {
	velerov1.BackupStorageLocationInterface
}

//nolint:unused
//goland:noinspection GoUnusedType
type veleroDeleteBackupRequest interface {
	velerov1.DeleteBackupRequestInterface
}

//nolint:unused
//goland:noinspection GoUnusedType
type ecosystemWatch interface {
	watch.Interface
}

type manager interface {
	backupManager
	restoreManager
}

type backupManager interface {
	// CreateBackup creates backup according to the backup configuration in v1.Backup.
	CreateBackup(ctx context.Context, backup *v1.Backup) error
	// DeleteBackup deletes backup from the cluster state and the backup storage.
	DeleteBackup(ctx context.Context, backup *v1.Backup) error
}

type restoreManager interface {
}
