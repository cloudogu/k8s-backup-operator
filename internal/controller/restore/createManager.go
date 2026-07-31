package restore

import (
	"context"
	"fmt"

	v1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/k8s-backup-operator/pkg/metrics"
	restoreprovider "github.com/cloudogu/k8s-backup-operator/pkg/provider"
	"github.com/cloudogu/k8s-registry-lib/repository"
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
	scaleManager          scaleManager
	recorder              eventRecorder
	maintenanceModeSwitch maintenanceModeSwitch
}

func newCreateManager(
	k8sClient k8sClient,
	namespace string,
	recorder eventRecorder,
	scaleManager scaleManager,
) *defaultCreateManager {
	maintenanceSwitch := repository.NewMaintenanceModeAdapter("k8s-backup-operator", k8sClient, namespace)
	return &defaultCreateManager{
		k8sClient:             k8sClient,
		namespace:             namespace,
		recorder:              recorder,
		maintenanceModeSwitch: maintenanceSwitch,
		scaleManager:          scaleManager,
	}
}

// create is in a wip. Right now it finishes the workflow of a Restore whose provider restore already succeeded: it
// synchronizes the backups, recovers the workloads and reports the outcome.
func (cm *defaultCreateManager) create(ctx context.Context, restore *v1.Restore) error {
	logger := log.FromContext(ctx)

	provider, err := restoreprovider.Get(ctx, restore, restore.Spec.Provider, restore.Namespace, cm.recorder, cm.k8sClient)
	if err != nil {
		return fmt.Errorf("failed to get restore provider [%s]: %w", restore.Spec.Provider, err)
	}

	// The maintenance mode was activated by the preparation stage and persisted across the whole
	// provider execution. A terminally failed provider restore is switched off by the completion stage,
	// so this deactivation only ever covers the successful path.
	defer func() {
		errDefer := cm.maintenanceModeSwitch.Deactivate(ctx, false)
		if errDefer != nil {
			logger.Error(fmt.Errorf("failed to deactivate maintenance mode: [%w]", errDefer), "restore error")
		}
	}()

	err = provider.SyncBackups(ctx)
	if err != nil {
		return fmt.Errorf("failed to sync backups with provider: %w", err)
	}

	err = cm.scaleManager.ScaleUp(ctx)
	if err != nil {
		return fmt.Errorf("failed to scale up workloads after restore: %w", err)
	}

	// The three milestones are written together because they are reached together in this manager. The
	// stages that replace it resolve each of them on its own, as it happens.
	_, err = newConditionUpdater(cm.k8sClient).setConditions(ctx, restore,
		reachedMilestone(v1.ConditionBackupsSynchronized, ReasonBackupSynchronizationCompleted,
			"The backup resources were synchronized with the provider."),
		reachedMilestone(v1.ConditionWorkloadsRecovered, ReasonWorkloadRecoveryCompleted,
			"The workloads were scaled up again and the maintenance mode was switched off."),
		reachedMilestone(v1.ConditionSuccessful, ReasonRestoreCompleted,
			"The restore workflow finished successfully."),
	)
	if err != nil {
		return fmt.Errorf("failed to set status [%s] in restore resource [%s]: %w", v1.RestoreStatusCompleted, restore.Name, err)
	}
	metrics.UpdateRestoreStatusMetrics(cm.namespace, restore.Name, restore.Spec.BackupName, v1.RestoreStatusCompleted)

	return nil
}

// reachedMilestone is a reached milestone of the restore workflow.
func reachedMilestone(conditionType string, reason string, message string) metav1.Condition {
	return metav1.Condition{
		Type:    conditionType,
		Status:  metav1.ConditionTrue,
		Reason:  reason,
		Message: message,
	}
}
