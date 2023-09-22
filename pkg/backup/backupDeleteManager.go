package backup

import (
	"context"
	v1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
)

type backupDeleteManager struct {
	client   ecosystemBackupInterface
	recorder eventRecorder
}

// NewBackupDeleteManager creates a new instance of backupDeleteManager.
func NewBackupDeleteManager(client ecosystemBackupInterface, recorder eventRecorder) *backupDeleteManager {
	return &backupDeleteManager{client: client, recorder: recorder}
}

func (bcm *backupDeleteManager) delete(ctx context.Context, backup *v1.Backup) error {
	return nil
}
