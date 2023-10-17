package restore

import (
	"context"
	"errors"
	"fmt"
	"github.com/cloudogu/k8s-backup-operator/pkg/maintenance"
	restoreprovider "github.com/cloudogu/k8s-backup-operator/pkg/provider"
	"github.com/cloudogu/k8s-backup-operator/pkg/requeue"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
)

const (
	maintenanceModeTitle = "Service temporary unavailable"
	maintenanceModeText  = "Restore in progress"
)

type defaultCreateManager struct {
	restoreClient         ecosystemRestoreInterface
	backupClient          ecosystemBackupInterface
	recorder              eventRecorder
	maintenanceModeSwitch maintenanceModeSwitch
}

func newCreateManager(restoreClient ecosystemRestoreInterface, backupClient ecosystemBackupInterface, recorder eventRecorder, registry cesRegistry) *defaultCreateManager {
	maintenanceSwitch := maintenance.New(registry.GlobalConfig())
	return &defaultCreateManager{restoreClient: restoreClient, backupClient: backupClient, recorder: recorder, maintenanceModeSwitch: maintenanceSwitch}
}

func (cm *defaultCreateManager) create(ctx context.Context, restore *v1.Restore) error {
	logger := log.FromContext(ctx)
	cm.recorder.Event(restore, corev1.EventTypeNormal, v1.CreateEventReason, "Start restore process")

	restoreName := restore.Name
	restore, err := cm.restoreClient.UpdateStatusInProgress(ctx, restore)
	if err != nil {
		return fmt.Errorf("failed to set status [%s] in restore resource [%s]: %w", v1.RestoreStatusInProgress, restoreName, err)
	}

	restore, err = cm.restoreClient.AddFinalizer(ctx, restore, v1.RestoreFinalizer)
	if err != nil {
		return fmt.Errorf("failed to add finalizer [%s] in restore resource [%s]: %w", v1.RestoreFinalizer, restoreName, err)
	}

	backup, err := cm.getBackupFromRestore(ctx, restore)
	if err != nil {
		return err
	}

	provider, err := restoreprovider.GetProvider(ctx, backup.Spec.Provider, restore.Namespace, cm.recorder)
	if err != nil {
		return fmt.Errorf("failed to get restore provider [%s]: %w", backup.Spec.Provider, err)
	}

	err = cm.maintenanceModeSwitch.ActivateMaintenanceMode(maintenanceModeTitle, maintenanceModeText)
	if err != nil {
		return fmt.Errorf("failed to activate maintenance mode: %w", err)
	}

	defer func() {
		errDefer := cm.maintenanceModeSwitch.DeactivateMaintenanceMode()
		if errDefer != nil {
			logger.Error(fmt.Errorf("failed to deactivate maintenance mode: [%w]", errDefer), "restore error")
		}
	}()

	err = provider.CreateRestore(ctx, restore)
	if err != nil {
		err = fmt.Errorf("failed to trigger provider: %w", err)
		_, updateStatusErr := cm.restoreClient.UpdateStatusFailed(ctx, restore)
		if updateStatusErr != nil {
			err = errors.Join(err, fmt.Errorf("failed to update restore status to '%s': %w", v1.RestoreStatusFailed, updateStatusErr))
		}

		return err
	}

	_, err = cm.restoreClient.UpdateStatusCompleted(ctx, restore)
	if err != nil {
		return fmt.Errorf("failed to set status [%s] in restore resource [%s]: %w", v1.RestoreStatusCompleted, restoreName, err)
	}

	return nil
}

func (cm *defaultCreateManager) getBackupFromRestore(ctx context.Context, restore *v1.Restore) (*v1.Backup, error) {
	backupName := restore.Spec.BackupName
	backup, err := cm.backupClient.Get(ctx, backupName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("found no backup with name [%s] for restore resource [%s]: %w", backupName, restore.Name, err)
		}

		return nil, &requeue.GenericRequeueableError{
			ErrMsg: fmt.Sprintf("failed to get backup [%s] for restore resource [%s]", backupName, restore.Name),
			Err:    err,
		}
	}

	return backup, nil
}
