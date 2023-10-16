package provider

import (
	"context"
	v1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
	"k8s.io/client-go/tools/record"
)

type EventRecorder interface {
	record.EventRecorder
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
