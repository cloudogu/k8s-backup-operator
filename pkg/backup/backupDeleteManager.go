package backup

import (
	"context"
	"errors"
	"fmt"
	v1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type backupDeleteManager struct {
	client   ecosystemBackupInterface
	recorder eventRecorder
}

// NewBackupDeleteManager creates a new instance of backupDeleteManager.
func NewBackupDeleteManager(client ecosystemBackupInterface, recorder eventRecorder) *backupDeleteManager {
	return &backupDeleteManager{client: client, recorder: recorder}
}

func (bdm *backupDeleteManager) delete(ctx context.Context, backup *v1.Backup) error {
	backup, err := bdm.client.UpdateStatusDeleting(ctx, backup)
	if err != nil {
		return fmt.Errorf("failed to set status [%s] in backup resource: %w", v1.BackupStatusDeleting, err)
	}

	err = bdm.triggerBackupDelete(ctx, backup)
	if err != nil {
		_, statusErr := bdm.client.UpdateStatusFailed(ctx, backup)
		if statusErr != nil {
			log.FromContext(ctx).Error(statusErr, "status error")
		}

		return fmt.Errorf("failed to delete backup: %w", err)
	}

	_, err = bdm.client.RemoveFinalizer(ctx, backup, v1.BackupFinalizer)
	if err != nil {
		return fmt.Errorf("failed to remove finalizer %s from backup resource: %w", v1.BackupFinalizer, err)
	}

	return nil
}

func (bdm *backupDeleteManager) triggerBackupDelete(ctx context.Context, backup *v1.Backup) error {
	backupProvider, err := getBackupProvider(backup, bdm.client, bdm.recorder)
	if err != nil {
		return fmt.Errorf("failed to get backup provider: %w", err)
	}

	return backupProvider.DeleteBackup(ctx, backup)
}

func getBackupProvider(backup *v1.Backup, client ecosystemBackupInterface, recorder eventRecorder) (Provider, error) {
	provider := backup.Spec.Provider
	switch provider {
	case v1.ProviderVelero:
		return newVeleroProvider(client, recorder, backup.Namespace)
	default:
		return nil, errors.New(fmt.Sprintf("unknown backup provider %s", provider))
	}
}
