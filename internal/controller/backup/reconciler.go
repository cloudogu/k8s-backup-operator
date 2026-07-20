package backup

import (
	"context"
	"fmt"
	"time"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/k8s-backup-operator/pkg/annotations"
	"github.com/go-logr/logr"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const veleroBackupStorageName = "default"
const (
	reasonVeleroBackupStorageNotAvailable             = "VeleroBackupStorageNotAvailable"
	reasonVeleroBackupStorageAvailable                = "VeleroBackupStorageAvailable"
	reasonPreparationNotCompleted                     = "PreparationNotCompleted"
	reasonMaintenanceModesIsNotActive                 = "MaintenanceModesIsNotActive"
	reasonVeleroBackupResourceDoesNotExist            = "VeleroBackupResourceDoesNotExist"
	reasonVeleroBackupNotCompleted                    = "VeleroBackupNotCompleted"
	reasonMaintenanceModeIsActiveAfterBackupCompleted = "MaintenanceModeIsActiveAfterBackupCompleted"
	reasonBackupCompleted                             = "BackupCompleted"
)
const (
	maintenanceModeTitle = "Service temporary unavailable"
	maintenanceModeText  = "Backup in progress"
)

// defaultBackupTTL is ten years, basically infinity in backup standards
const defaultBackupTTL = 87660 * time.Hour

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

func NewReconciler(client client.Client, maintenanceGateway maintenanceGateway) *defaultReconciler {
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

		patchErr := c.markVeleroBackupStorageNotAvailable(ctx, backup)
		if patchErr != nil {
			logger.Error(err, fmt.Sprintf("Failed to patch condition for backup namespace='%s' name='%s'", namespace, backup.Name))
			return Abort, fmt.Errorf("patch conditions to mark preparation as failed: %w", patchErr)
		}

		return Retry, err
	}

	if veleroBackupStorageLocation.Status.Phase != velerov1.BackupStorageLocationPhaseAvailable {
		logger.Info(fmt.Sprintf("Velero backup storage location 'name=%s' is not available.", veleroBackupStorageName))

		patchErr := c.markVeleroBackupStorageNotAvailable(ctx, backup)
		if patchErr != nil {
			logger.Error(err, fmt.Sprintf("Failed to patch condition for backup namespace='%s' name='%s'", namespace, backup.Name))
			return Abort, fmt.Errorf("patch conditions to mark preparation as failed: %w", patchErr)
		}
		return Retry, nil
	}

	patchErr := c.markVeleroBackupStorageAvailable(ctx, backup)
	if patchErr != nil {
		logger.Error(err, fmt.Sprintf("Failed to patch condition for backup namespace='%s' name='%s'", namespace, backup.Name))
		return Abort, fmt.Errorf("patch status to mark the preparation conditions as failed %w", patchErr)
	}

	return Next, nil
}

func (c *defaultReconciler) checkMaintenanceModeActiveBeforeBackup(ctx context.Context, backup *backupv1.Backup, namespace string, logger logr.Logger) (action, error) {
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

func (c *defaultReconciler) checkVeleroBackupResource(
	ctx context.Context,
	backup *backupv1.Backup,
	namespace string,
	logger logr.Logger,
) (action, error) {
	var veleroBackup = &velerov1.Backup{}
	name := types.NamespacedName{Namespace: backup.Namespace, Name: backup.Name}
	err := c.client.Get(ctx, name, veleroBackup)

	if apierrors.IsNotFound(err) {
		veleroBackupCr := c.createVeleroBackupResource(backup)
		createErr := c.client.Create(ctx, veleroBackupCr)
		if createErr != nil {
			logger.Error(err, "Failed to create velero backup resource", "namespace", backup.Namespace, "name", backup.Name)
			return Abort, fmt.Errorf("create velero backup resource: %w", createErr)
		}

		patchErr := c.markVeleroBackupResourceDoesNotExist(ctx, backup)
		if patchErr != nil {
			logger.Error(err, "Failed to patch status of backup resource", "namespace", backup.Namespace, "name", backup.Name)
			return Abort, fmt.Errorf("patch status of backup resource: %w", patchErr)
		}
		return Retry, nil
	}

	if err != nil {
		logger.Error(err, "Failed to get velero backup resource", "namespace", backup.Namespace, "name", backup.Name)
		return Abort, fmt.Errorf("get velero backup resource: %w", err)
	}

	return Next, nil
}

func (c *defaultReconciler) checkVeleroBackupCompletion(
	ctx context.Context,
	backup *backupv1.Backup,
	namespace string,
	logger logr.Logger,
) (action, error) {
	var veleroBackup = &velerov1.Backup{}
	name := types.NamespacedName{Namespace: backup.Namespace, Name: backup.Name}
	err := c.client.Get(ctx, name, veleroBackup)

	if err != nil {
		logger.Error(err, "Failed to get velero backup resource while checking for completion",
			"namespace", backup.Namespace,
			"name", backup.Name,
		)
		return Abort, fmt.Errorf("checking velero backup resource for completion: %w", err)
	}

	if veleroBackup.Status.Phase != velerov1.BackupPhaseCompleted {
		patchErr := c.markBackupAsNotCompleted(ctx, backup, veleroBackup.Status.Phase)
		if patchErr != nil {
			logger.Error(err, "Failed to patch backup status condition while marking backup as not completed",
				"namespace", backup.Namespace,
				"name", backup.Name,
			)
			return Abort, fmt.Errorf("mark backup as not completed: %w", patchErr)
		}
		return Retry, nil
	}

	return Next, nil
}

func (c *defaultReconciler) createVeleroBackupResource(backup *backupv1.Backup) *velerov1.Backup {
	selectors := []*metav1.LabelSelector{
		{MatchLabels: map[string]string{"k8s.cloudogu.com/type": "global-config"}},
		{MatchExpressions: []metav1.LabelSelectorRequirement{
			{Key: "dogu.name", Operator: metav1.LabelSelectorOpExists},
		}},
		// everything besides dogu-specific config that should be included in the backup, e.g., PVCs of components etc.
		{MatchExpressions: []metav1.LabelSelectorRequirement{
			{Key: "k8s.cloudogu.com/backup-scope", Operator: metav1.LabelSelectorOpExists},
		}},
	}
	volumeFsBackup := false
	return &velerov1.Backup{
		ObjectMeta: metav1.ObjectMeta{
			Name:        backup.Name,
			Namespace:   backup.Namespace,
			Labels:      map[string]string{"app": "ces", "k8s.cloudogu.com/part-of": "backup"},
			Annotations: annotations.GetBackupAnnotations(backup.ObjectMeta),
		},
		Spec: velerov1.BackupSpec{
			IncludedNamespaces:       []string{backup.Namespace},
			IncludedResources:        []string{"configmaps", "secrets", "persistentvolumeclaims", "persistentvolumes", "dogus.k8s.cloudogu.com"},
			OrLabelSelectors:         selectors,
			TTL:                      metav1.Duration{Duration: defaultBackupTTL},
			StorageLocation:          veleroBackupStorageName,
			DefaultVolumesToFsBackup: &volumeFsBackup,
		},
	}
}

func (c *defaultReconciler) checkMaintenanceModeNotActiveAfterBackup(ctx context.Context, backup *backupv1.Backup, namespace string, logger logr.Logger) (action, error) {
	var veleroBackup = &velerov1.Backup{}
	name := types.NamespacedName{Namespace: backup.Namespace, Name: backup.Name}
	err := c.client.Get(ctx, name, veleroBackup)

	if err != nil {
		logger.Error(err,
			"Error retrieving the Velero backup resource while checking if maintenance mode is active after the backup completes.",
			"namespace", backup.Namespace,
			"backup", backup.Name,
		)
		return Abort, fmt.Errorf("get velero backup resource: %w", err)
	}

	backupCompleted := veleroBackup.Status.Phase == velerov1.BackupPhaseCompleted
	maintenanceModeIsActive, err := c.maintenanceGateway.isMaintenanceModeActive(ctx)
	if err != nil {
		logger.Error(err, "Error checking maintenance mode after backup completion")
		return Abort, fmt.Errorf("check maintenance mode: %w", err)
	}

	if maintenanceModeIsActive && backupCompleted {
		err2 := c.maintenanceGateway.deactivateMaintenanceMode(ctx)
		if err2 != nil {
			logger.Error(err, "Error deactivating the maintenance mode after backup completion")
			return Abort, fmt.Errorf("deactivate maintenance mode: %w", err)
		}

		patchErr := c.markMaintenanceModeIsActiveAfterBackupCompleted(ctx, backup)
		if patchErr != nil {
			logger.Error(err,
				"Error marking the backup as incomplete because maintenance mode is active after the backup completed.",
				"namespace", backup.Namespace,
				"backup", backup.Name,
			)
			return Abort, fmt.Errorf("mark backup as incompleted: %w", patchErr)
		}
		return Retry, nil
	}

	if !maintenanceModeIsActive && backupCompleted {
		patchErr := c.markBackupAsCompleted(ctx, backup)
		if patchErr != nil {
			logger.Error(err,
				"Error marking the backup as completed.",
				"namespace", backup.Namespace,
				"backup", backup.Name,
			)
			return Abort, fmt.Errorf("mark backup as completed: %w", patchErr)
		}
		return Next, nil
	}

	return Next, nil
}

func (c *defaultReconciler) markVeleroBackupStorageAvailable(ctx context.Context, backup *backupv1.Backup) error {
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

func (c *defaultReconciler) markVeleroBackupStorageNotAvailable(ctx context.Context, backup *backupv1.Backup) error {
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

func (c *defaultReconciler) markVeleroBackupResourceDoesNotExist(ctx context.Context, backup *backupv1.Backup) error {
	completed := metav1.Condition{
		Type:    backupv1.ConditionCompleted,
		Status:  metav1.ConditionFalse,
		Reason:  reasonVeleroBackupResourceDoesNotExist,
		Message: "Preparation not completed",
	}
	return c.patchStatus(ctx, backup, func(status *backupv1.BackupStatus) {
		meta.SetStatusCondition(&status.Conditions, completed)
	})
}

func (c *defaultReconciler) markBackupAsNotCompleted(ctx context.Context, backup *backupv1.Backup, veleroBackupPhase velerov1.BackupPhase) error {
	completed := metav1.Condition{
		Type:    backupv1.ConditionCompleted,
		Status:  metav1.ConditionFalse,
		Reason:  reasonVeleroBackupNotCompleted,
		Message: fmt.Sprintf("Velero backup not completed. Velero is in phase: %v", veleroBackupPhase),
	}
	return c.patchStatus(ctx, backup, func(status *backupv1.BackupStatus) {
		meta.SetStatusCondition(&status.Conditions, completed)
	})
}

func (c *defaultReconciler) markMaintenanceModeIsActiveAfterBackupCompleted(ctx context.Context, backup *backupv1.Backup) error {
	completed := metav1.Condition{
		Type:    backupv1.ConditionCompleted,
		Status:  metav1.ConditionFalse,
		Reason:  reasonMaintenanceModeIsActiveAfterBackupCompleted,
		Message: "The maintenance mode is active after the backup completed.",
	}
	return c.patchStatus(ctx, backup, func(status *backupv1.BackupStatus) {
		meta.SetStatusCondition(&status.Conditions, completed)
	})
}

func (c *defaultReconciler) markBackupAsCompleted(ctx context.Context, backup *backupv1.Backup) error {
	completed := metav1.Condition{
		Type:    backupv1.ConditionCompleted,
		Status:  metav1.ConditionTrue,
		Reason:  reasonBackupCompleted,
		Message: "Backup completed.",
	}
	return c.patchStatus(ctx, backup, func(status *backupv1.BackupStatus) {
		status.CompletionTimestamp = metav1.Now()
		meta.SetStatusCondition(&status.Conditions, completed)
	})
}

func (c *defaultReconciler) patchStatus(ctx context.Context, backup *backupv1.Backup, updateFn statusUpdate) error {
	backupBeforePatch := backup.DeepCopy()
	updateFn(&backup.Status)

	return c.client.Status().Patch(ctx, backup, client.MergeFrom(backupBeforePatch))
}
