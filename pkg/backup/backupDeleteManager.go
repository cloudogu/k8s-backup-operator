package backup

import (
	"context"
	"fmt"
	k8sv1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
	"github.com/cloudogu/k8s-backup-operator/pkg/provider"
)

type backupDeleteManager struct {
	k8sClient k8sClient
	clientSet ecosystemInterface
	namespace string
	recorder  eventRecorder
}

// newBackupDeleteManager creates a new instance of backupDeleteManager.
func newBackupDeleteManager(k8sClient k8sClient, clientSet ecosystemInterface, namespace string, recorder eventRecorder) *backupDeleteManager {
	return &backupDeleteManager{k8sClient: k8sClient, clientSet: clientSet, namespace: namespace, recorder: recorder}
}

func (bdm *backupDeleteManager) delete(ctx context.Context, backup *k8sv1.Backup) error {
	backupClient := bdm.clientSet.EcosystemV1Alpha1().Backups(bdm.namespace)
	backup, err := backupClient.UpdateStatusDeleting(ctx, backup)
	if err != nil {
		return fmt.Errorf("failed to set status [%s] in backup resource: %w", k8sv1.BackupStatusDeleting, err)
	}

	err = bdm.triggerBackupDelete(ctx, backup)
	if err != nil {
		return fmt.Errorf("failed to delete backup: %w", err)
	}

	_, err = backupClient.RemoveFinalizer(ctx, backup, k8sv1.BackupFinalizer)
	if err != nil {
		return fmt.Errorf("failed to remove finalizer %s from backup resource: %w", k8sv1.BackupFinalizer, err)
	}

	return nil
}

func (bdm *backupDeleteManager) triggerBackupDelete(ctx context.Context, backup *k8sv1.Backup) error {
	backupProvider, err := provider.Get(ctx, backup, backup.Spec.Provider, backup.Namespace, bdm.recorder, bdm.k8sClient)
	if err != nil {
		return fmt.Errorf("failed to get backup provider: %w", err)
	}

	err = backupProvider.DeleteBackup(ctx, backup)
	return err
}
