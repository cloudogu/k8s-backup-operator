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
	"github.com/cloudogu/k8s-backup-operator/internal/conditions"
	"github.com/cloudogu/k8s-backup-operator/internal/logging"
	veleroprovider "github.com/cloudogu/k8s-backup-operator/internal/provider/velero"
	blueprintv3 "github.com/cloudogu/k8s-blueprint-lib/v3/api/v3"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	reasonProviderBackupStorageLocationNotFound     = "ProviderBackupStorageLocationNotFound"
	reasonProviderBackupStorageLocationNotAvailable = "ProviderBackupStorageLocationNotAvailable"
	reasonProviderBackupStorageLocationAvailable    = "ProviderBackupStorageLocationAvailable"
	reasonMaintenanceModesIsNotActive               = "MaintenanceModesIsNotActive"
	reasonMaintenanceModesStatusFailed              = "MaintenanceModesStatusFailed"
	reasonMaintenanceModesActivation                = "MaintenanceModesActivation"
	reasonMaintenanceModesDeactivation              = "MaintenanceModesDeactivation"
	reasonMaintenanceModesActivationFailed          = "MaintenanceModesActivationFailed"
	reasonMaintenanceModesDeactivationFailed        = "MaintenanceModesDeactivationFailed"
	reasonProviderBackupResourceDoesNotExist        = "ProviderBackupResourceDoesNotExist"
	reasonProviderBackupInProgress                  = "ProviderBackupInProgress"
	reasonProviderBackupFailed                      = "ProviderBackupFailed"
	reasonProviderBackupDeletionFailed              = "ProviderBackupDeletionFailed"
	reasonProviderBackupDeletion                    = "ProviderBackupDeletion"
	reasonWaitingForProviderBackupCompletion        = "WaitingForProviderBackupCompletion"
	reasonProviderBackupSucceeded                   = "ProviderBackupSucceeded"
	reasonBackupStarted                             = "BackupStarted"
	reasonBackupDeleting                            = "BackupDeleting"
	reasonBackupSucceeded                           = "BackupSucceeded"
	reasonBackupDeletingFailed                      = "BackupDeletingFaild"
	reasonBackupNotDeleting                         = "BackupNotDeleting"
	reasonTimeWindowNotExpired                      = "TimeWindowNotExpired"
	reasonTimeWindowExpiredBackupNotStarted         = "TimeWindowExpiredBackupNotStarted"
	reasonTimeWindowExpiredBackupInProgress         = "TimeWindowExpiredBackupInProgress"
	reasonTimeWindowExpiredBackupFailed             = "TimeWindowExpiredBackupFailed"
	reasonTimeWindowExpiredBackupSucceeded          = "TimeWindowExpiredBackupSucceeded"
	reasonVeleroStatusSynced                        = "VeleroStatusSynced"
	reasonVeleroBackupRunning                       = "VeleroBackupRunning"
	reasonVeleroBackupFailed                        = "VeleroBackupFailed"
	reasonBackupLeaseAquired                        = "BackupLeaseAquired"
	reasonBackupLeaseFailed                         = "BackupLeaseFailed"
)

const (
	maintenanceModeTitle = "Service temporary unavailable"
	maintenanceModeText  = "Backup in progress"
)

const (
	backupConfigMapName     = "k8s-backup-operator-backup-config"
	backupRetryTimeLimitKey = "retryTimeLimit"
)

var defaultLabels = map[string]string{
	"app":                      "ces",
	"k8s.cloudogu.com/part-of": "backup",
}

var veleroBackupIsRunningPhases = []velerov1.BackupPhase{
	velerov1.BackupPhaseNew,
	velerov1.BackupPhaseInProgress,
	velerov1.BackupPhaseFinalizing,
	velerov1.BackupPhaseFinalizingPartiallyFailed,
	velerov1.BackupPhaseWaitingForPluginOperations,
	velerov1.BackupPhaseWaitingForPluginOperationsPartiallyFailed,
}

var veleroBackupFailedPhases = []velerov1.BackupPhase{
	velerov1.BackupPhaseFailedValidation,
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

type eventRecorder interface {
	record.EventRecorder
}

type defaultReconciler struct {
	client             client.Client
	recorder           eventRecorder
	maintenanceGateway maintenanceGateway
	clock              Clock
	backupStorageName  string
}

func NewReconciler(client client.Client, recorder eventRecorder, maintenanceGateway maintenanceGateway, clock Clock, backupStorageName string) *defaultReconciler {
	return &defaultReconciler{
		client:             client,
		recorder:           recorder,
		maintenanceGateway: maintenanceGateway,
		clock:              clock,
		backupStorageName:  backupStorageName,
	}
}

func (c *defaultReconciler) ensureProviderBackupDeleted(ctx context.Context, backup *backupv1.Backup) (action, error) {
	isBackupDeleting := !backup.DeletionTimestamp.IsZero()
	if isBackupDeleting {
		var veleroBackup, err = c.getProviderBackup(ctx, backup.GetNamespacedName())
		if err != nil {
			c.recorder.Event(backup, corev1.EventTypeWarning, reasonProviderBackupDeletionFailed, "Failed to get provider backup")
			return Abort, fmt.Errorf("get the velero backup resource to check if it exists: %w", err)
		}
		if veleroBackup == nil {
			logging.Debug(ctx, "ensureProviderBackupDeleted: backup is deleted and provider backup not found -> remove finalizer, ABORT")

			controllerutil.RemoveFinalizer(backup, backupv1.BackupFinalizer)
			updateErr := c.client.Update(ctx, backup)
			if updateErr != nil {
				c.recorder.Event(backup, corev1.EventTypeWarning, reasonBackupDeletingFailed, "Failed to remove backup finalizer")
				return Abort, fmt.Errorf("update backup after removing finalizer: %w", updateErr)
			}
			// Removing the finalizer releases the backup for deletion, so this is the last point at
			// which the deletion can be reported.
			logging.Info(ctx, "deleted the backup")
			// This is for the sake of completeness only - the object will be gone
			c.recorder.Event(backup, corev1.EventTypeNormal, reasonBackupDeleting, "Removed finalizer - deletion completed")
			return Abort, nil
		}

		if isProviderBackupInProgress(veleroBackup) {
			if cleanupErr := veleroprovider.DeleteVeleroDeleteBackupRequestIfExists(ctx, c.client, backup); cleanupErr != nil {
				return Abort, cleanupErr
			}

			patchErr := c.patchStatus(ctx, backup, func(status *backupv1.BackupStatus) {
				meta.SetStatusCondition(&status.Conditions, metav1.Condition{
					Type:    backupv1.ConditionDeleting,
					Status:  metav1.ConditionTrue,
					Reason:  reasonWaitingForProviderBackupCompletion,
					Message: fmt.Sprintf("Waiting for the provider backup to complete before deleting it (phase: %s)", veleroBackup.Status.Phase),
				})
			})
			if patchErr != nil {
				return Abort, fmt.Errorf("patch conditions while waiting for provider backup completion: %w", patchErr)
			}
			c.recorder.Event(backup, corev1.EventTypeNormal, reasonProviderBackupDeletion, "Waiting for the provider backup to complete")
			return Retry, nil
		}

		deleteReq, createErr := veleroprovider.CreateVeleroDeleteBackupRequestIfNotExists(ctx, c.client, backup)
		if createErr != nil {
			c.recorder.Event(backup, corev1.EventTypeWarning, reasonProviderBackupDeletionFailed, "Failed to create provider delete request")
			return Abort, createErr
		}
		waited := conditions.ElapsedInCurrentStatus(backup.Status.Conditions, backupv1.ConditionDeleting, c.clock.Now())
		deleting := metav1.Condition{
			Type:    backupv1.ConditionDeleting,
			Status:  metav1.ConditionTrue,
			Reason:  reasonBackupDeleting,
			Message: fmt.Sprintf("Backup is deleting (phase: %s, running for %s)", deleteReq.Status.Phase, conditions.FormatWaitDuration(waited)),
		}
		reportDeleting := conditions.WillChange(backup.Status.Conditions, deleting)

		patchErr := c.patchStatus(ctx, backup, func(status *backupv1.BackupStatus) {
			meta.SetStatusCondition(&status.Conditions, deleting)
		})
		if patchErr != nil {
			return Abort, fmt.Errorf("patch conditions to mark backup as deleting: %w", patchErr)
		}
		if reportDeleting {
			// The delete request carries the provider errors of a deletion that cannot finish. They
			// are reported here because they are otherwise only visible on the delete request itself.
			logging.Info(ctx, "waiting for the velero backup to be deleted",
				"phase", deleteReq.Status.Phase,
				"running for", conditions.FormatWaitDuration(waited),
				"providerErrors", deleteReq.Status.Errors,
			)
		}
		logging.Debug(ctx, "Retrying backup reconciliation", "reason", "the provider backup deletion is still in progress")
		c.recorder.Event(backup, corev1.EventTypeNormal, reasonProviderBackupDeletion, "The provider backup deletion is still in progress")
		return Retry, nil
	}

	logging.Debug(ctx, "ensureProviderBackupDeleted: backup is not deleted -> mark backup as not deleting, NEXT")

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

func (c *defaultReconciler) ensureCompletedBackupIsIgnored(ctx context.Context, backup *backupv1.Backup) (action, error) {
	succeededCondition := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionSucceeded)
	if succeededCondition == nil || succeededCondition.Status == metav1.ConditionUnknown {
		logging.Debug(ctx, "ensureCompletedBackupIsIgnored: backup not completed -> NEXT")
		return Next, nil
	}

	logging.Debug(ctx, "ensureCompletedBackupIsIgnored: backup completed -> ABORT")
	return Abort, nil
}

func (c *defaultReconciler) ensureBackupSetup(ctx context.Context, backup *backupv1.Backup) (action, error) {
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
		return Abort, fmt.Errorf("update backup to set labels, annotations and finalizer: %w", err)
	}

	logging.Info(ctx, "persisted backup labels, annotations and finalizer")
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

func (c *defaultReconciler) ensureBackupIsCanceledAfterTimeWindowExpired(ctx context.Context, backup *backupv1.Backup) (action, error) {
	timeWindowHasExpired, err := c.hasTimeWindowExpired(ctx, backup)
	if err != nil {
		return Abort, err
	}

	if !timeWindowHasExpired {
		logging.Debug(ctx, "ensureBackupIsCanceledAfterTimeWindowExpired: time window has not expired -> Canceled = False, NEXT")

		patchErr := c.patchStatus(ctx, backup, func(status *backupv1.BackupStatus) {
			meta.SetStatusCondition(&status.Conditions, metav1.Condition{
				Type:    backupv1.ConditionCanceled,
				Status:  metav1.ConditionFalse,
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
		return c.handleTimeWindowExpiredBackupNotStarted(ctx, backup)
	}

	return c.handleTimeWindowExpiredBackupStarted(ctx, backup)

}

func (c *defaultReconciler) ensureBackupIsPrepared(ctx context.Context, backup *backupv1.Backup) (action, error) {
	veleroBackupStorageLocation := velerov1.BackupStorageLocation{}
	err := c.client.Get(
		ctx,
		types.NamespacedName{Namespace: backup.Namespace, Name: c.backupStorageName},
		&veleroBackupStorageLocation,
	)

	if apierrors.IsNotFound(err) {
		return c.handleBackupStorageLocationNotFound(ctx, backup)
	}

	if err != nil {
		return Abort, fmt.Errorf("get velero backup storage location 'name=%s': %w", c.backupStorageName, err)
	}

	if veleroBackupStorageLocation.Status.Phase != velerov1.BackupStorageLocationPhaseAvailable {
		return c.handleBackupStorageLocationNotAvailable(ctx, backup, veleroBackupStorageLocation.Status.Phase)
	}

	return c.markBackupPrepared(ctx, backup)
}

func (c *defaultReconciler) handleBackupStorageLocationNotFound(ctx context.Context, backup *backupv1.Backup) (action, error) {
	logging.Debug(ctx, fmt.Sprintf(
		"ensureBackupIsPrepared: backup storage location '%s' not found -> Prepared = False, RETRY",
		c.backupStorageName,
	))

	notFound := metav1.Condition{
		Type:    backupv1.ConditionPrepared,
		Status:  metav1.ConditionFalse,
		Reason:  reasonProviderBackupStorageLocationNotFound,
		Message: "The provider backup storage location not found.",
	}
	reportNotFound := conditions.WillChange(backup.Status.Conditions, notFound)

	patchErr := c.patchStatus(ctx, backup, func(status *backupv1.BackupStatus) {
		meta.SetStatusCondition(&status.Conditions, notFound)
	})
	if patchErr != nil {
		return Abort, fmt.Errorf("patch conditions to mark preparation as failed: %w", patchErr)
	}
	if reportNotFound {
		logging.Info(ctx, "waiting for the velero backup storage location to appear", "backupStorageLocation", c.backupStorageName)
	}
	logging.Debug(ctx, "Retrying backup reconciliation", "reason", "the provider backup storage location was not found")
	c.recorder.Event(backup, corev1.EventTypeWarning, reasonProviderBackupStorageLocationNotFound, "The provider backup storage location not found")
	return Retry, nil
}

func (c *defaultReconciler) handleBackupStorageLocationNotAvailable(
	ctx context.Context,
	backup *backupv1.Backup,
	phase velerov1.BackupStorageLocationPhase,
) (action, error) {
	logging.Debug(ctx, "ensureBackupIsPrepared: backup storage location is not available -> Prepared = False, RETRY")

	notAvailable := metav1.Condition{
		Type:    backupv1.ConditionPrepared,
		Status:  metav1.ConditionFalse,
		Reason:  reasonProviderBackupStorageLocationNotAvailable,
		Message: fmt.Sprintf("velero backup storage location 'name=%s' is not available.", c.backupStorageName),
	}
	reportNotAvailable := conditions.WillChange(backup.Status.Conditions, notAvailable)

	patchErr := c.patchStatus(ctx, backup, func(status *backupv1.BackupStatus) {
		meta.SetStatusCondition(&status.Conditions, notAvailable)
	})
	if patchErr != nil {
		return Abort, fmt.Errorf("patch conditions to mark preparation as failed: %w", patchErr)
	}
	if reportNotAvailable {
		logging.Info(ctx, "waiting for the velero backup storage location to become available",
			"backupStorageLocation", c.backupStorageName,
			"phase", phase,
		)
	}
	logging.Debug(ctx, "Retrying backup reconciliation", "reason", "the provider backup storage location is not available")
	c.recorder.Event(backup, corev1.EventTypeWarning, reasonProviderBackupStorageLocationNotAvailable, "The provider backup storage location is not available")
	return Retry, nil
}

func (c *defaultReconciler) markBackupPrepared(ctx context.Context, backup *backupv1.Backup) (action, error) {
	prepared := metav1.Condition{
		Type:    backupv1.ConditionPrepared,
		Status:  metav1.ConditionTrue,
		Reason:  reasonProviderBackupStorageLocationAvailable,
		Message: fmt.Sprintf("velero backup storage location 'name=%s' is available.", c.backupStorageName),
	}
	reportPrepared := conditions.WillChange(backup.Status.Conditions, prepared)

	patchErr := c.patchStatus(ctx, backup, func(status *backupv1.BackupStatus) {
		meta.SetStatusCondition(&status.Conditions, prepared)
	})
	if patchErr != nil {
		return Abort, fmt.Errorf("patch status to mark the preparation conditions as failed: %w", patchErr)
	}

	logging.Debug(ctx, "ensureBackupIsPrepared: backup storage location is available -> Prepared = True, NEXT")
	if reportPrepared {
		logging.Info(ctx, "backup prepared", "backupStorageLocation", c.backupStorageName)
	}
	return Next, nil
}

func (c *defaultReconciler) ensureMaintenanceActivated(ctx context.Context, backup *backupv1.Backup) (action, error) {
	isActive, err := c.maintenanceGateway.isMaintenanceModeActive(ctx)
	if err != nil {
		c.recorder.Event(backup, corev1.EventTypeWarning, reasonMaintenanceModesStatusFailed, "Failed to get maintenance mode status")
		return Abort, fmt.Errorf("check if maintenance is active: %w", err)
	}

	if isActive {
		logging.Debug(ctx, "ensureMaintenanceActivated: is active -> NEXT")
		return Next, nil
	}

	succeededCondition := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionSucceeded)
	if succeededCondition != nil && succeededCondition.Status == metav1.ConditionTrue {
		logging.Debug(ctx, "ensureMaintenanceActivated: maintenance mode is not active and backup succeeded -> ABORT")
		return Abort, nil
	}
	if succeededCondition != nil && succeededCondition.Status == metav1.ConditionFalse {
		logging.Debug(ctx, "ensureMaintenanceActivated: maintenance mode is not active and backup failed -> ABORT")
		return Abort, nil
	}

	logging.Debug(ctx, "ensureMaintenanceActivated: maintenance mode is not active -> activate it; RETRY")

	maintenanceErr := c.maintenanceGateway.activateMaintenanceMode(ctx, maintenanceModeTitle, maintenanceModeText)
	if maintenanceErr != nil {
		c.recorder.Event(backup, corev1.EventTypeWarning, reasonMaintenanceModesActivationFailed, "Maintenance mode activation has failed")
		return Abort, fmt.Errorf("activate maintenance mode: %w", maintenanceErr)
	}
	logging.Info(ctx, "activated maintenance mode")

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
	logging.Debug(ctx, "Retrying backup reconciliation", "reason", "the maintenance mode was activated")
	c.recorder.Event(backup, corev1.EventTypeNormal, reasonMaintenanceModesActivation, "Maintenance mode was activated")
	return Retry, nil
}

func (c *defaultReconciler) ensureProviderBackupCreated(ctx context.Context, backup *backupv1.Backup) (action, error) {
	var veleroBackup, err = c.getProviderBackup(ctx, backup.GetNamespacedName())
	if err != nil {
		return Abort, fmt.Errorf("get velero backup resource: %w", err)
	}

	if veleroBackup == nil {
		logging.Debug(ctx, "ensureProviderBackupCreated: provider backup not found -> Succeeded = Unknown, RETRY")

		veleroBackupCr := veleroprovider.CreateVeleroBackupResource(backup, c.backupStorageName, defaultLabels)
		createErr := c.client.Create(ctx, veleroBackupCr)
		if createErr != nil {
			return Abort, fmt.Errorf("create velero backup resource: %w", createErr)
		}
		logging.Info(ctx, "created the velero backup", "backupStorageLocation", c.backupStorageName)

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

		c.recorder.Event(backup, corev1.EventTypeNormal, reasonProviderBackupInProgress, "Provider backup started")
		logging.Debug(ctx, "Retrying backup reconciliation", "reason", "the provider backup was created")
		return Retry, nil
	}

	logging.Debug(ctx, "ensureProviderBackupCreated: provider backup resource found -> NEXT")
	return Next, nil
}

func (c *defaultReconciler) ensureProviderBackupCompleted(ctx context.Context, backup *backupv1.Backup) (action, error) {
	var providerBackup, err = c.getProviderBackup(ctx, backup.GetNamespacedName())
	if err != nil || providerBackup == nil {
		return Abort, fmt.Errorf("checking velero backup resource for completion: %w", err)
	}

	if isProviderBackupInProgress(providerBackup) {
		logging.Debug(ctx, "ensureProviderBackupCompleted: provider backup is in progress -> RETRY")

		// The elapsed time is part of the message, so the message changes once per minute and the
		// guard below turns into a heartbeat for a wait that spans many reconciliations.
		waited := conditions.ElapsedInCurrentStatus(backup.Status.Conditions, backupv1.ConditionSucceeded, c.clock.Now())
		inProgress := metav1.Condition{
			Type:    backupv1.ConditionSucceeded,
			Status:  metav1.ConditionUnknown,
			Reason:  reasonProviderBackupInProgress,
			Message: fmt.Sprintf("Provider backup is in progress (running for %s).", conditions.FormatWaitDuration(waited)),
		}
		reportProgress := conditions.WillChange(backup.Status.Conditions, inProgress)

		patchErr := c.patchStatus(ctx, backup, func(status *backupv1.BackupStatus) {
			meta.SetStatusCondition(&status.Conditions, inProgress)
		})
		if patchErr != nil {
			return Abort, fmt.Errorf("patch status of backup resource: %w", patchErr)
		}
		if reportProgress {
			logging.Info(ctx, "waiting for the velero backup to complete",
				"phase", providerBackup.Status.Phase,
				"running for", conditions.FormatWaitDuration(waited),
			)
		}
		logging.Debug(ctx, "Retrying backup reconciliation", "reason", "the provider backup is still in progress")
		return Retry, nil
	}

	if hasProviderBackupFailed(providerBackup) {
		logging.Debug(ctx, "ensureProviderBackupCompleted: provider backup has failed -> ABORT")
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
		// Velero rejecting or failing a backup is an expected outcome of this stage rather than an
		// operator error, so it is reported without an error.
		logging.Info(ctx, "the velero backup failed", "phase", providerBackup.Status.Phase)
		return Next, nil
	}

	if hasProviderBackupSucceeded(providerBackup) {
		logging.Debug(ctx, "ensureProviderBackupCompleted: provider backup has succeeded -> NEXT")
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
		logging.Info(ctx, "the velero backup succeeded", "phase", providerBackup.Status.Phase)
		c.recorder.Event(backup, corev1.EventTypeNormal, reasonProviderBackupSucceeded, "Provider Backup completed")
		return Next, nil
	}

	return Abort, fmt.Errorf("provider backup with status='%s' should not occur here", providerBackup.Status.Phase)
}

func (c *defaultReconciler) ensureMaintenanceDeactivated(ctx context.Context, backup *backupv1.Backup) (action, error) {
	maintenanceModeIsActive, err := c.maintenanceGateway.isMaintenanceModeActive(ctx)
	if err != nil {
		c.recorder.Event(backup, corev1.EventTypeWarning, reasonMaintenanceModesStatusFailed, "Failed to get maintenance mode status")
		return Abort, fmt.Errorf("check maintenance mode: %w", err)
	}

	if !maintenanceModeIsActive {
		logging.Debug(ctx, "ensureMaintenanceDeactivated: is not active -> Next")
		return Next, nil
	}

	succeededCondition := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionSucceeded)
	if hasBackupSucceededOrFailed(succeededCondition) {
		logging.Debug(ctx, "ensureMaintenanceDeactivated: is active and backup succeeded or failed -> deactivate it,  Retry")

		maintenanceErr := c.maintenanceGateway.deactivateMaintenanceMode(ctx)
		if maintenanceErr != nil {
			c.recorder.Event(backup, corev1.EventTypeWarning, reasonMaintenanceModesDeactivationFailed, "Failed to deactivate maintenance mode")
			return Abort, fmt.Errorf("deactivate maintenance mode: %w", maintenanceErr)
		}
		logging.Info(ctx, "deactivated maintenance mode")
		logging.Debug(ctx, "Retrying backup reconciliation", "reason", "the maintenance mode was deactivated")
		c.recorder.Event(backup, corev1.EventTypeNormal, reasonMaintenanceModesDeactivation, "Maintenance mode deactivated")
		return Retry, nil
	}

	logging.Debug(ctx, "ensureMaintenanceDeactivated: is active and backup in progress -> Next")
	return Next, nil
}

func (c *defaultReconciler) patchStatus(ctx context.Context, backup *backupv1.Backup, updateFn statusUpdate) error {
	return newConditionsUpdater(c.client).updateStatus(ctx, backup, updateFn)
}

func (c *defaultReconciler) ensureVeleroStatusSynced(
	ctx context.Context,
	backup *backupv1.Backup,
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
	succeeded := metav1.Condition{
		Type:   backupv1.ConditionSucceeded,
		Status: metav1.ConditionUnknown,
	}

	nextAction := Retry
	report := ""
	switch {
	case veleroBackup.Status.Phase == velerov1.BackupPhaseCompleted:
		succeeded.Status = metav1.ConditionTrue
		succeeded.Reason = reasonVeleroStatusSynced
		succeeded.Message = "The succeeded backup status was synchronized from Velero."
		report = "synchronized the succeeded velero backup status"
		nextAction = Abort
	case slices.Contains(veleroBackupFailedPhases, veleroBackup.Status.Phase):
		succeeded.Status = metav1.ConditionFalse
		succeeded.Reason = reasonVeleroBackupFailed
		succeeded.Message = fmt.Sprintf("The Velero backup failed in phase %s.", veleroBackup.Status.Phase)
		report = "synchronized the failed velero backup status"
		nextAction = Abort
	default:
		succeeded.Reason = reasonVeleroBackupRunning
		succeeded.Message = fmt.Sprintf("The Velero backup is in phase %s.", veleroBackup.Status.Phase)
		report = "waiting for the synchronized velero backup to complete"
	}

	reportSync := conditions.WillChange(backup.Status.Conditions, succeeded)

	if err := c.patchStatus(ctx, backup, func(status *backupv1.BackupStatus) {
		meta.SetStatusCondition(&status.Conditions, prepared)
		meta.SetStatusCondition(&status.Conditions, succeeded)

		if veleroBackup.Status.StartTimestamp != nil {
			status.StartTimestamp = *veleroBackup.Status.StartTimestamp
		}
		if veleroBackup.Status.CompletionTimestamp != nil {
			status.CompletionTimestamp = *veleroBackup.Status.CompletionTimestamp
		}
	}); err != nil {
		return Abort, fmt.Errorf("patch backup status synchronized from Velero: %w", err)
	}

	logging.Debug(ctx, fmt.Sprintf("synchronized backup status from Velero- Phase: %s", veleroBackup.Status.Phase))
	if reportSync {
		logging.Info(ctx, report, "phase", veleroBackup.Status.Phase)
	}
	if nextAction == Retry {
		logging.Debug(ctx, "Retrying backup reconciliation", "reason", "the synchronized provider backup is still in progress")
	}
	return nextAction, nil
}

func (c *defaultReconciler) handleTimeWindowExpiredBackupNotStarted(ctx context.Context, backup *backupv1.Backup) (action, error) {
	logging.Debug(ctx, "ensureBackupIsCanceledAfterTimeWindowExpired: time window has expired, Backup has not started -> Canceled = True, ABORT")

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
	logging.Info(ctx, "canceled the backup", "reason", "the time window expired before the backup started")
	c.recorder.Event(backup, corev1.EventTypeWarning, reasonTimeWindowExpiredBackupNotStarted, "The backup is being canceled - time window expired")
	return Abort, nil
}

func (c *defaultReconciler) handleTimeWindowExpiredBackupStarted(ctx context.Context, backup *backupv1.Backup) (action, error) {
	veleroBackup, err := c.getProviderBackup(ctx, backup.GetNamespacedName())
	if err != nil {
		return Abort, fmt.Errorf("get velero backup: %w", err)
	}

	logging.Debug(ctx,
		fmt.Sprintf(
			"ensureBackupIsCanceledAfterTimeWindowExpired: time window has expired, Backup is running (Velero backup phase: '%s')",
			veleroBackup.Status.Phase,
		),
	)

	if isProviderBackupInProgress(veleroBackup) {
		logging.Debug(ctx, "ensureBackupIsCanceledAfterTimeWindowExpired: time window has expired, Backup is running -> Canceled = False, NEXT")

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
		c.recorder.Event(backup, corev1.EventTypeNormal, reasonTimeWindowExpiredBackupInProgress, "The backup was running when the time window expired -> Continue")
		return Next, nil
	}

	if hasProviderBackupFailed(veleroBackup) {
		logging.Debug(ctx, "ensureBackupIsCanceledAfterTimeWindowExpired: time window has expired, Backup failed -> Canceled = True, ABORT")

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

		logging.Info(ctx, "canceled the backup", "reason", "the time window expired and the velero backup had failed", "phase", veleroBackup.Status.Phase)
		c.recorder.Event(backup, corev1.EventTypeWarning, reasonTimeWindowExpiredBackupFailed, "The backup had failed when the time window expired")
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

func hasBackupSucceededOrFailed(condition *metav1.Condition) bool {
	return condition != nil && (condition.Status == metav1.ConditionTrue || condition.Status == metav1.ConditionFalse)
}
