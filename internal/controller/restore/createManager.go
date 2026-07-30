package restore

import (
	"context"
	"errors"
	"fmt"

	v1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/k8s-backup-operator/internal/provider/velero"
	"github.com/cloudogu/k8s-backup-operator/pkg/metrics"
	restoreprovider "github.com/cloudogu/k8s-backup-operator/pkg/provider"
	"github.com/cloudogu/k8s-registry-lib/repository"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	maintenanceModeTitle = "Service temporary unavailable"
	maintenanceModeText  = "Restore in progress"
)

type defaultCreateManager struct {
	k8sClient             k8sClient
	namespace             string
	cleanup               cleanupManager
	scaleManager          scaleManager
	recorder              eventRecorder
	maintenanceModeSwitch maintenanceModeSwitch
}

func newCreateManager(
	k8sClient k8sClient,
	namespace string,
	recorder eventRecorder,
	cleanup cleanupManager,
	scaleManager scaleManager,
) *defaultCreateManager {
	maintenanceSwitch := repository.NewMaintenanceModeAdapter("k8s-backup-operator", k8sClient, namespace)
	return &defaultCreateManager{
		k8sClient:             k8sClient,
		namespace:             namespace,
		recorder:              recorder,
		maintenanceModeSwitch: maintenanceSwitch,
		cleanup:               cleanup,
		scaleManager:          scaleManager,
	}
}

func (cm *defaultCreateManager) create(ctx context.Context, restore *v1.Restore) error {
	logger := log.FromContext(ctx)
	metrics.InitRestoreStatusMetrics(cm.namespace, restore.Name, restore.Spec.BackupName)
	cm.recorder.Event(restore, corev1.EventTypeNormal, v1.CreateEventReason, "Start restore process")

	conditions := newConditionUpdater(cm.k8sClient)

	restoreName := restore.Name

	// The readiness gate becomes its own stage; it stays first here so that an unready provider aborts
	// before the destructive preparation runs.
	provider, err := restoreprovider.Get(ctx, restore, restore.Spec.Provider, restore.Namespace, cm.recorder, cm.k8sClient)
	if err != nil {
		return fmt.Errorf("failed to get restore provider [%s]: %w", restore.Spec.Provider, err)
	}

	// The in-progress scalar status is no longer written here: the condition stage already initialized
	// the workflow conditions, which is what the scalar is derived from.
	metrics.UpdateRestoreStatusMetrics(cm.namespace, restore.Name, restore.Spec.BackupName, v1.RestoreStatusInProgress)

	err = cm.maintenanceModeSwitch.Activate(ctx, repository.MaintenanceModeDescription{Title: maintenanceModeTitle, Text: maintenanceModeText}, false)
	if err != nil {
		logger.Error(err, "The Maintenance mode could not be activated. Continuing anyways...")
	}

	defer func() {
		errDefer := cm.maintenanceModeSwitch.Deactivate(ctx, false)
		if errDefer != nil {
			logger.Error(fmt.Errorf("failed to deactivate maintenance mode: [%w]", errDefer), "restore error")
		}
	}()

	err = cm.scaleManager.ScaleDown(ctx)
	if err != nil {
		return fmt.Errorf("failed to scale down workloads before restore: %w", err)
	}

	err = cm.cleanup.Cleanup(ctx)
	if err != nil {
		return fmt.Errorf("failed to cleanup before restore: %w", err)
	}

	err = cm.runProviderRestore(ctx, provider, restore)
	if err != nil {
		_, updateStatusErr := conditions.setConditions(ctx, restore, metav1.Condition{
			Type:    v1.ConditionSuccessful,
			Status:  metav1.ConditionFalse,
			Reason:  classifyProviderRestoreFailure(err),
			Message: fmt.Sprintf("The provider restore failed terminally: %v", err),
		})
		if updateStatusErr != nil {
			err = errors.Join(err, fmt.Errorf("failed to update restore status to '%s': %w", v1.RestoreStatusFailed, updateStatusErr))
		}
		metrics.UpdateRestoreStatusMetrics(cm.namespace, restore.Name, restore.Spec.BackupName, v1.RestoreStatusFailed)
		return err
	}

	err = provider.SyncBackups(ctx)
	if err != nil {
		return fmt.Errorf("failed to sync backups with provider: %w", err)
	}

	err = cm.scaleManager.ScaleUp(ctx)
	if err != nil {
		return fmt.Errorf("failed to scale up workloads after restore: %w", err)
	}

	_, err = conditions.setConditions(ctx, restore, metav1.Condition{
		Type:    v1.ConditionSuccessful,
		Status:  metav1.ConditionTrue,
		Reason:  ReasonRestoreCompleted,
		Message: "The restore workflow finished successfully.",
	})
	if err != nil {
		return fmt.Errorf("failed to set status [%s] in restore resource [%s]: %w", v1.RestoreStatusCompleted, restoreName, err)
	}
	metrics.UpdateRestoreStatusMetrics(cm.namespace, restore.Name, restore.Spec.BackupName, v1.RestoreStatusCompleted)
	return nil
}

// runProviderRestore ensures the one owned provider restore of the given Restore exists and then waits
// for the provider to finish it. Creating the child is idempotent, so a repeated attempt after a
// crash between child creation and parent status persistence reuses the child instead of failing.
func (cm *defaultCreateManager) runProviderRestore(ctx context.Context, provider restoreProvider, restore *v1.Restore) error {
	_, err := velero.EnsureRestore(ctx, cm.k8sClient, restore)
	if err != nil {
		cm.recorder.Event(restore, corev1.EventTypeWarning, v1.ErrorOnCreateEventReason, err.Error())

		return fmt.Errorf("failed to ensure the provider restore: %w", err)
	}

	err = provider.WaitForRestore(ctx, restore)
	if err != nil {
		return fmt.Errorf("failed to trigger provider: %w", err)
	}

	return nil
}

func classifyProviderRestoreFailure(err error) string {
	var conflictErr *velero.ConflictError
	if errors.As(err, &conflictErr) {
		return ReasonProviderRestoreConflict
	}

	return ReasonProviderRestoreFailed
}
