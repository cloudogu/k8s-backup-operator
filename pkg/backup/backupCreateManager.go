package backup

import (
	"context"
	"errors"
	"fmt"
	"github.com/cloudogu/cesapp-lib/registry"
	"github.com/cloudogu/k8s-backup-operator/pkg/api/ecosystem"
	v1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
	"github.com/cloudogu/k8s-backup-operator/pkg/maintenance"
	"github.com/cloudogu/k8s-backup-operator/pkg/provider/velero"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var newVeleroProvider = func(client ecosystem.BackupInterface, recorder eventRecorder) Provider {
	return velero.New(client, recorder)
}

const (
	CreateEventReason        = "Creation"
	ErrorOnCreateEventReason = "ErrCreation"
)

const (
	// TODO
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
	bcm.recorder.Event(backup, corev1.EventTypeNormal, CreateEventReason, "Start backup process")

	backup, err := bcm.client.UpdateStatusInProgress(ctx, backup)
	if err != nil {
		return fmt.Errorf("failed to set status [%s] in backup resource: %w", v1.BackupStatusInProgress, err)
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
		return fmt.Errorf("failed to trigger backup provider: %w", err)
	}

	_, err = bcm.client.UpdateStatusCompleted(ctx, backup)
	if err != nil {
		return fmt.Errorf("failed to set status [%s] in backup resource: %w", v1.BackupStatusCompleted, err)
	}

	return nil
}

func (bcm *backupCreateManager) triggerBackup(ctx context.Context, backup *v1.Backup) error {
	provider := backup.Spec.Provider
	var backupProvider Provider = nil
	switch provider {
	case v1.ProviderVelero:
		bcm.recorder.Event(backup, corev1.EventTypeNormal, CreateEventReason, "Use velero as backup provider")
		backupProvider = newVeleroProvider(bcm.client, bcm.recorder)
	default:
		return errors.New(fmt.Sprintf("unknown backup provider %s", provider))
	}

	return backupProvider.CreateBackup(ctx, backup)
}
