package backup

import (
	"context"
	"errors"
	"fmt"
	k8sv1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
)

type backupDeleteManager struct {
	client   ecosystemBackupInterface
	recorder eventRecorder
}

// NewBackupDeleteManager creates a new instance of backupDeleteManager.
func NewBackupDeleteManager(client ecosystemBackupInterface, recorder eventRecorder) *backupDeleteManager {
	return &backupDeleteManager{client: client, recorder: recorder}
}

func (bdm *backupDeleteManager) delete(ctx context.Context, backup *k8sv1.Backup) error {
	backup, err := bdm.client.UpdateStatusDeleting(ctx, backup)
	if err != nil {
		return fmt.Errorf("failed to set status [%s] in backup resource: %w", k8sv1.BackupStatusDeleting, err)
	}

	err = bdm.triggerBackupDelete(ctx, backup)
	if err != nil {
		return fmt.Errorf("failed to delete backup: %w", err)
	}

	_, err = bdm.client.RemoveFinalizer(ctx, backup, k8sv1.BackupFinalizer)
	if err != nil {
		return fmt.Errorf("failed to remove finalizer %s from backup resource: %w", k8sv1.BackupFinalizer, err)
	}

	return nil
}

func (bdm *backupDeleteManager) triggerBackupDelete(ctx context.Context, backup *k8sv1.Backup) error {
	backupProvider, err := getBackupProvider(backup, bdm.client, bdm.recorder)
	if err != nil {
		return fmt.Errorf("failed to get backup provider: %w", err)
	}

	err = backupProvider.DeleteBackup(ctx, backup)
	return err
}

func getBackupProvider(backup *k8sv1.Backup, client ecosystemBackupInterface, recorder eventRecorder) (Provider, error) {
	provider := backup.Spec.Provider
	// TODO Check if Provider is really installed.
	switch provider {
	case k8sv1.ProviderVelero:
		return newVeleroProvider(client, recorder, backup.Namespace)
	default:
		return nil, errors.New(fmt.Sprintf("unknown backup provider %s", provider))
	}
}
