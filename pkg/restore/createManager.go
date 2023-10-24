package restore

import (
	"context"
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
	"github.com/cloudogu/k8s-backup-operator/pkg/maintenance"
	restoreprovider "github.com/cloudogu/k8s-backup-operator/pkg/provider"
)

const (
	maintenanceModeTitle = "Service temporary unavailable"
	maintenanceModeText  = "Restore in progress"
)

type defaultCreateManager struct {
	restoreClient         ecosystemRestoreInterface
	cleanup               cleanupManager
	recorder              eventRecorder
	maintenanceModeSwitch maintenanceModeSwitch
}

func newCreateManager(
	restoreClient ecosystemRestoreInterface,
	recorder eventRecorder,
	registry cesRegistry,
	statefulSetInterface statefulSetInterface,
	serviceInterface serviceInterface,
	cleanup cleanupManager,
) *defaultCreateManager {
	maintenanceSwitch := maintenance.NewWithLooseCoupling(registry.GlobalConfig(), statefulSetInterface, serviceInterface)
	return &defaultCreateManager{
		restoreClient:         restoreClient,
		recorder:              recorder,
		maintenanceModeSwitch: maintenanceSwitch,
		cleanup:               cleanup,
	}
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

	restore, err = cm.restoreClient.AddLabels(ctx, restore)
	if err != nil {
		return fmt.Errorf("failed to add labels to restore resource [%s]: %w", restoreName, err)
	}

	provider, err := restoreprovider.GetProvider(ctx, restore, restore.Spec.Provider, restore.Namespace, cm.recorder)
	if err != nil {
		return fmt.Errorf("failed to get restore provider [%s]: %w", restore.Spec.Provider, err)
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

	err = cm.cleanup.Cleanup(ctx)
	if err != nil {
		return fmt.Errorf("failed to cleanup before restore: %w", err)
	}

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
