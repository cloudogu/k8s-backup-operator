package backup

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/cloudogu/ces-commons-lib/errors"
	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/k8s-backup-operator/pkg/annotations"
	blueprintv3 "github.com/cloudogu/k8s-blueprint-lib/v3/api/v3"
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

const (
	reasonProviderBackupStorageLocationNotFound     = "ProviderBackupStorageLocationNotFound"
	reasonProviderBackupStorageLocationNotAvailable = "ProviderBackupStorageLocationNotAvailable"
	reasonProviderBackupStorageLocationAvailable    = "ProviderBackupStorageLocationAvailable"
	reasonMaintenanceModesIsNotActive               = "MaintenanceModesIsNotActive"
	reasonProviderBackupResourceDoesNotExist        = "ProviderBackupResourceDoesNotExist"
	reasonProviderBackupInProgress                  = "ProviderBackupInProgress"
	reasonProviderBackupFailed                      = "ProviderBackupFailed"
	reasonProviderBackupSucceeded                   = "ProviderBackupSucceeded"
	reasonBackupDeleting                            = "BackupDeleting"
	reasonBackupNotDeleting                         = "BackupNotDeleting"
	reasonTimeWindowNotExpired                      = "TimeWindowNotExpired"
	reasonTimeWindowExpiredBackupNotStarted         = "TimeWindowExpiredBackupNotStarted"
	reasonTimeWindowExpiredBackupInProgress         = "TimeWindowExpiredBackupInProgress"
	reasonTimeWindowExpiredBackupFailed             = "TimeWindowExpiredBackupFailed"
	reasonTimeWindowExpiredBackupSucceeded          = "TimeWindowExpiredBackupSucceeded"
	reasonVeleroStatusSynced                        = "VeleroStatusSynced"
	reasonVeleroBackupRunning                       = "VeleroBackupRunning"
	reasonVeleroBackupFailed                        = "VeleroBackupFailed"
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

var defaultLabels = map[string]string{
	"app":                      "ces",
	"k8s.cloudogu.com/part-of": "backup",
}

var veleroBackupIsRunningPhases = []velerov1.BackupPhase{
	velerov1.BackupPhaseNew,
	velerov1.BackupPhaseInProgress,
	velerov1.BackupPhaseFinalizing,
	velerov1.BackupPhaseWaitingForPluginOperations,
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

type Clock interface {
	Now() time.Time
}

type defaultReconciler struct {
	client             client.Client
	maintenanceGateway maintenanceGateway
	clock              Clock
	backupStorageName  string
}

func NewReconciler(client client.Client, maintenanceGateway maintenanceGateway, clock Clock, backupStorageName string) *defaultReconciler {
	return &defaultReconciler{
		client:             client,
		maintenanceGateway: maintenanceGateway,
		clock:              clock,
		backupStorageName:  backupStorageName,
	}
}

func (c *defaultReconciler) ensureProviderBackupDeleted(ctx context.Context, backup *backupv1.Backup, logger logr.Logger) (action, error) {
	isBackupDeleting := !backup.DeletionTimestamp.IsZero()
	if isBackupDeleting {
		var veleroBackup, err = c.getProviderBackup(ctx, backup.GetNamespacedName())
		if err != nil {
			return Abort, fmt.Errorf("get the velero backup resource to check if it exists: %w", err)
		}

		if veleroBackup == nil {
			debug(logger, "ensureProviderBackupDeleted: backup is deleted and provider backup not found -> remove finalizer, ABORT")

			controllerutil.RemoveFinalizer(backup, backupv1.BackupFinalizer)
			updateErr := c.client.Update(ctx, backup)
			if updateErr != nil {
				return Abort, fmt.Errorf("update backup after removing finalizer: %w", updateErr)
			}
			return Abort, nil
		}

		deleteReq, createErr := c.createVeleroDeleteBackupRequestIfNotExists(ctx, backup, logger)
		if createErr != nil {
			return Abort, createErr
		}
		patchErr := c.patchStatus(ctx, backup, func(status *backupv1.BackupStatus) {
			meta.SetStatusCondition(&status.Conditions, metav1.Condition{
				Type:    backupv1.ConditionDeleting,
				Status:  metav1.ConditionTrue,
				Reason:  reasonBackupDeleting,
				Message: fmt.Sprintf("Backup is deleting (phase: %s)", deleteReq.Status.Phase),
			})
		})
		if patchErr != nil {
			return Abort, fmt.Errorf("patch conditions to mark backup as deleting: %w", patchErr)
		}
		return Retry, nil
	}

	debug(logger, "ensureProviderBackupDeleted: backup is not deleted -> mark backup as not deleting, NEXT")

	patchErr := c.patchStatus(ctx, backup, func(status *backupv1.BackupStatus) {
		meta.SetStatusCondition(&status.Conditions, metav1.Condition{
			Type:    backupv1.ConditionDeleting,
			Status:  metav1.ConditionFalse,
			Reason:  reasonBackupNotDeleting,
			Message: "Backup is not deleting.",
		})
	})
	if patchErr != nil {
		return Abort, fmt.Errorf("patch condition to mark backup as not deleting: %w", patchErr)
	}

	return Next, nil
}

func (c *defaultReconciler) ensureCompletedBackupIsIgnored(ctx context.Context, backup *backupv1.Backup, logger logr.Logger) (action, error) {
	succeededCondition := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionSucceeded)
	if succeededCondition == nil || succeededCondition.Status == metav1.ConditionUnknown {
		debug(logger, "ensureCompletedBackupIsIgnored: backup not completed -> NEXT")
		return Next, nil
	}

	debug(logger, "ensureCompletedBackupIsIgnored: backup completed -> ABORT")
	return Abort, nil
}

func (c *defaultReconciler) ensureBackupSetup(ctx context.Context, backup *backupv1.Backup, logger logr.Logger) (action, error) {
	metadataChanged := mergeMissingOrChangedValues(&backup.Labels, defaultLabels)

	var blueprintList = blueprintv3.BlueprintList{}
	err := c.client.List(ctx, &blueprintList, client.InNamespace(backup.Namespace))
	// no blueprint or no clueprint crd
	if err != nil {
		if !errors.IsNotFoundError(err) && !meta.IsNoMatchError(err) {
			return Abort, fmt.Errorf("list blueprints: %w", err)
		}
		blueprintList.Items = nil
	}

	// only take first blueprint
	if len(blueprintList.Items) > 0 {
		blueprint := blueprintList.Items[0]

		dogusAsJson, jsonerr := json.Marshal(blueprint.Spec.Blueprint.Dogus)
		if jsonerr != nil {
			return Abort, fmt.Errorf("marshal blueprint dogus to json: %w", jsonerr)
		}

		annotation := map[string]string{
			blueprintIdAnnotation:    blueprint.Spec.DisplayName,
			blueprintDogusAnnotation: string(dogusAsJson),
		}
		metadataChanged = mergeMissingOrChangedValues(&backup.Annotations, annotation) || metadataChanged
	}

	// finalizer
	metadataChanged = controllerutil.AddFinalizer(backup, backupv1.BackupFinalizer) || metadataChanged

	if !metadataChanged {
		return Next, nil
	}

	// write backup
	err = c.client.Update(ctx, backup)
	if err != nil {
		logger.Error(err, "failed to update backup to set labels and annotations")
		return Abort, err
	}

	return Next, nil
}

func mergeMissingOrChangedValues(target *map[string]string, desired map[string]string) bool {
	changed := false
	for key, desiredValue := range desired {
		if currentValue, exists := (*target)[key]; exists && currentValue == desiredValue {
			continue
		}

		if *target == nil {
			*target = make(map[string]string)
		}
		(*target)[key] = desiredValue
		changed = true
	}

	return changed
}

func (c *defaultReconciler) ensureBackupIsCanceledAfterTimeWindowExpired(ctx context.Context, backup *backupv1.Backup, logger logr.Logger) (action, error) {
	timeWindowHasExpired, err := c.hasTimeWindowExpired(ctx, backup)
	if err != nil {
		return Abort, err
	}

	if !timeWindowHasExpired {
		debug(logger, "ensureBackupIsCanceledAfterTimeWindowExpired: time window has not expired -> Canceled = False, NEXT")

		patchErr := c.patchStatus(ctx, backup, func(status *backupv1.BackupStatus) {
			meta.SetStatusCondition(&status.Conditions, metav1.Condition{
				Type:    backupv1.ConditionCanceled,
				Status:  metav1.ConditionUnknown,
				Reason:  reasonTimeWindowNotExpired,
				Message: "The time window has not expired.",
			})
		})
		if patchErr != nil {
			return Abort, fmt.Errorf("patch status to mark the canceled condition as 'time window not expired'")
		}
		return Next, nil
	}

	backupHasNotStarted := backup.Status.StartTimestamp.IsZero()
	if backupHasNotStarted {
		return c.handleTimeWindowExpiredBackupNotStarted(ctx, backup, logger)
	}

	return c.handleTimeWindowExpiredBackupStarted(ctx, backup, logger)

}

func (c *defaultReconciler) ensureBackupIsPrepared(ctx context.Context, backup *backupv1.Backup, logger logr.Logger) (action, error) {
	veleroBackupStorageLocation := velerov1.BackupStorageLocation{}
	err := c.client.Get(
		ctx,
		types.NamespacedName{Namespace: backup.Namespace, Name: c.backupStorageName},
		&veleroBackupStorageLocation,
	)

	if apierrors.IsNotFound(err) {
		debug(logger, fmt.Sprintf(
			"ensureBackupIsPrepared: backup storage location '%s' not found -> Prepared = False, RETRY",
			c.backupStorageName,
		))

		patchErr := c.patchStatus(ctx, backup, func(status *backupv1.BackupStatus) {
			meta.SetStatusCondition(&status.Conditions, metav1.Condition{
				Type:    backupv1.ConditionPrepared,
				Status:  metav1.ConditionFalse,
				Reason:  reasonProviderBackupStorageLocationNotFound,
				Message: "The provider backup storage location not found.",
			})
		})
		if patchErr != nil {
			return Abort, fmt.Errorf("patch conditions to mark preparation as failed: %w", patchErr)
		}
		return Retry, nil
	}

	if err != nil {
		return Abort, fmt.Errorf("get velero backup storage location 'name=%s': %w", c.backupStorageName, err)
	}

	if veleroBackupStorageLocation.Status.Phase != velerov1.BackupStorageLocationPhaseAvailable {
		logger.Info(fmt.Sprintf("Velero backup storage location 'name=%s' is not available.", c.backupStorageName))

		patchErr := c.patchStatus(ctx, backup, func(status *backupv1.BackupStatus) {
			meta.SetStatusCondition(&status.Conditions, metav1.Condition{
				Type:    backupv1.ConditionPrepared,
				Status:  metav1.ConditionFalse,
				Reason:  reasonProviderBackupStorageLocationNotAvailable,
				Message: fmt.Sprintf("velero backup storage location 'name=%s' is not available.", c.backupStorageName),
			})
		})
		if patchErr != nil {
			return Abort, fmt.Errorf("patch conditions to mark preparation as failed: %w", patchErr)
		}
		return Retry, nil
	}

	patchErr := c.patchStatus(ctx, backup, func(status *backupv1.BackupStatus) {
		meta.SetStatusCondition(&status.Conditions, metav1.Condition{
			Type:    backupv1.ConditionPrepared,
			Status:  metav1.ConditionTrue,
			Reason:  reasonProviderBackupStorageLocationAvailable,
			Message: fmt.Sprintf("velero backup storage location 'name=%s' is available.", c.backupStorageName),
		})
	})
	if patchErr != nil {
		return Abort, fmt.Errorf("patch status to mark the preparation conditions as failed: %w", patchErr)
	}

	debug(logger, "ensureBackupIsPrepared: backup storage location is available -> Prepared = True, NEXT")
	return Next, nil
}

func (c *defaultReconciler) ensureMaintenanceActivated(ctx context.Context, backup *backupv1.Backup, logger logr.Logger) (action, error) {
	isActive, err := c.maintenanceGateway.isMaintenanceModeActive(ctx)
	if err != nil {
		return Abort, fmt.Errorf("check if maintenance is active: %w", err)
	}

	if isActive {
		debug(logger, "ensureMaintenanceActivated: is active -> NEXT")
		return Next, nil
	}

	succeededCondition := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionSucceeded)
	if succeededCondition != nil && succeededCondition.Status == metav1.ConditionTrue {
		debug(logger, "ensureMaintenanceActivated: maintenance mode is not active and backup succeeded -> ABORT")
		return Abort, nil
	}
	if succeededCondition != nil && succeededCondition.Status == metav1.ConditionFalse {
		debug(logger, "ensureMaintenanceActivated: maintenance mode is not active and backup failed -> ABORT")
		return Abort, nil
	}

	debug(logger, "ensureMaintenanceActivated: maintenance mode is not active -> activate it; RETRY")

	maintenanceErr := c.maintenanceGateway.activateMaintenanceMode(ctx, maintenanceModeTitle, maintenanceModeText)
	if maintenanceErr != nil {
		return Abort, fmt.Errorf("activate maintenance mode: %w", maintenanceErr)
	}

	patchErr := c.patchStatus(ctx, backup, func(status *backupv1.BackupStatus) {
		meta.SetStatusCondition(&status.Conditions, metav1.Condition{
			Type:    backupv1.ConditionSucceeded,
			Status:  metav1.ConditionUnknown,
			Reason:  reasonMaintenanceModesIsNotActive,
			Message: "Maintenance mode is not active",
		})
	})
	if patchErr != nil {
		return Abort, fmt.Errorf("patch status to mark the complete condition as failed: %w", patchErr)
	}
	return Retry, nil
}

func (c *defaultReconciler) ensureProviderBackupCreated(ctx context.Context, backup *backupv1.Backup, logger logr.Logger) (action, error) {
	var veleroBackup, err = c.getProviderBackup(ctx, backup.GetNamespacedName())
	if err != nil {
		return Abort, fmt.Errorf("get velero backup resource: %w", err)
	}

	if veleroBackup == nil {
		debug(logger, "ensureProviderBackupCreated: provider backup not found -> Succeeded = Unknown, RETRY")

		veleroBackupCr := c.createVeleroBackupResource(backup)
		createErr := c.client.Create(ctx, veleroBackupCr)
		if createErr != nil {
			return Abort, fmt.Errorf("create velero backup resource: %w", createErr)
		}

		patchErr := c.patchStatus(ctx, backup, func(status *backupv1.BackupStatus) {
			meta.SetStatusCondition(&status.Conditions, metav1.Condition{
				Type:    backupv1.ConditionSucceeded,
				Status:  metav1.ConditionUnknown,
				Reason:  reasonProviderBackupResourceDoesNotExist,
				Message: "Provider backup resource does not exist.",
			})
			if status.StartTimestamp.IsZero() {
				status.StartTimestamp = metav1.NewTime(c.clock.Now())
			}
		})
		if patchErr != nil {
			return Abort, fmt.Errorf("patch status of backup resource: %w", patchErr)
		}

		return Retry, nil
	}

	debug(logger, "ensureProviderBackupCreated: provider backup resource found -> NEXT")
	return Next, nil
}

func (c *defaultReconciler) ensureProviderBackupCompleted(ctx context.Context, backup *backupv1.Backup, logger logr.Logger) (action, error) {
	var providerBackup, err = c.getProviderBackup(ctx, backup.GetNamespacedName())
	if err != nil || providerBackup == nil {
		return Abort, fmt.Errorf("checking velero backup resource for completion: %w", err)
	}

	if isProviderBackupInProgress(providerBackup) {
		debug(logger, "ensureProviderBackupCompleted: provider backup is in progress -> RETRY")
		patchErr := c.patchStatus(ctx, backup, func(status *backupv1.BackupStatus) {
			meta.SetStatusCondition(&status.Conditions, metav1.Condition{
				Type:    backupv1.ConditionSucceeded,
				Status:  metav1.ConditionUnknown,
				Reason:  reasonProviderBackupInProgress,
				Message: "Provider backup is in progress.",
			})
		})
		if patchErr != nil {
			return Abort, fmt.Errorf("patch status of backup resource: %w", patchErr)
		}
		return Retry, nil
	}

	if hasProviderBackupFailed(providerBackup) {
		debug(logger, "ensureProviderBackupCompleted: provider backup has failed -> ABORT")
		patchErr := c.patchStatus(ctx, backup, func(status *backupv1.BackupStatus) {
			meta.SetStatusCondition(&status.Conditions, metav1.Condition{
				Type:    backupv1.ConditionSucceeded,
				Status:  metav1.ConditionFalse,
				Reason:  reasonProviderBackupFailed,
				Message: "Provider backup has failed.",
			})
			if status.CompletionTimestamp.IsZero() {
				status.CompletionTimestamp = metav1.NewTime(c.clock.Now())
			}
		})
		if patchErr != nil {
			return Abort, fmt.Errorf("patch status of backup resource: %w", patchErr)
		}
		return Next, nil
	}

	if hasProviderBackupSucceeded(providerBackup) {
		debug(logger, "ensureProviderBackupCompleted: provider backup has succeeded -> NEXT")
		patchErr := c.patchStatus(ctx, backup, func(status *backupv1.BackupStatus) {
			meta.SetStatusCondition(&status.Conditions, metav1.Condition{
				Type:    backupv1.ConditionSucceeded,
				Status:  metav1.ConditionTrue,
				Reason:  reasonProviderBackupSucceeded,
				Message: "Provider backup has succeeded.",
			})
			if status.CompletionTimestamp.IsZero() {
				status.CompletionTimestamp = metav1.NewTime(c.clock.Now())
			}
		})
		if patchErr != nil {
			return Abort, fmt.Errorf("patch status of backup resource: %w", patchErr)
		}
		return Next, nil
	}

	return Abort, fmt.Errorf("provider backup with status='%s' should not occur here", providerBackup.Status.Phase)
}

func (c *defaultReconciler) ensureMaintenanceDeactivated(ctx context.Context, backup *backupv1.Backup, logger logr.Logger) (action, error) {
	maintenanceModeIsActive, err := c.maintenanceGateway.isMaintenanceModeActive(ctx)
	if err != nil {
		return Abort, fmt.Errorf("check maintenance mode: %w", err)
	}

	if !maintenanceModeIsActive {
		debug(logger, "ensureMaintenanceDeactivated: is not active -> Next")
		return Next, nil
	}

	succeededCondition := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionSucceeded)
	if hasBackupSucceededOrFailed(succeededCondition) {
		debug(logger, "ensureMaintenanceDeactivated: is active and backup succeeded or failed -> deactivate it,  Retry")

		maintenanceErr := c.maintenanceGateway.deactivateMaintenanceMode(ctx)
		if maintenanceErr != nil {
			return Abort, fmt.Errorf("deactivate maintenance mode: %w", maintenanceErr)
		}
		return Retry, nil
	}

	debug(logger, "ensureMaintenanceDeactivated: is active and backup in progress -> Next")
	return Next, nil
}

func (c *defaultReconciler) patchStatus(ctx context.Context, backup *backupv1.Backup, updateFn statusUpdate) error {
	return newConditionsUpdater(c.client).updateStatus(ctx, backup, updateFn)
}

func (c *defaultReconciler) ensureVeleroStatusSynced(
	ctx context.Context,
	backup *backupv1.Backup,
	logger logr.Logger,
) (action, error) {
	// Backups created locally do not need their status synchronized from Velero.
	if !backup.Spec.SyncedFromProvider {
		return Next, nil
	}

	// otherwise refresh conditions and status
	veleroBackup := &velerov1.Backup{}
	if err := c.client.Get(ctx, backup.GetNamespacedName(), veleroBackup); err != nil {
		return Abort, fmt.Errorf("get velero backup to sync status: %w", err)
	}

	prepared := metav1.Condition{
		Type:    backupv1.ConditionPrepared,
		Status:  metav1.ConditionTrue,
		Reason:  reasonVeleroStatusSynced,
		Message: "The backup already exists in Velero.",
	}
	completed := metav1.Condition{
		Type:   backupv1.ConditionCompleted,
		Status: metav1.ConditionFalse,
	}

	nextAction := Retry
	switch {
	case veleroBackup.Status.Phase == velerov1.BackupPhaseCompleted:
		completed.Status = metav1.ConditionTrue
		completed.Reason = reasonVeleroStatusSynced
		completed.Message = "The completed backup status was synchronized from Velero."
		nextAction = Abort
	case slices.Contains(veleroBackupFailedPhases, veleroBackup.Status.Phase):
		completed.Reason = reasonVeleroBackupFailed
		completed.Message = fmt.Sprintf("The Velero backup failed in phase %s.", veleroBackup.Status.Phase)
		nextAction = Abort
	default:
		completed.Reason = reasonVeleroBackupRunning
		completed.Message = fmt.Sprintf("The Velero backup is in phase %s.", veleroBackup.Status.Phase)
	}

	if err := c.patchStatus(ctx, backup, func(status *backupv1.BackupStatus) {
		meta.SetStatusCondition(&status.Conditions, prepared)
		meta.SetStatusCondition(&status.Conditions, completed)

		if veleroBackup.Status.StartTimestamp != nil {
			status.StartTimestamp = *veleroBackup.Status.StartTimestamp
		}
		if veleroBackup.Status.CompletionTimestamp != nil {
			status.CompletionTimestamp = *veleroBackup.Status.CompletionTimestamp
		}

		switch {
		case veleroBackup.Status.Phase == velerov1.BackupPhaseCompleted:
			status.Status = backupv1.BackupStatusCompleted
		case slices.Contains(veleroBackupFailedPhases, veleroBackup.Status.Phase):
			status.Status = backupv1.BackupStatusFailed
		default:
			status.Status = backupv1.BackupStatusInProgress
		}
	}); err != nil {
		return Abort, fmt.Errorf("patch backup status synchronized from Velero: %w", err)
	}

	debug(logger, fmt.Sprintf("synchronized backup status from Velero- Phase: %s", veleroBackup.Status.Phase))
	return nextAction, nil
}

func (c *defaultReconciler) createVeleroDeleteBackupRequestIfNotExists(
	ctx context.Context,
	backup *backupv1.Backup,
	logger logr.Logger,
) (*velerov1.DeleteBackupRequest, error) {
	var deleteBackupRequest = &velerov1.DeleteBackupRequest{}
	err := c.client.Get(ctx, backup.GetNamespacedName(), deleteBackupRequest)
	if apierrors.IsNotFound(err) {
		debug(logger, "ensureProviderBackupDeleted: delete backup request not found -> create one")

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

	debug(logger, "ensureProviderBackupDeleted: delete backup request already exists")
	return deleteBackupRequest, nil
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
			Labels:      defaultLabels,
			Annotations: annotations.GetBackupAnnotations(backup.ObjectMeta),
		},
		Spec: velerov1.BackupSpec{
			IncludedNamespaces: []string{backup.Namespace},
			IncludedResources: []string{
				"configmaps",
				"secrets",
				"persistentvolumeclaims",
				"persistentvolumes",
				"dogus.k8s.cloudogu.com",
			},
			OrLabelSelectors:         selectors,
			TTL:                      metav1.Duration{Duration: defaultBackupTTL},
			StorageLocation:          c.backupStorageName,
			DefaultVolumesToFsBackup: &volumeFsBackup,
		},
	}
}

func (c *defaultReconciler) handleTimeWindowExpiredBackupNotStarted(ctx context.Context, backup *backupv1.Backup, logger logr.Logger) (action, error) {
	debug(logger, "ensureBackupIsCanceledAfterTimeWindowExpired: time window has expired, Backup has not started -> Canceled = True, ABORT")

	patchErr := c.patchStatus(ctx, backup, func(status *backupv1.BackupStatus) {
		meta.SetStatusCondition(&status.Conditions, metav1.Condition{
			Type:    backupv1.ConditionCanceled,
			Status:  metav1.ConditionTrue,
			Reason:  reasonTimeWindowExpiredBackupNotStarted,
			Message: "The backup has not started when the time window expired.",
		})
	})
	if patchErr != nil {
		return Abort, fmt.Errorf("patch status to mark the canceled condition as 'time window not expired'")
	}
	return Abort, nil
}

func (c *defaultReconciler) handleTimeWindowExpiredBackupStarted(ctx context.Context, backup *backupv1.Backup, logger logr.Logger) (action, error) {
	veleroBackup, err := c.getProviderBackup(ctx, backup.GetNamespacedName())
	if err != nil {
		return Abort, fmt.Errorf("get velero backup: %w", err)
	}

	debug(logger,
		fmt.Sprintf(
			"ensureBackupIsCanceledAfterTimeWindowExpired: time window has expired, Backup is running (Velero backup phase: '%s')",
			veleroBackup.Status.Phase,
		),
	)

	if isProviderBackupInProgress(veleroBackup) {
		debug(logger, "ensureBackupIsCanceledAfterTimeWindowExpired: time window has expired, Backup is running -> Canceled = False, NEXT")

		patchErr := c.patchStatus(ctx, backup, func(status *backupv1.BackupStatus) {
			meta.SetStatusCondition(&status.Conditions, metav1.Condition{
				Type:    backupv1.ConditionCanceled,
				Status:  metav1.ConditionFalse,
				Reason:  reasonTimeWindowExpiredBackupInProgress,
				Message: "The backup was running when the time window expired.",
			})
		})
		if patchErr != nil {
			return Abort, fmt.Errorf("patch status to mark the canceled condition as 'time window expired and backup is running'")
		}
		return Next, nil
	}

	if hasProviderBackupFailed(veleroBackup) {
		debug(logger, "ensureBackupIsCanceledAfterTimeWindowExpired: time window has expired, Backup failed -> Canceled = True, ABORT")

		patchErr := c.patchStatus(ctx, backup, func(status *backupv1.BackupStatus) {
			meta.SetStatusCondition(&status.Conditions, metav1.Condition{
				Type:    backupv1.ConditionCanceled,
				Status:  metav1.ConditionTrue,
				Reason:  reasonTimeWindowExpiredBackupFailed,
				Message: "The backup had failed when the time window expired.",
			})
		})
		if patchErr != nil {
			return Abort, fmt.Errorf("patch status to mark the canceled condition as 'time window expired and backup has failed'")
		}

		return Abort, nil
	}

	patchErr := c.patchStatus(ctx, backup, func(status *backupv1.BackupStatus) {
		meta.SetStatusCondition(&status.Conditions, metav1.Condition{
			Type:    backupv1.ConditionCanceled,
			Status:  metav1.ConditionFalse,
			Reason:  reasonTimeWindowExpiredBackupSucceeded,
			Message: "The backup has succeeded when the time window expired.",
		})
	})
	if patchErr != nil {
		return Abort, fmt.Errorf("patch status to mark the canceled condition as 'time window expired and backup has succeeded'")
	}

	return Next, nil
}

func (c *defaultReconciler) hasTimeWindowExpired(ctx context.Context, backup *backupv1.Backup) (bool, error) {
	var backupConfigMap = &corev1.ConfigMap{}
	err := c.client.Get(ctx, types.NamespacedName{Namespace: backup.Namespace, Name: backupConfigMapName}, backupConfigMap)
	if err != nil {
		return false, fmt.Errorf("get backup operator config map '%s/%s': %w", backup.Namespace, backupConfigMapName, err)
	}

	backupRetryTimeLimitAsStr, ok := backupConfigMap.Data[backupRetryTimeLimitKey]
	if !ok {
		return false, fmt.Errorf("read key '%s' from backup operator config map '%s'", backupRetryTimeLimitKey, backupConfigMapName)
	}

	backupRetryTimeLimit, err := strconv.Atoi(backupConfigMap.Data[backupRetryTimeLimitKey])
	if err != nil {
		return false, fmt.Errorf("convert backup retry limit from string '%s' to int: %w", backupRetryTimeLimitAsStr, err)
	}

	return c.clock.Now().Sub(backup.CreationTimestamp.Time) > time.Duration(backupRetryTimeLimit)*time.Minute, nil
}

func (c *defaultReconciler) getProviderBackup(ctx context.Context, namespacedName types.NamespacedName) (*velerov1.Backup, error) {
	var veleroBackup = &velerov1.Backup{}
	err := c.client.Get(ctx, namespacedName, veleroBackup)
	if apierrors.IsNotFound(err) {
		return nil, nil
	}
	return veleroBackup, err
}

func isProviderBackupInProgress(backup *velerov1.Backup) bool {
	return slices.Contains(veleroBackupIsRunningPhases, backup.Status.Phase)
}

func hasProviderBackupFailed(backup *velerov1.Backup) bool {
	return slices.Contains(veleroBackupFailedPhases, backup.Status.Phase)
}

func hasProviderBackupSucceeded(backup *velerov1.Backup) bool {
	return backup.Status.Phase == velerov1.BackupPhaseCompleted
}

func debug(logger logr.Logger, message string) {
	logger.V(1).Info(message)
}

func hasBackupSucceededOrFailed(condition *metav1.Condition) bool {
	return condition != nil && (condition.Status == metav1.ConditionTrue || condition.Status == metav1.ConditionFalse)
}
