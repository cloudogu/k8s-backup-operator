package restore

import (
	"context"
	"errors"
	"fmt"
	v1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
	restoreprovider "github.com/cloudogu/k8s-backup-operator/pkg/provider"
	"github.com/cloudogu/k8s-registry-lib/repository"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	maintenanceModeTitle = "Service temporary unavailable"
	maintenanceModeText  = "Restore in progress"
)

type defaultCreateManager struct {
	k8sClient             k8sClient
	ecosystemClientSet    ecosystemInterface
	namespace             string
	cleanup               cleanupManager
	recorder              eventRecorder
	maintenanceModeSwitch maintenanceModeSwitch
	ownerRefRestorer      ownerReferenceRestore
}

func newCreateManager(
	k8sClient k8sClient,
	ecosystemClientSet ecosystemInterface,
	namespace string,
	recorder eventRecorder,
	cleanup cleanupManager,
	ownerRefRestorer ownerReferenceRestore,
) *defaultCreateManager {
	maintenanceSwitch := repository.NewMaintenanceModeAdapter("k8s-backup-operator", ecosystemClientSet.CoreV1().ConfigMaps(namespace))
	return &defaultCreateManager{
		k8sClient:             k8sClient,
		ecosystemClientSet:    ecosystemClientSet,
		namespace:             namespace,
		recorder:              recorder,
		maintenanceModeSwitch: maintenanceSwitch,
		cleanup:               cleanup,
		ownerRefRestorer:      ownerRefRestorer,
	}
}

func (cm *defaultCreateManager) create(ctx context.Context, restore *v1.Restore) error {
	logger := log.FromContext(ctx)
	cm.recorder.Event(restore, corev1.EventTypeNormal, v1.CreateEventReason, "Start restore process")

	restoreClient := cm.ecosystemClientSet.EcosystemV1Alpha1().Restores(cm.namespace)

	restoreName := restore.Name
	restore, err := restoreClient.UpdateStatusInProgress(ctx, restore)
	if err != nil {
		return fmt.Errorf("failed to set status [%s] in restore resource [%s]: %w", v1.RestoreStatusInProgress, restoreName, err)
	}

	restore, err = restoreClient.AddFinalizer(ctx, restore, v1.RestoreFinalizer)
	if err != nil {
		return fmt.Errorf("failed to add finalizer [%s] in restore resource [%s]: %w", v1.RestoreFinalizer, restoreName, err)
	}

	restore, err = restoreClient.AddLabels(ctx, restore)
	if err != nil {
		return fmt.Errorf("failed to add labels to restore resource [%s]: %w", restoreName, err)
	}

	provider, err := restoreprovider.Get(ctx, restore, restore.Spec.Provider, restore.Namespace, cm.recorder, cm.k8sClient, cm.ecosystemClientSet)
	if err != nil {
		return fmt.Errorf("failed to get restore provider [%s]: %w", restore.Spec.Provider, err)
	}

	err = cm.maintenanceModeSwitch.Activate(ctx, repository.MaintenanceModeDescription{Title: maintenanceModeTitle, Text: maintenanceModeText})
	if err != nil {
		logger.Error(err, "The Maintenance mode could not be activated. Continuing anyways...")
	}

	defer func() {
		errDefer := cm.maintenanceModeSwitch.Deactivate(ctx)
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
		_, updateStatusErr := restoreClient.UpdateStatusFailed(ctx, restore)
		if updateStatusErr != nil {
			err = errors.Join(err, fmt.Errorf("failed to update restore status to '%s': %w", v1.RestoreStatusFailed, updateStatusErr))
		}

		return err
	}

	err = provider.SyncBackups(ctx)
	if err != nil {
		return fmt.Errorf("failed to sync backups with provider: %w", err)
	}

	err = cm.ownerRefRestorer.RestoreOwnerReferences(ctx)
	if err != nil {
		return fmt.Errorf("failed to restore owner references: %w", err)
	}

	_, err = restoreClient.UpdateStatusCompleted(ctx, restore)
	if err != nil {
		return fmt.Errorf("failed to set status [%s] in restore resource [%s]: %w", v1.RestoreStatusCompleted, restoreName, err)
	}

	return nil
}
