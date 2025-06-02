package backup

import (
	"context"
	"errors"
	"fmt"
	v1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
	"github.com/cloudogu/k8s-backup-operator/pkg/provider"
	"github.com/cloudogu/k8s-backup-operator/pkg/retry"
	"github.com/cloudogu/k8s-registry-lib/repository"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	maintenanceModeTitle = "Service temporary unavailable"
	maintenanceModeText  = "Backup in progress"
)

type backupCreateManager struct {
	k8sClient              k8sClient
	clientSet              ecosystemInterface
	namespace              string
	globalConfigRepository globalConfigRepository
	recorder               eventRecorder
	maintenanceModeSwitch  MaintenanceModeSwitch
	ownerRefBackuper       ownerReferenceBackup
}

// newBackupCreateManager creates a new instance of backupCreateManager.
func newBackupCreateManager(k8sClient k8sClient, clientSet ecosystemInterface, namespace string, recorder eventRecorder, globalConfigRepository globalConfigRepository, ownerRefBackuper ownerReferenceBackup) *backupCreateManager {
	maintenanceModeSwitch := repository.NewMaintenanceModeAdapter("k8s-backup-operator", clientSet.CoreV1().ConfigMaps(namespace))
	return &backupCreateManager{k8sClient: k8sClient, clientSet: clientSet, namespace: namespace, globalConfigRepository: globalConfigRepository, recorder: recorder, maintenanceModeSwitch: maintenanceModeSwitch, ownerRefBackuper: ownerRefBackuper}
}

func (bcm *backupCreateManager) create(ctx context.Context, backup *v1.Backup) error {
	logger := log.FromContext(ctx)
	bcm.recorder.Event(backup, corev1.EventTypeNormal, v1.CreateEventReason, "Start backup process")
	backupClient := bcm.clientSet.EcosystemV1Alpha1().Backups(bcm.namespace)

	backup, err := backupClient.UpdateStatusInProgress(ctx, backup)
	if err != nil {
		return fmt.Errorf("failed to set status [%s] in backup resource: %w", v1.BackupStatusInProgress, err)
	}

	backup.Status.StartTimestamp = metav1.Now()
	backup, err = backupClient.UpdateStatus(ctx, backup, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update start time in status of backup resource: %w", err)
	}

	defer func(backup *v1.Backup) {
		errDefer := bcm.updateCompletionTimestamp(ctx, backup)
		if errDefer != nil {
			logger.Error(fmt.Errorf("failed to update completion time in status of backup resource: %w", err), "backup error")
		}
	}(backup)

	err = bcm.ownerRefBackuper.BackupOwnerReferences(ctx)
	if err != nil {
		return fmt.Errorf("failed to backup owner references: %w", err)
	}

	backup, err = backupClient.AddFinalizer(ctx, backup, v1.BackupFinalizer)
	if err != nil {
		return fmt.Errorf("failed to set finalizer %s to backup resource: %w", v1.BackupFinalizer, err)
	}

	backup, err = backupClient.AddLabels(ctx, backup)
	if err != nil {
		return fmt.Errorf("failed to add labels to backup resource: %w", err)
	}

	err = bcm.triggerBackup(ctx, backup)
	if err != nil {
		err = fmt.Errorf("failed to trigger backup provider: %w", err)
		_, updateStatusErr := backupClient.UpdateStatusFailed(ctx, backup)
		if updateStatusErr != nil {
			err = errors.Join(err, fmt.Errorf("failed to update backups status to 'Failed': %w", updateStatusErr))
		}

		return err
	}

	_, err = backupClient.UpdateStatusCompleted(ctx, backup)
	if err != nil {
		return fmt.Errorf("failed to set status [%s] in backup resource: %w", v1.BackupStatusCompleted, err)
	}

	return nil
}

func (bcm *backupCreateManager) updateCompletionTimestamp(ctx context.Context, backup *v1.Backup) error {
	backupClient := bcm.clientSet.EcosystemV1Alpha1().Backups(bcm.namespace)
	return retry.OnConflict(func() error {
		backup, err := backupClient.Get(ctx, backup.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		backup.Status.CompletionTimestamp = metav1.Now()
		_, err = backupClient.UpdateStatus(ctx, backup, metav1.UpdateOptions{})
		return err
	})
}

func (bcm *backupCreateManager) triggerBackup(ctx context.Context, backup *v1.Backup) error {
	backupProvider, err := provider.Get(ctx, backup, backup.Spec.Provider, backup.Namespace, bcm.recorder, bcm.k8sClient)
	if err != nil {
		return fmt.Errorf("failed to get backup provider: %w", err)
	}

	return backupProvider.CreateBackup(ctx, backup)
}
