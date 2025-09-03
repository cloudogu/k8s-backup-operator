package provider

import (
	"context"
	"github.com/cloudogu/k8s-backup-lib/api/ecosystem"
	v1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// EventRecorder provides functionality to commit events to kubernetes resources.
type EventRecorder interface {
	record.EventRecorder
}

// Provider encapsulates different provider like velero.
type Provider interface {
	// CreateBackup creates backup according to the backup configuration in v1.Backup.
	CreateBackup(ctx context.Context, backup *v1.Backup) error
	// DeleteBackup deletes backup from the cluster state and the backup storage.
	DeleteBackup(ctx context.Context, backup *v1.Backup) error
	// CheckReady validates if the provider is ready to receive backup requests.
	CheckReady(ctx context.Context) error
	// CreateRestore creates a restore according to the restore configuration in v1.Restore.
	CreateRestore(ctx context.Context, restore *v1.Restore) error
	// DeleteRestore just deletes the provider restore object.
	DeleteRestore(ctx context.Context, restore *v1.Restore) error
	// SyncBackups syncs backup CRs with provider backups.
	SyncBackups(ctx context.Context) error
	// SyncBackupStatus syncs the status of the backup CR with the corresponding provider backup.
	// The provider backup must be completed or an error is thrown.
	SyncBackupStatus(ctx context.Context, backup *v1.Backup) error
}

type K8sClient interface {
	client.WithWatch
}

type EcosystemClientSet interface {
	ecosystem.Interface
}
