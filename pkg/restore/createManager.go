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
	clientSet             ecosystemInterface
	recorder              eventRecorder
	maintenanceModeSwitch maintenanceModeSwitch
}

func newCreateManager(clientSet ecosystemInterface, recorder eventRecorder, registry cesRegistry) *defaultCreateManager {
	maintenanceSwitch := maintenance.New(registry.GlobalConfig())
	return &defaultCreateManager{clientSet: clientSet, recorder: recorder, maintenanceModeSwitch: maintenanceSwitch}
}

func (cm *defaultCreateManager) create(ctx context.Context, restore *v1.Restore) error {
	logger := log.FromContext(ctx)
	cm.recorder.Event(restore, corev1.EventTypeNormal, v1.CreateEventReason, "Start restore process")

	restoreClient := cm.clientSet.EcosystemV1Alpha1().Restores(restore.Namespace)
	restore, err := restoreClient.UpdateStatusInProgress(ctx, restore)
	if err != nil {
		return fmt.Errorf("failed to set status [%s] in restore resource [%s]: %w", v1.RestoreStatusInProgress, restore.Name, err)
	}

	restore, err = restoreClient.AddFinalizer(ctx, restore, v1.RestoreFinalizer)
	if err != nil {
		return fmt.Errorf("failed to add finalizer [%s] in restore resource [%s]: %w", v1.RestoreFinalizer, restore.Name, err)
	}

	backup, err := cm.getBackupFromRestore(ctx, restore)
	if err != nil {
		return err
	}

	provider, err := restoreprovider.GetProvider(ctx, backup.Spec.Provider, restore.Namespace, cm.recorder)
	if err != nil {
		return fmt.Errorf("failed to get restore provider: %w", err)
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
		_, updateStatusErr := restoreClient.UpdateStatusFailed(ctx, restore)
		if updateStatusErr != nil {
			err = errors.Join(err, fmt.Errorf("failed to update restore status to '%s': %w", v1.RestoreStatusFailed, updateStatusErr))
		}
	}

	_, err = restoreClient.UpdateStatusCompleted(ctx, restore)
	if err != nil {
		return fmt.Errorf("failed to set status [%s] in backup resource: %w", v1.RestoreStatusCompleted, err)
	}

	return nil
}

func (cm *defaultCreateManager) getBackupFromRestore(ctx context.Context, restore *v1.Restore) (*v1.Backup, error) {
	backupName := restore.Spec.BackupName
	backup, err := cm.clientSet.EcosystemV1Alpha1().Backups(restore.Namespace).Get(ctx, backupName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("found no backup with name [%s] for restore resource [%s]", backupName, restore.Name)
		}

		return nil, &requeue.GenericRequeueableError{
			ErrMsg: fmt.Sprintf("failed to get backup [%s] for restore resource [%s]", backupName, restore.Name),
			Err:    err,
		}
	}

	return backup, nil
}
