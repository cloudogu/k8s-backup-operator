package velero

import (
	"context"
	"k8s.io/client-go/discovery"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/cloudogu/k8s-backup-lib/api/v1"

	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/record"
)

type eventRecorder interface {
	record.EventRecorder
}

type k8sWatchClient interface {
	client.WithWatch
}

type discoveryClient interface {
	discovery.DiscoveryInterface
}

type manager interface {
	backupManager
	restoreManager
	syncManager
}

type backupManager interface {
	// CreateBackup creates backup according to the backup configuration in v1.Backup.
	CreateBackup(ctx context.Context, backup *v1.Backup) error
	// DeleteBackup deletes backup from the cluster state and the backup storage.
	DeleteBackup(ctx context.Context, backup *v1.Backup) error
}

type restoreManager interface {
	// CreateRestore creates a restore according to the restore configuration in v1.Restore.
	CreateRestore(ctx context.Context, restore *v1.Restore) error
	// DeleteRestore deletes the velero restore resource.
	DeleteRestore(ctx context.Context, restore *v1.Restore) error
}

type syncManager interface {
	// SyncBackups syncs backup CRs with velero CRs
	SyncBackups(ctx context.Context) error
	// SyncBackupStatus syncs the status of the backup CR with the corresponding velero backup.
	// The velero backup must be completed or an error is thrown.
	SyncBackupStatus(ctx context.Context, backup *v1.Backup) error
}

// The following interfaces are here to generate mocks.

//nolint:unused
//goland:noinspection GoUnusedType
type watchInterface interface {
	watch.Interface
}
