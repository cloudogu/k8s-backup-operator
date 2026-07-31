package backup

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"time"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/k8s-backup-operator/pkg/annotations"
	"github.com/go-logr/logr"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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
	reasonBackupDeleting                              = "BackupDeleting"
	reasonBackupNotDeleting                           = "BackupNotDeleting"
	reasonTimeWindowNotExpired                        = "TimeWindowNotExpired"
	reasonTimeWindowExpiredBackupNotStarted           = "TimeWindowExpiredBackupNotStarted"
	reasonTimeWindowExpiredBackupIsRunning            = "TimeWindowExpiredBackupIsRunning"
	reasonTimeWindowExpiredBackupHasFailed            = "TimeWindowExpiredBackupHasFailed"
)

const (
	maintenanceModeTitle = "Service temporary unavailable"
	maintenanceModeText  = "Backup in progress"
)

const (
	backupConfigMapName     = "k8s-backup-operator-backup-config"
	backupRetryTimeLimitKey = "retryTimeLimit"
)

// defaultBackupTTL is ten years, basically infinity in backup standards
const defaultBackupTTL = 87660 * time.Hour

const debug = 1

var defaultLabels = map[string]string{
	"app":                      "ces",
	"k8s.cloudogu.com/part-of": "backup",
}

var veleroBackupIsRunningPhases = []velerov1.BackupPhase{
	velerov1.BackupPhaseNew,
	velerov1.BackupPhaseInProgress,
	velerov1.BackupPhaseFinalizing,
}

var veleroBackupFailedPhases = []velerov1.BackupPhase{
	velerov1.BackupPhaseFailedValidation,
	velerov1.BackupPhaseWaitingForPluginOperationsPartiallyFailed,
	velerov1.BackupPhaseFinalizingPartiallyFailed,
	velerov1.BackupPhasePartiallyFailed,
	velerov1.BackupPhaseFailed,
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

type Clock interface {
	Now() time.Time
}

type DefaultClock struct{}

func (d DefaultClock) Now() time.Time {
	return time.Now()
}

type defaultReconciler struct {
	client             client.Client
	maintenanceGateway maintenanceGateway
	clock              Clock
}

func NewReconciler(client client.Client, maintenanceGateway maintenanceGateway, clock Clock) *defaultReconciler {
	return &defaultReconciler{
		client:             client,
		maintenanceGateway: maintenanceGateway,
		clock:              clock,
	}
}

func (c *defaultReconciler) checkBackupDeletion(ctx context.Context, backup *backupv1.Backup, logger logr.Logger) (action, error) {
	if !backup.DeletionTimestamp.IsZero() {
		var veleroBackup = &velerov1.Backup{}
		err := c.client.Get(ctx, backup.GetNamespacedName(), veleroBackup)
		if apierrors.IsNotFound(err) {
			logger.V(debug).Info("check backup deletion: backup is deleted and velero backup not found -> remove finalizer, Abort")

			controllerutil.RemoveFinalizer(backup, backupv1.BackupFinalizer)
			updateErr := c.client.Update(ctx, backup)
			if updateErr != nil {
				return Abort, fmt.Errorf("update backup after removing finalizer: %w", updateErr)
			}
			return Abort, nil
		}
		if err != nil {
			return Abort, fmt.Errorf("get the velero backup resource to check if it exists: %w", err)
		}
		deleteReq, createErr := c.createVeleroDeleteBackupRequestIfNotExists(ctx, backup, logger)
		if createErr != nil {
			return Abort, createErr
		}
		patchErr := c.markBackupAsDeleting(ctx, backup, deleteReq.Status.Phase)
		if patchErr != nil {
			return Abort, fmt.Errorf("patch conditions to mark backup as deleting: %w", patchErr)
		}
		return Retry, nil
	}

	logger.V(debug).Info("check backup deletion: backup is not deleted -> mark backup as not deleting, Next")

	patchErr := c.markBackupAsNotDeleting(ctx, backup)
	if patchErr != nil {
		logger.Error(patchErr, "Failed to patch condition to mark backup as not deleting")
		return Abort, fmt.Errorf("patch condition to mark backup as not deleting: %w", patchErr)
	}

	return Next, nil
}

func (c *defaultReconciler) createVeleroDeleteBackupRequestIfNotExists(
	ctx context.Context,
	backup *backupv1.Backup,
	logger logr.Logger,
) (*velerov1.DeleteBackupRequest, error) {
	var deleteBackupRequest = &velerov1.DeleteBackupRequest{}
	err := c.client.Get(ctx, backup.GetNamespacedName(), deleteBackupRequest)
	if apierrors.IsNotFound(err) {
		logger.V(debug).Info("check backup deletion: delete backup request not found -> create one")

		var newDeleteBackupRequest = &velerov1.DeleteBackupRequest{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: backup.Namespace,
				Name:      backup.Name,
			},
			Spec: velerov1.DeleteBackupRequestSpec{
				BackupName: backup.Name,
			},
		}
		createErr := c.client.Create(ctx, newDeleteBackupRequest)
		if createErr != nil {
			return nil, fmt.Errorf("create velero delete backup request: %w", createErr)
		}
		return newDeleteBackupRequest, nil
	}

	if err != nil {
		return nil, fmt.Errorf("get velero delete backup request: %w", err)
	}

	logger.V(debug).Info("check backup deletion: delete backup request already exists")
	return deleteBackupRequest, nil
}

func (c *defaultReconciler) checkBackupCompletion(ctx context.Context, backup *backupv1.Backup, logger logr.Logger) (action, error) {
	if backup.Status.CompletionTimestamp.IsZero() {
		logger.V(debug).Info("checkBackupCompletion: backup not completed -> NEXT")
		return Next, nil
	}

	logger.V(debug).Info("checkBackupCompletion: backup completed -> ABORT")
	return Abort, nil
}

func (c *defaultReconciler) checkBackupCancellation(ctx context.Context, backup *backupv1.Backup, logger logr.Logger) (action, error) {
	var backupConfigMap = &corev1.ConfigMap{}
	err := c.client.Get(ctx, types.NamespacedName{Namespace: backup.Namespace, Name: backupConfigMapName}, backupConfigMap)
	if err != nil {
		return Abort, fmt.Errorf("get backup operator config map '%s/%s': %w", backup.Namespace, backupConfigMapName, err)
	}

	backupRetryTimeLimitAsStr, ok := backupConfigMap.Data[backupRetryTimeLimitKey]
	if !ok {
		return Abort, fmt.Errorf("read key '%s' from backup operator config map '%s'", backupRetryTimeLimitKey, backupConfigMapName)
	}

	backupRetryTimeLimit, err := strconv.Atoi(backupConfigMap.Data[backupRetryTimeLimitKey])
	if err != nil {
		return Abort, fmt.Errorf("convert backup retry limit from string '%s' to int: %w", backupRetryTimeLimitAsStr, err)
	}

	timeWindowHasExpired := c.clock.Now().Sub(backup.CreationTimestamp.Time) > time.Duration(backupRetryTimeLimit)*time.Minute
	if !timeWindowHasExpired {
		logger.V(debug).Info("checkBackupCancellation: time window has not expired -> Canceled = False, NEXT")

		patchErr := c.markBackupAsTimeWindowNotExpired(ctx, backup)
		if patchErr != nil {
			return Abort, fmt.Errorf("patch status to mark the canceled condition as 'time window not expired'")
		}
		return Next, nil
	}

	backupHasNotStarted := backup.Status.StartTimestamp.IsZero()
	if backupHasNotStarted {
		logger.V(debug).Info("checkBackupCancellation: time window has expired, Backup has not started -> Canceled = True, ABORT")

		patchErr := c.markBackupAsTimeWindowExpiredBackupNotStarted(ctx, backup)
		if patchErr != nil {
			return Abort, fmt.Errorf("patch status to mark the canceled condition as 'time window not expired'")
		}
		return Abort, nil
	}

	backupHasStarted := !backup.Status.StartTimestamp.IsZero()
	if backupHasStarted {
		var veleroBackup = &velerov1.Backup{}
		err = c.client.Get(ctx, backup.GetNamespacedName(), veleroBackup)
		if err != nil {
			return Abort, fmt.Errorf("get velero backup: %w", err)
		}

		isVeleroBackupRunning := slices.Contains(veleroBackupIsRunningPhases, veleroBackup.Status.Phase)
		if isVeleroBackupRunning {
			logger.V(debug).Info("checkBackupCancellation: time window has expired, Backup is running -> Canceled = False, NEXT")

			patchErr := c.markBackupAsTimeWindowExpiredBackupRunning(ctx, backup)
			if patchErr != nil {
				return Abort, fmt.Errorf("patch status to mark the canceled condition as 'time window expired and backup is running'")
			}
			return Next, nil
		}

		hasVeleroBackupFailed := slices.Contains(veleroBackupFailedPhases, veleroBackup.Status.Phase)
		if hasVeleroBackupFailed {
			logger.V(debug).Info("checkBackupCancellation: time window has expired, Backup failed -> Canceled = True, Abort")

			patchErr := c.markBackupAsTimeWindowExpiredBackupFailed(ctx, backup)
			if patchErr != nil {
				return Abort, fmt.Errorf("patch status to mark the canceled condition as 'time window expired and backup has failed'")
			}
			return Abort, nil
		}
	}

	return Abort, nil
}

func (c *defaultReconciler) checkVeleroBackupStorage(ctx context.Context, backup *backupv1.Backup, logger logr.Logger) (action, error) {
	veleroBackupStorageLocation := velerov1.BackupStorageLocation{}
	namespacedName := types.NamespacedName{Namespace: backup.Namespace, Name: veleroBackupStorageName}
	err := c.client.Get(ctx, namespacedName, &veleroBackupStorageLocation)

	if err != nil {
		patchErr := c.markVeleroBackupStorageNotAvailable(ctx, backup)
		if patchErr != nil {
			return Abort, fmt.Errorf("patch conditions to mark preparation as failed: %w", patchErr)
		}

		logger.V(debug).Info("checkVeleroBackupStorage: backup storage location not available because of an error -> Prepared = False, RETRY")

		return Retry, fmt.Errorf("check velero backup storage location 'name=%s': %w", veleroBackupStorageName, err)
	}

	if veleroBackupStorageLocation.Status.Phase != velerov1.BackupStorageLocationPhaseAvailable {
		logger.Info(fmt.Sprintf("Velero backup storage location 'name=%s' is not available.", veleroBackupStorageName))

		patchErr := c.markVeleroBackupStorageNotAvailable(ctx, backup)
		if patchErr != nil {
			return Abort, fmt.Errorf("patch conditions to mark preparation as failed: %w", patchErr)
		}
		return Retry, nil
	}

	patchErr := c.markVeleroBackupStorageAvailable(ctx, backup)
	if patchErr != nil {
		return Abort, fmt.Errorf("patch status to mark the preparation conditions as failed: %w", patchErr)
	}

	logger.V(debug).Info("checkVeleroBackupStorage: backup storage location is available -> Prepared = True, NEXT")

	return Next, nil
}

func (c *defaultReconciler) checkMaintenanceModeActiveBeforeBackup(ctx context.Context, backup *backupv1.Backup, logger logr.Logger) (action, error) {
	isActive, err := c.maintenanceGateway.isMaintenanceModeActive(ctx)
	if err != nil {
		return Abort, fmt.Errorf("check if maintenance is active: %w", err)
	}

	isBackupCompleted := !backup.Status.CompletionTimestamp.IsZero()
	if !isActive && isBackupCompleted {
		logger.V(debug).Info("check maintenance mode before backup: is not active but backup is completed -> NEXT")
		return Next, nil
	}

	if !isActive {
		logger.V(debug).Info("check maintenance mode before backup: is not active -> activate it; RETRY")

		err2 := c.maintenanceGateway.activateMaintenanceMode(ctx, maintenanceModeTitle, maintenanceModeText)
		if err2 != nil {
			return Abort, fmt.Errorf("activate maintenance mode: %w", err)
		}

		patchErr := c.makeBackupAsMaintenanceModeNotActive(ctx, backup)
		if patchErr != nil {
			return Abort, fmt.Errorf("patch status to mark the complete condition as failed")
		}
		return Retry, nil
	}

	logger.V(debug).Info("check maintenance mode before backup: is active -> NEXT")
	return Next, nil
}

func (c *defaultReconciler) checkVeleroBackupResource(ctx context.Context, backup *backupv1.Backup, logger logr.Logger) (action, error) {
	var veleroBackup = &velerov1.Backup{}
	err := c.client.Get(ctx, backup.GetNamespacedName(), veleroBackup)

	if apierrors.IsNotFound(err) {
		veleroBackupCr := c.createVeleroBackupResource(backup)
		createErr := c.client.Create(ctx, veleroBackupCr)
		if createErr != nil {
			return Abort, fmt.Errorf("create velero backup resource: %w", createErr)
		}

		patchErr := c.markVeleroBackupResourceDoesNotExist(ctx, backup)
		if patchErr != nil {
			return Abort, fmt.Errorf("patch status of backup resource: %w", patchErr)
		}

		logger.V(debug).Info("checkVeleroBackupResource: velero backup not found -> Completed = False, RETRY")

		return Retry, nil
	}

	if err != nil {
		return Abort, fmt.Errorf("get velero backup resource: %w", err)
	}

	logger.V(debug).Info("checkVeleroBackupResource: velero backup found -> NEXT")

	return Next, nil
}

func (c *defaultReconciler) checkVeleroBackupCompletion(ctx context.Context, backup *backupv1.Backup, logger logr.Logger) (action, error) {
	var veleroBackup = &velerov1.Backup{}
	err := c.client.Get(ctx, backup.GetNamespacedName(), veleroBackup)
	if err != nil {
		return Abort, fmt.Errorf("checking velero backup resource for completion: %w", err)
	}

	if veleroBackup.Status.Phase != velerov1.BackupPhaseCompleted {
		logger.V(debug).Info(fmt.Sprintf("checkVeleroBackupCompletion: velero backup not completed (phase: %s) -> NEXT", veleroBackup.Status.Phase))

		patchErr := c.markBackupAsNotCompleted(ctx, backup, veleroBackup.Status.Phase)
		if patchErr != nil {
			return Abort, fmt.Errorf("mark backup as not completed: %w", patchErr)
		}
		return Retry, nil
	}

	logger.V(debug).Info("checkVeleroBackupCompletion: velero backup completed -> NEXT")
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

func (c *defaultReconciler) checkMaintenanceModeNotActiveAfterBackup(ctx context.Context, backup *backupv1.Backup, logger logr.Logger) (action, error) {
	var veleroBackup = &velerov1.Backup{}
	err := c.client.Get(ctx, backup.GetNamespacedName(), veleroBackup)

	if err != nil {
		return Abort, fmt.Errorf("get velero backup resource: %w", err)
	}

	backupCompleted := veleroBackup.Status.Phase == velerov1.BackupPhaseCompleted
	maintenanceModeIsActive, err := c.maintenanceGateway.isMaintenanceModeActive(ctx)
	if err != nil {
		return Abort, fmt.Errorf("check maintenance mode after backup completion: %w", err)
	}

	if maintenanceModeIsActive && backupCompleted {
		logger.V(debug).Info("checkMaintenanceModeNotActiveAfterBackup: is active and backup is completed -> deactivate it, Completed = True, Next")

		err2 := c.maintenanceGateway.deactivateMaintenanceMode(ctx)
		if err2 != nil {
			return Abort, fmt.Errorf("deactivate maintenance mode after backup completion: %w", err)
		}

		patchErr := c.markBackupAsCompleted(ctx, backup)
		if patchErr != nil {
			return Abort, fmt.Errorf("mark backup as completed: %w", patchErr)
		}
		return Next, nil
	}

	logger.V(debug).Info("checkMaintenanceModeNotActiveAfterBackup: is not active -> Next",
		"isMaintenanceActive", maintenanceModeIsActive,
		"isBackupCompleted", backupCompleted,
	)
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

func (c *defaultReconciler) makeBackupAsMaintenanceModeNotActive(ctx context.Context, backup *backupv1.Backup) error {
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
	return patchErr
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

func (c *defaultReconciler) markBackupAsNotDeleting(ctx context.Context, backup *backupv1.Backup) error {
	deleting := metav1.Condition{
		Type:    backupv1.ConditionDeleting,
		Status:  metav1.ConditionFalse,
		Reason:  reasonBackupNotDeleting,
		Message: "Backup is not deleting.",
	}
	return c.patchStatus(ctx, backup, func(status *backupv1.BackupStatus) {
		meta.SetStatusCondition(&status.Conditions, deleting)
	})
}

func (c *defaultReconciler) markBackupAsDeleting(ctx context.Context, backup *backupv1.Backup, deletingPhase velerov1.DeleteBackupRequestPhase) error {
	deleting := metav1.Condition{
		Type:    backupv1.ConditionDeleting,
		Status:  metav1.ConditionTrue,
		Reason:  reasonBackupDeleting,
		Message: fmt.Sprintf("Backup is deleting (phase: %s)", deletingPhase),
	}
	return c.patchStatus(ctx, backup, func(status *backupv1.BackupStatus) {
		meta.SetStatusCondition(&status.Conditions, deleting)
	})
}

func (c *defaultReconciler) markBackupAsTimeWindowNotExpired(ctx context.Context, backup *backupv1.Backup) error {
	canceled := metav1.Condition{
		Type:    backupv1.ConditionCanceled,
		Status:  metav1.ConditionFalse,
		Reason:  reasonTimeWindowNotExpired,
		Message: "The time window has not expired.",
	}
	return c.patchStatus(ctx, backup, func(status *backupv1.BackupStatus) {
		meta.SetStatusCondition(&status.Conditions, canceled)
	})
}

func (c *defaultReconciler) markBackupAsTimeWindowExpiredBackupNotStarted(ctx context.Context, backup *backupv1.Backup) error {
	canceled := metav1.Condition{
		Type:    backupv1.ConditionCanceled,
		Status:  metav1.ConditionTrue,
		Reason:  reasonTimeWindowExpiredBackupNotStarted,
		Message: "The backup has not started when the time window expired.",
	}
	return c.patchStatus(ctx, backup, func(status *backupv1.BackupStatus) {
		meta.SetStatusCondition(&status.Conditions, canceled)
	})
}

func (c *defaultReconciler) markBackupAsTimeWindowExpiredBackupRunning(ctx context.Context, backup *backupv1.Backup) error {
	canceled := metav1.Condition{
		Type:    backupv1.ConditionCanceled,
		Status:  metav1.ConditionFalse,
		Reason:  reasonTimeWindowExpiredBackupIsRunning,
		Message: "The backup was running when the time window expired.",
	}
	return c.patchStatus(ctx, backup, func(status *backupv1.BackupStatus) {
		meta.SetStatusCondition(&status.Conditions, canceled)
	})
}

func (c *defaultReconciler) markBackupAsTimeWindowExpiredBackupFailed(ctx context.Context, backup *backupv1.Backup) error {
	canceled := metav1.Condition{
		Type:    backupv1.ConditionCanceled,
		Status:  metav1.ConditionTrue,
		Reason:  reasonTimeWindowExpiredBackupHasFailed,
		Message: "The backup had failed when the time window expired.",
	}
	return c.patchStatus(ctx, backup, func(status *backupv1.BackupStatus) {
		meta.SetStatusCondition(&status.Conditions, canceled)
	})
}

func (c *defaultReconciler) patchStatus(ctx context.Context, backup *backupv1.Backup, updateFn statusUpdate) error {
	backupBeforePatch := backup.DeepCopy()
	updateFn(&backup.Status)

	return c.client.Status().Patch(ctx, backup, client.MergeFrom(backupBeforePatch))
}
