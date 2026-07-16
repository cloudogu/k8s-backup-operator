package backup

import (
	"context"
	"fmt"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/go-logr/logr"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const veleroBackupStorageName = "default"
const (
	reasonVeleroBackupStorageNotAvailable = "VeleroBackupStorageNotAvailable"
	reasonVeleroBackupStorageAvailable    = "VeleroBackupStorageAvailable"
	reasonPreparationNotCompleted         = "PreparationNotCompleted"
	reasonMaintenanceModesIsNotActive     = "MaintenanceModesIsNotActive"
)
const (
	maintenanceModeTitle = "Service temporary unavailable"
	maintenanceModeText  = "Backup in progress"
)

var defaultLabels = map[string]string{
	"app":                      "ces",
	"k8s.cloudogu.com/part-of": "backup",
}

const (
	blueprintIdAnnotation    = "backup.cloudogu.com/blueprintId"
	blueprintDogusAnnotation = "backup.cloudogu.com/dogus"
)

type maintenanceGateway interface {
	isMaintenanceModeActive(ctx context.Context) (bool, error)
	activateMaintenanceMode(ctx context.Context, title string, text string) error
	deactivateMaintenanceMode(ctx context.Context) error
}

type statusUpdate func(status *backupv1.BackupStatus)

type defaultReconciler struct {
	client             client.Client
	maintenanceGateway maintenanceGateway
}

func newReconciler(client client.Client, maintenanceGateway maintenanceGateway) *defaultReconciler {
	return &defaultReconciler{
		client:             client,
		maintenanceGateway: maintenanceGateway,
	}
}

func (c *defaultReconciler) checkVeleroBackupStorage(ctx context.Context, backup *backupv1.Backup, namespace string, logger logr.Logger) (action, error) {
	veleroBackupStorageLocation := velerov1.BackupStorageLocation{}
	namespacedName := types.NamespacedName{Namespace: namespace, Name: veleroBackupStorageName}
	err := c.client.Get(ctx, namespacedName, &veleroBackupStorageLocation)

	if err != nil {
		logger.Error(err, fmt.Sprintf("Failed to check velero backup storage location 'name=%s'", veleroBackupStorageName))

		patchErr := c.markPreparationFailed(ctx, backup)
		if patchErr != nil {
			logger.Error(err, fmt.Sprintf("Failed to patch condition for backup namespace='%s' name='%s'", namespace, backup.Name))
			return Abort, fmt.Errorf("patch conditions to mark preparation as failed: %w", patchErr)
		}

		return Retry, err
	}

	if veleroBackupStorageLocation.Status.Phase != velerov1.BackupStorageLocationPhaseAvailable {
		logger.Info(fmt.Sprintf("Velero backup storage location 'name=%s' is not available.", veleroBackupStorageName))

		patchErr := c.markPreparationFailed(ctx, backup)
		if patchErr != nil {
			logger.Error(err, fmt.Sprintf("Failed to patch condition for backup namespace='%s' name='%s'", namespace, backup.Name))
			return Abort, fmt.Errorf("patch conditions to mark preparation as failed: %w", patchErr)
		}
		return Retry, nil
	}

	patchErr := c.markPreparationSuccess(ctx, backup)
	if patchErr != nil {
		logger.Error(err, fmt.Sprintf("Failed to patch condition for backup namespace='%s' name='%s'", namespace, backup.Name))
		return Abort, fmt.Errorf("patch status to mark the preparation conditions as failed %w", patchErr)
	}

	return Next, nil
}

func (c *defaultReconciler) checkMaintenanceModeIsActive(ctx context.Context, backup *backupv1.Backup, namespace string, logger logr.Logger) (action, error) {
	isActive, err := c.maintenanceGateway.isMaintenanceModeActive(ctx)
	if err != nil {
		logger.Error(err, "Failed to check maintenance mode")
		return Abort, fmt.Errorf("check if maintenance is active: %w", err)
	}

	if !isActive {
		err2 := c.maintenanceGateway.activateMaintenanceMode(ctx, maintenanceModeTitle, maintenanceModeText)
		if err2 != nil {
			logger.Error(err, "Failed to activate maintenance mode")
			return Abort, fmt.Errorf("activate maintenance mode: %w", err)
		}

		completed := metav1.Condition{
			Type:    backupv1.ConditionCompleted,
			Status:  metav1.ConditionFalse,
			Reason:  reasonMaintenanceModesIsNotActive,
			Message: "Maintenance mode is not active",
		}

		patchErr := c.patchStatus(ctx, backup, func(status *backupv1.BackupStatus) {
			meta.SetStatusCondition(&status.Conditions, completed)
			status.StartTimestamp = metav1.Now()
		})
		if patchErr != nil {
			return Abort, fmt.Errorf("patch status to mark the complete condition as failed")
		}
		return Retry, nil

	}

	return Next, nil
}

func (c *defaultReconciler) checkVeleroBackup(ctx context.Context, backup *backupv1.Backup, namespace string, logger logr.Logger) (action, error) {
	//TODO implement me
	panic("implement me")
}

func (c *defaultReconciler) markPreparationSuccess(ctx context.Context, backup *backupv1.Backup) error {
	prepared := metav1.Condition{
		Type:    backupv1.ConditionPrepared,
		Status:  metav1.ConditionTrue,
		Reason:  reasonVeleroBackupStorageAvailable,
		Message: fmt.Sprintf("velero backup storage location 'name=%s' is available.", veleroBackupStorageName),
	}
	return c.patchStatus(ctx, backup, func(status *backupv1.BackupStatus) {
		meta.SetStatusCondition(&status.Conditions, prepared)
	})
}

func (c *defaultReconciler) markPreparationFailed(ctx context.Context, backup *backupv1.Backup) error {
	prepared := metav1.Condition{
		Type:    backupv1.ConditionPrepared,
		Status:  metav1.ConditionFalse,
		Reason:  reasonVeleroBackupStorageNotAvailable,
		Message: fmt.Sprintf("velero backup storage location 'name=%s' is not available.", veleroBackupStorageName),
	}
	completed := metav1.Condition{
		Type:    backupv1.ConditionCompleted,
		Status:  metav1.ConditionFalse,
		Reason:  reasonPreparationNotCompleted,
		Message: "Preparation not completed",
	}
	return c.patchStatus(ctx, backup, func(status *backupv1.BackupStatus) {
		meta.SetStatusCondition(&status.Conditions, prepared)
		meta.SetStatusCondition(&status.Conditions, completed)
	})
}

func (c *defaultReconciler) patchStatus(ctx context.Context, backup *backupv1.Backup, updateFn statusUpdate) error {
	backupBeforePatch := backup.DeepCopy()
	updateFn(&backup.Status)

	return c.client.Status().Patch(ctx, backup, client.MergeFrom(backupBeforePatch))
}
