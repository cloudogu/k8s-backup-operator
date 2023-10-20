package backup

import (
	"context"
	"errors"
	"fmt"
	"github.com/cloudogu/k8s-backup-operator/pkg/provider"

	"github.com/cloudogu/cesapp-lib/registry"
	v1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
	"github.com/cloudogu/k8s-backup-operator/pkg/maintenance"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	maintenanceModeTitle = "Service temporary unavailable"
	maintenanceModeText  = "Backup in progress"
)

type backupCreateManager struct {
	client                ecosystemBackupInterface
	registry              registry.Registry
	recorder              eventRecorder
	maintenanceModeSwitch MaintenanceModeSwitch
}

// NewBackupCreateManager creates a new instance of backupCreateManager.
func NewBackupCreateManager(client ecosystemBackupInterface, recorder eventRecorder, registry registry.Registry) *backupCreateManager {
	maintenanceModeSwitch := maintenance.New(registry.GlobalConfig())
	return &backupCreateManager{client: client, registry: registry, recorder: recorder, maintenanceModeSwitch: maintenanceModeSwitch}
}

func (bcm *backupCreateManager) create(ctx context.Context, backup *v1.Backup) error {
	logger := log.FromContext(ctx)
	bcm.recorder.Event(backup, corev1.EventTypeNormal, v1.CreateEventReason, "Start backup process")

	backup, err := bcm.client.UpdateStatusInProgress(ctx, backup)
	if err != nil {
		return fmt.Errorf("failed to set status [%s] in backup resource: %w", v1.BackupStatusInProgress, err)
	}

	backup, err = bcm.client.AddFinalizer(ctx, backup, v1.BackupFinalizer)
	if err != nil {
		return fmt.Errorf("failed to set finalizer %s to backup resource: %w", v1.BackupFinalizer, err)
	}

	backup, err = bcm.client.AddLabels(ctx, backup)
	if err != nil {
		return fmt.Errorf("failed to add labels to backup resource: %w", err)
	}

	err = bcm.maintenanceModeSwitch.ActivateMaintenanceMode(maintenanceModeTitle, maintenanceModeText)
	if err != nil {
		return fmt.Errorf("failed to active maintenance mode: %w", err)
	}

	defer func() {
		errDefer := bcm.maintenanceModeSwitch.DeactivateMaintenanceMode()
		if errDefer != nil {
			logger.Error(fmt.Errorf("failed to deactivate maintenance mode: [%w]", errDefer), "backup error")
		}
	}()

	err = bcm.triggerBackup(ctx, backup)
	if err != nil {
		err = fmt.Errorf("failed to trigger backup provider: %w", err)
		_, updateStatusErr := bcm.client.UpdateStatusFailed(ctx, backup)
		if updateStatusErr != nil {
			err = errors.Join(err, fmt.Errorf("failed to update backups status to 'Failed': %w", updateStatusErr))
		}

		return err
	}

	_, err = bcm.client.UpdateStatusCompleted(ctx, backup)
	if err != nil {
		return fmt.Errorf("failed to set status [%s] in backup resource: %w", v1.BackupStatusCompleted, err)
	}

	return nil
}

func (bcm *backupCreateManager) triggerBackup(ctx context.Context, backup *v1.Backup) error {
	backupProvider, err := provider.GetProvider(ctx, backup.Spec.Provider, backup.Namespace, bcm.recorder)
	if err != nil {
		return fmt.Errorf("failed to get backup provider: %w", err)
	}

	return backupProvider.CreateBackup(ctx, backup)
}
