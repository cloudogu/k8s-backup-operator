package restore

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/cloudogu/k8s-backup-operator/internal/provider/velero"
	"github.com/cloudogu/k8s-backup-operator/pkg/metrics"
	restoreprovider "github.com/cloudogu/k8s-backup-operator/pkg/provider"
	"github.com/cloudogu/k8s-registry-lib/repository"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	k8sv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
)

type operation string

const (
	operationCreate = operation("create")
	operationDelete = operation("delete")
	operationIgnore = operation("ignore")
)

const (
	maintenanceModeTitle = "Service temporary unavailable"
	maintenanceModeText  = "Restore in progress"
)

func NewRestoreReconciler(
	k8sClient k8sClient,
	recorder eventRecorder,
	namespace string,
	cleanup cleanupManager,
	scaleManager scaleManager,
) *restoreReconciler {
	return &restoreReconciler{
		k8sClient:             k8sClient,
		recorder:              recorder,
		namespace:             namespace,
		cleanup:               cleanup,
		scaleManager:          scaleManager,
		maintenanceModeSwitch: repository.NewMaintenanceModeAdapter("k8s-backup-operator", k8sClient, namespace),
	}
}

// restoreReconciler reconciles a Restore object
type restoreReconciler struct {
	k8sClient             k8sClient
	recorder              eventRecorder
	namespace             string
	cleanup               cleanupManager
	scaleManager          scaleManager
	maintenanceModeSwitch maintenanceModeSwitch
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.15.0/pkg/reconcile
func (r *restoreReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	metrics.UpdateRestoreReconcileTotalMetric()

	restore := &k8sv1.Restore{}
	err := r.k8sClient.Get(ctx, client.ObjectKey{Namespace: r.namespace, Name: req.Name}, restore)
	if err != nil {
		logger.Info(fmt.Sprintf("failed to get restore resource %s/%s: %s", r.namespace, req.Name, err))
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logger.Info(fmt.Sprintf("found restore resource %s", req.NamespacedName))

	switch requiredOperation(restore) {
	case operationDelete:
		return runStages(
			ctx,
			restore,
			r.ensureProviderRestoreDeleted,
			r.ensureDeletingStatus,
			r.ensureDeletionFinalized,
		)
	case operationIgnore:
		return runStages(
			ctx,
			restore,
			r.ensureLegacyConditionsMigrated,
		)
	case operationCreate:
		return runStages(
			ctx,
			restore,
			r.ensureLegacyConditionsMigrated,
			r.ensureConditionsInitialized,
			r.ensureMetadata,
			r.ensureProviderChildState,
			r.ensurePreparation,
			r.ensureProviderRestore,
			r.ensureProviderCompletion,
			r.ensureBackupsSynchronized,
			r.ensureWorkloadsRecovered,
		)
	default:
		return ctrl.Result{}, nil
	}
}

// requiredOperation decides what this reconciliation has to do. The decision is derived from the
// deletion timestamp and from the effective Successful condition.
func requiredOperation(restore *k8sv1.Restore) operation {
	if restore.DeletionTimestamp != nil && !restore.DeletionTimestamp.IsZero() {
		return operationDelete
	}

	if isTerminal(restore) {
		return operationIgnore
	}

	return operationCreate
}

// ensureConditionsInitialized makes a restore that is about to run observable: all conditions of the
// workflow are written as Unknown in one status write, before any destructive stage, so that a
// running restore shows the milestones it has not reached yet instead of hiding them. Only absent
// conditions are written, so a milestone a later stage resolved is never reset to Unknown.
func (r *restoreReconciler) ensureConditionsInitialized(ctx context.Context, restore *k8sv1.Restore) (*k8sv1.Restore, stageOutcome) {
	missing := missingWorkflowConditions(restore)
	if len(missing) == 0 {
		return restore, next()
	}

	initialized, err := newConditionUpdater(r.k8sClient).setConditions(ctx, restore, missing...)
	if err != nil {
		return restore, retryOnError(fmt.Errorf("failed to initialize the conditions of restore %s: %w", restore.Name, err))
	}

	// retry after defaultDelay is a fallback since the status write triggers an instant requeue anyway
	return initialized, retryAfter(defaultRequeueDelay)
}

// ensureLegacyConditionsMigrated persists the Successful condition derived from the
// status phase of a Restore created before conditions existed. A Restore that is being deleted is left alone:
// its outcome no longer matters and writing conditions would only fight the deletion.
func (r *restoreReconciler) ensureLegacyConditionsMigrated(ctx context.Context, restore *k8sv1.Restore) (*k8sv1.Restore, stageOutcome) {
	updater := newConditionUpdater(r.k8sClient)

	migrated, err := updater.setConditionsFromLegacyStatus(ctx, restore)
	if err != nil {
		return restore, retryOnError(err)
	}

	return migrated, next()
}

// ensureMetadata adds the finalizer and the labels of a Restore that is about to be worked on,
// in one no-op-aware write. A write ends the reconciliation and asks for another one, so that at most
// one mutating stage runs per invocation.
func (r *restoreReconciler) ensureMetadata(ctx context.Context, restore *k8sv1.Restore) (*k8sv1.Restore, stageOutcome) {
	written, err := ensureRestoreMetadata(ctx, r.k8sClient, restore)
	if err != nil {
		return restore, retryOnError(fmt.Errorf("failed to converge the metadata of restore %s: %w", restore.Name, err))
	}

	if written {
		// retry after defaultDelay is a fallback since the metadata write triggers an instant requeue anyway
		return restore, retryAfter(defaultRequeueDelay)
	}

	return restore, next()
}

// ensureProviderChildState reads the owned provider restore before any destructive preparation can
// run, which is the workflow's safety barrier: the crash window between child creation and parent
// status persistence must not repeat cleanup or scale-down. The stage never writes the child. An
// existing child that this Restore may not use is a terminal failure; an existing child that is ours
// means the preparation has already happened and must not run again.
func (r *restoreReconciler) ensureProviderChildState(ctx context.Context, restore *k8sv1.Restore) (*k8sv1.Restore, stageOutcome) {
	child, err := velero.GetRestore(ctx, r.k8sClient, restore)
	if err != nil {
		return restore, retryOnError(fmt.Errorf("failed to read the provider restore of restore %s: %w", restore.Name, err))
	}

	if child == nil {
		return restore, next()
	}

	if conflictErr := velero.CheckRestoreForConflicts(restore, child); conflictErr != nil {
		return r.failOnProviderChildConflict(ctx, restore, conflictErr)
	}

	// Our child already exists, so the preparation is done. Continue with next stage.
	log.FromContext(ctx).Info(fmt.Sprintf("provider restore of restore %s/%s already exists: skip preparation", r.namespace, restore.Name))

	return restore, next()
}

// failOnProviderChildConflict reports an existing provider restore that this Restore may not adopt as
// a terminal failure, before any preparation ran.
func (r *restoreReconciler) failOnProviderChildConflict(ctx context.Context, restore *k8sv1.Restore, conflictErr error) (*k8sv1.Restore, stageOutcome) {
	r.recorder.Event(restore, corev1.EventTypeWarning, k8sv1.ErrorOnCreateEventReason, conflictErr.Error())
	metrics.InitRestoreStatusMetrics(r.namespace, restore.Name, restore.Spec.BackupName)

	updated, err := newConditionUpdater(r.k8sClient).setConditions(ctx, restore,
		metav1.Condition{
			Type:    k8sv1.ConditionProviderRestoreSuccessful,
			Status:  metav1.ConditionFalse,
			Reason:  ReasonProviderRestoreConflict,
			Message: conflictErr.Error(),
		},
		metav1.Condition{
			Type:    k8sv1.ConditionSuccessful,
			Status:  metav1.ConditionFalse,
			Reason:  ReasonProviderRestoreConflict,
			Message: fmt.Sprintf("The restore was not started: %v", conflictErr),
		},
	)
	if err != nil {
		return restore, retryOnError(fmt.Errorf("failed to report the provider restore conflict of restore %s: %w", restore.Name, err))
	}

	metrics.UpdateRestoreStatusMetrics(r.namespace, restore.Name, restore.Spec.BackupName, k8sv1.RestoreStatusFailed) // NOSONAR -- legacy restore status compatibility

	return updated, abort()
}

// ensurePreparation runs the destructive preparation of the ecosystem — maintenance mode, scale-down
// and cleanup. Maintenance mode is activated best-effort: a restore is more valuable than the
// unavailability notice, so a failed activation is logged and the workflow continues.
func (r *restoreReconciler) ensurePreparation(ctx context.Context, restore *k8sv1.Restore) (*k8sv1.Restore, stageOutcome) {
	prepared, err := r.isAlreadyPrepared(ctx, restore)
	if err != nil {
		return restore, retryOnError(err)
	}
	if prepared {
		return restore, next()
	}

	// The provider is checked before anything is touched: the preparation is irreversible, so an
	// unready provider must not cost the ecosystem its availability for a restore that cannot start.
	_, err = restoreprovider.Get(ctx, restore, restore.Spec.Provider, restore.Namespace, r.recorder, r.k8sClient)
	if err != nil {
		return restore, retryOnError(fmt.Errorf("failed to get restore provider [%s]: %w", restore.Spec.Provider, err))
	}

	logger := log.FromContext(ctx)

	err = r.maintenanceModeSwitch.Activate(ctx, repository.MaintenanceModeDescription{Title: maintenanceModeTitle, Text: maintenanceModeText}, false)
	maintenanceMessage := "maintenance mode is active,"
	if err != nil {
		logger.Error(err, "The Maintenance mode could not be activated. Continuing anyways...")
		maintenanceMessage = ""
	}

	if err := r.scaleManager.ScaleDown(ctx); err != nil {
		return r.reportFailedPreparation(ctx, restore, fmt.Errorf("failed to scale down workloads before restore: %w", err))
	}

	if err := r.cleanup.Cleanup(ctx); err != nil {
		return r.reportFailedPreparation(ctx, restore, fmt.Errorf("failed to cleanup before restore: %w", err))
	}

	updated, err := newConditionUpdater(r.k8sClient).setConditions(ctx, restore, metav1.Condition{
		Type:    k8sv1.ConditionPrepared,
		Status:  metav1.ConditionTrue,
		Reason:  ReasonPreparationCompleted,
		Message: fmt.Sprintf("The ecosystem was prepared for the restore: %s workloads are scaled down and the resources to be restored are removed.", maintenanceMessage),
	})
	if err != nil {
		return restore, retryOnError(fmt.Errorf("failed to persist the preparation of restore %s: %w", restore.Name, err))
	}

	// retry after defaultDelay is a fallback since the status write triggers an instant requeue anyway
	return updated, retryAfter(defaultRequeueDelay)
}

// isAlreadyPrepared reports whether the destructive preparation must be skipped because it has
// already happened.
func (r *restoreReconciler) isAlreadyPrepared(ctx context.Context, restore *k8sv1.Restore) (bool, error) {
	if meta.IsStatusConditionTrue(restore.Status.Conditions, k8sv1.ConditionPrepared) {
		return true, nil
	}

	// The condition is not the single source of truth: If an owned child exists, the preparation already happened.
	child, err := velero.GetRestore(ctx, r.k8sClient, restore)
	if err != nil {
		return false, fmt.Errorf("failed to read the provider restore of restore %s: %w", restore.Name, err)
	}

	return child != nil && velero.IsOwnedRestore(restore, child), nil
}

// reportFailedPreparation records a failed preparation as Prepared=False and retries it with the controller-runtime backoff.
func (r *restoreReconciler) reportFailedPreparation(ctx context.Context, restore *k8sv1.Restore, preparationErr error) (*k8sv1.Restore, stageOutcome) {
	updated, err := newConditionUpdater(r.k8sClient).setConditions(ctx, restore, metav1.Condition{
		Type:    k8sv1.ConditionPrepared,
		Status:  metav1.ConditionFalse,
		Reason:  ReasonPreparationFailed,
		Message: fmt.Sprintf("The preparation of the ecosystem failed and is retried: %v", preparationErr),
	})
	if err != nil {
		preparationErr = errors.Join(preparationErr, fmt.Errorf("failed to report the failed preparation of restore %s: %w", restore.Name, err))
	}

	return updated, retryOnError(preparationErr)
}

// ensureProviderRestore starts the provider restore and returns without waiting for it, so no worker
// is occupied while the provider runs. Creating the child is idempotent, an existing owned child will be used
// instead of starting a second restore.
func (r *restoreReconciler) ensureProviderRestore(ctx context.Context, restore *k8sv1.Restore) (*k8sv1.Restore, stageOutcome) {
	// A provider restore already succeeded -> skip
	if meta.IsStatusConditionTrue(restore.Status.Conditions, k8sv1.ConditionProviderRestoreSuccessful) {
		return restore, next()
	}

	existing, err := velero.GetRestore(ctx, r.k8sClient, restore)
	if err != nil {
		return restore, retryOnError(fmt.Errorf("failed to read the provider restore of restore %s: %w", restore.Name, err))
	}
	if existing != nil {
		return restore, next()
	}

	//TODO: Think about this again, initializer increments the new metric on every retry
	metrics.InitRestoreStatusMetrics(r.namespace, restore.Name, restore.Spec.BackupName)
	r.recorder.Event(restore, corev1.EventTypeNormal, k8sv1.CreateEventReason, "Start restore process")

	if _, err := velero.EnsureRestore(ctx, r.k8sClient, restore); err != nil {
		// A conflict is not transient: another restore's child occupies the expected name, and no
		// number of retries will change that.
		var conflictErr *velero.ConflictError
		if errors.As(err, &conflictErr) {
			return r.failOnProviderChildConflict(ctx, restore, conflictErr)
		}

		r.recorder.Event(restore, corev1.EventTypeWarning, k8sv1.ErrorOnCreateEventReason, err.Error())

		return restore, retryOnError(fmt.Errorf("failed to start the provider restore of restore %s: %w", restore.Name, err))
	}

	metrics.UpdateRestoreStatusMetrics(r.namespace, restore.Name, restore.Spec.BackupName, k8sv1.RestoreStatusInProgress) // NOSONAR -- legacy restore status compatibility

	// The child's own events drive the next reconciliation; the delay is only the fallback.
	return restore, retryAfter(defaultRequeueDelay)
}

// ensureProviderCompletion observes the owned provider restore without blocking. If the restore is not
// terminated yet, stop reconciliation and wait for next change on the child resource.
// Only a provider success continues the workflow; a failure is terminal.
func (r *restoreReconciler) ensureProviderCompletion(ctx context.Context, restore *k8sv1.Restore) (*k8sv1.Restore, stageOutcome) {
	if meta.IsStatusConditionTrue(restore.Status.Conditions, k8sv1.ConditionProviderRestoreSuccessful) {
		return restore, next()
	}

	child, err := velero.GetRestore(ctx, r.k8sClient, restore)
	if err != nil {
		return restore, retryOnError(fmt.Errorf("failed to read the provider restore of restore %s: %w", restore.Name, err))
	}
	if child == nil {
		// The child vanished between the stages. There is nothing to observe, and the next
		// reconciliation lets the provider restore stage create it again.
		return restore, retryAfter(defaultRequeueDelay)
	}

	status, reason := observeProviderRestoreState(velero.ObserveRestorePhase(child.Status.Phase))
	if status == metav1.ConditionFalse {
		return r.failOnProviderRestore(ctx, restore, child, reason)
	}

	message := fmt.Sprintf("The provider restore %q reports the phase %q.", child.Name, child.Status.Phase)
	if status == metav1.ConditionTrue {
		message = fmt.Sprintf("The provider restored the backup %q.", restore.Spec.BackupName)
		r.recorder.Eventf(restore, corev1.EventTypeNormal, k8sv1.CreateEventReason, "Successfully completed the provider restore [%s]", child.Name)
	}

	// The write is no-op aware, so an unchanged state costs nothing; a changed one makes the phase the
	// restore is actually in visible in its status.
	updated, err := newConditionUpdater(r.k8sClient).setConditions(ctx, restore, metav1.Condition{
		Type:    k8sv1.ConditionProviderRestoreSuccessful,
		Status:  status,
		Reason:  reason,
		Message: message,
	})
	if err != nil {
		return restore, retryOnError(fmt.Errorf("failed to persist the provider restore state of restore %s: %w", restore.Name, err))
	}

	if status == metav1.ConditionTrue {
		// retry after defaultDelay is a fallback since the status write triggers an instant requeue anyway
		return updated, retryAfter(defaultRequeueDelay)
	}

	return updated, retryAfter(providerObservationRecoveryDelay)
}

// failOnProviderRestore reports a terminally failed provider restore.
func (r *restoreReconciler) failOnProviderRestore(ctx context.Context, restore *k8sv1.Restore, child *velerov1.Restore, reason string) (*k8sv1.Restore, stageOutcome) {
	message := fmt.Sprintf("The provider restore %q failed terminally in the phase %q.", child.Name, child.Status.Phase)
	r.recorder.Event(restore, corev1.EventTypeWarning, k8sv1.ErrorOnCreateEventReason, message)

	if err := r.maintenanceModeSwitch.Deactivate(ctx, false); err != nil {
		log.FromContext(ctx).Error(err, "The maintenance mode could not be deactivated after a failed provider restore. Continuing anyways...")
	}

	updated, err := newConditionUpdater(r.k8sClient).setConditions(ctx, restore,
		metav1.Condition{
			Type:    k8sv1.ConditionProviderRestoreSuccessful,
			Status:  metav1.ConditionFalse,
			Reason:  reason,
			Message: message,
		},
		metav1.Condition{
			Type:    k8sv1.ConditionWorkloadsRecovered,
			Status:  metav1.ConditionFalse,
			Reason:  ReasonRecoveryNotAttemptedAfterProviderFailure,
			Message: "The workloads were deliberately left scaled down because the provider restore failed.",
		},
		metav1.Condition{
			Type:    k8sv1.ConditionBackupsSynchronized,
			Status:  metav1.ConditionFalse,
			Reason:  ReasonSynchronizationNotAttemptedAfterProviderFailure,
			Message: "The backups were deliberately not synchronized because the provider restore failed.",
		},
		metav1.Condition{
			Type:    k8sv1.ConditionSuccessful,
			Status:  metav1.ConditionFalse,
			Reason:  reason,
			Message: message,
		},
	)
	if err != nil {
		return restore, retryOnError(fmt.Errorf("failed to report the failed provider restore of restore %s: %w", restore.Name, err))
	}

	metrics.UpdateRestoreStatusMetrics(r.namespace, restore.Name, restore.Spec.BackupName, k8sv1.RestoreStatusFailed) // NOSONAR -- legacy restore status compatibility

	return updated, abort()
}

// ensureBackupsSynchronized synchronizes the Backup resources with the ones the provider knows about
func (r *restoreReconciler) ensureBackupsSynchronized(ctx context.Context, restore *k8sv1.Restore) (*k8sv1.Restore, stageOutcome) {
	if meta.IsStatusConditionTrue(restore.Status.Conditions, k8sv1.ConditionBackupsSynchronized) {
		return restore, next()
	}

	provider, err := restoreprovider.Get(ctx, restore, restore.Spec.Provider, restore.Namespace, r.recorder, r.k8sClient)
	if err != nil {
		return restore, retryOnError(fmt.Errorf("failed to get restore provider [%s]: %w", restore.Spec.Provider, err))
	}

	if err := provider.SyncBackups(ctx); err != nil {
		return r.reportUnreachedMilestone(ctx, restore, k8sv1.ConditionBackupsSynchronized, ReasonBackupSynchronizationFailed,
			fmt.Errorf("failed to sync backups with provider: %w", err))
	}

	updated, err := newConditionUpdater(r.k8sClient).setConditions(ctx, restore,
		reachedMilestone(k8sv1.ConditionBackupsSynchronized, ReasonBackupSynchronizationCompleted,
			"The backup resources were synchronized with the provider."))
	if err != nil {
		return restore, retryOnError(fmt.Errorf("failed to persist the backup synchronization of restore %s: %w", restore.Name, err))
	}

	// retry after defaultDelay is a fallback since the status write triggers an instant requeue anyway
	return updated, retryAfter(defaultRequeueDelay)
}

// ensureWorkloadsRecovered ends the workflow: the workloads are scaled up again, the unavailability
// notice is taken down and the restore is reported as successful.
func (r *restoreReconciler) ensureWorkloadsRecovered(ctx context.Context, restore *k8sv1.Restore) (*k8sv1.Restore, stageOutcome) {
	if err := r.scaleManager.ScaleUp(ctx); err != nil {
		return r.reportUnreachedMilestone(ctx, restore, k8sv1.ConditionWorkloadsRecovered, ReasonWorkloadRecoveryFailed,
			fmt.Errorf("failed to scale up workloads after restore: %w", err))
	}

	// Taking maintenance mode down is best-effort, like the activation: the workloads are up
	// again, so a remaining notice is a nuisance and not a reason to fail a successful restore.
	if err := r.maintenanceModeSwitch.Deactivate(ctx, false); err != nil {
		return restore, retryOnError(fmt.Errorf("The maintenance mode could not be deactivated after the restore %s: %w", restore.Name, err))
	}

	// Both milestones are written together because they are reached together: the restore is
	// successful exactly because the workloads were recovered.
	updated, err := newConditionUpdater(r.k8sClient).setConditions(ctx, restore,
		reachedMilestone(k8sv1.ConditionWorkloadsRecovered, ReasonWorkloadRecoveryCompleted,
			"The workloads were scaled up again and the maintenance mode was switched off."),
		reachedMilestone(k8sv1.ConditionSuccessful, ReasonRestoreCompleted,
			"The restore workflow finished successfully."),
	)
	if err != nil {
		return restore, retryOnError(fmt.Errorf("failed to persist the workload recovery of restore %s: %w", restore.Name, err))
	}

	metrics.UpdateRestoreStatusMetrics(r.namespace, restore.Name, restore.Spec.BackupName, k8sv1.RestoreStatusCompleted) // NOSONAR -- legacy restore status compatibility
	r.recorder.Event(restore, corev1.EventTypeNormal, k8sv1.CreateEventReason, "Restore successful")

	// The restore is terminal now, so there is nothing left to do.
	return updated, abort()
}

// ensureDeletingStatus persists the deprecated scalar deleting status for consumers that have not
// migrated to deletion timestamps yet. The deletion timestamp remains the source of truth; this
// compatibility write is no-op-aware and ends the reconciliation when it changes the Restore.
func (r *restoreReconciler) ensureDeletingStatus(ctx context.Context, restore *k8sv1.Restore) (*k8sv1.Restore, stageOutcome) {
	// Check legacy state first
	if restore.Status.Status == k8sv1.RestoreStatusDeleting { // NOSONAR -- legacy restore status compatibility
		return restore, next()
	}

	// Re-Set conditions to synchronize state
	updated, err := newConditionUpdater(r.k8sClient).setConditions(ctx, restore)
	if err != nil {
		return restore, retryOnError(fmt.Errorf(
			"failed to persist deleting status of restore %s: %w",
			restore.Name,
			err,
		))
	}

	return updated, retryAfter(defaultRequeueDelay)
}

// ensureProviderRestoreDeleted removes the provider child before the parent is allowed to disappear.
// It only deletes an owned child, except for restores migrated from the legacy status model whose
// children predate owner references. A foreign namesake is left untouched and does not wedge deletion
// of the parent. An accepted child deletion is verified by a later reconciliation before continuing.
func (r *restoreReconciler) ensureProviderRestoreDeleted(ctx context.Context, restore *k8sv1.Restore) (*k8sv1.Restore, stageOutcome) {
	// Check if provider (velero for now) Restore exists
	child, err := velero.GetRestore(ctx, r.k8sClient, restore)
	if err != nil {
		// notfound is not an error - child would be nil in this case
		return restore, retryOnError(fmt.Errorf(
			"failed to get provider restore of restore %s: %w",
			restore.Name,
			err,
		))
	}

	// The provider restore is already deleted
	if child == nil {
		return restore, next()
	}

	successfulCondition := effectiveSuccessfulCondition(restore)
	isLegacyRestore := successfulCondition != nil &&
		successfulCondition.Reason == ReasonMigratedFromLegacyStatus

	// check if provider restore is owned by own restore
	if !velero.IsOwnedRestore(restore, child) && !isLegacyRestore {
		message := fmt.Sprintf(
			"Leaving provider restore [%s] untouched because it is not owned by this restore. "+
				"Remove it manually if it is not needed.",
			child.Name,
		)

		log.FromContext(ctx).Info(message)
		r.recorder.Event(
			restore,
			corev1.EventTypeWarning,
			k8sv1.DeleteEventReason,
			message,
		)

		// the foreign child does not prevent deletion of parent
		return restore, next()
	}

	// if the child is still deleting, we have to wait (requeue)
	if child.DeletionTimestamp != nil && !child.DeletionTimestamp.IsZero() {
		return restore, retryAfter(defaultRequeueDelay)
	}

	// trigger actual provider restore deletion
	if err := velero.DeleteRestore(ctx, r.k8sClient, child); err != nil {
		return restore, retryOnError(fmt.Errorf(
			"failed to delete provider restore of restore %s: %w",
			restore.Name,
			err,
		))
	}

	// Delete only acknowledges acceptance of the deletion request. Only a subsequent
	// Get can determine whether the child has actually been removed.
	return restore, retryAfter(defaultRequeueDelay)

}

// ensureDeletionFinalized releases the parent after every preceding deletion stage converged. Removing
// the controller finalizer lets Kubernetes complete the already requested deletion; an absent
// finalizer is treated as the desired state and no other finalizer is changed.
func (r *restoreReconciler) ensureDeletionFinalized(ctx context.Context, restore *k8sv1.Restore) (*k8sv1.Restore, stageOutcome) {
	// No-op on repeated reconciliations or if the finalizer was never added.
	if !controllerutil.ContainsFinalizer(restore, k8sv1.RestoreFinalizer) {
		return restore, abort()
	}

	// Remove finalizer to get the Restore deleted
	err := removeFinalizer(
		ctx,
		r.k8sClient,
		restore,
		k8sv1.RestoreFinalizer,
	)

	if err != nil {
		wrappedErr := fmt.Errorf(
			"failed to remove finalizer %s from restore %s: %w",
			k8sv1.RestoreFinalizer,
			restore.Name,
			err,
		)

		r.recorder.Event(
			restore,
			corev1.EventTypeWarning,
			k8sv1.DeleteEventReason,
			fmt.Sprintf("Delete failed. Reason: %s", wrappedErr),
		)

		return restore, retryOnError(wrappedErr)
	}

	r.recorder.Event(
		restore,
		corev1.EventTypeNormal,
		k8sv1.DeleteEventReason,
		"Delete successful",
	)

	// Removing the finalizer completes the workflow. If no other finalizers
	// remain, Kubernetes deletes the Restore afterwards.
	return restore, abort()
}

// reportUnreachedMilestone records a milestone the stage failed to reach as False and retries the
// stage with the controller-runtime backoff
func (r *restoreReconciler) reportUnreachedMilestone(ctx context.Context, restore *k8sv1.Restore, conditionType string, reason string, stageErr error) (*k8sv1.Restore, stageOutcome) {
	r.recorder.Event(restore, corev1.EventTypeWarning, k8sv1.ErrorOnCreateEventReason, stageErr.Error())

	updated, err := newConditionUpdater(r.k8sClient).setConditions(ctx, restore, metav1.Condition{
		Type:    conditionType,
		Status:  metav1.ConditionFalse,
		Reason:  reason,
		Message: fmt.Sprintf("The restore workflow did not reach this milestone and is retried: %v", stageErr),
	})
	if err != nil {
		stageErr = errors.Join(stageErr, fmt.Errorf("failed to report the unreached milestone %s of restore %s: %w", conditionType, restore.Name, err))
	}

	return updated, retryOnError(stageErr)
}

// performOperation executes the given operationFn, reports its outcome as an event and translates a
// failure into a retry. The stage outcome is the only requeue authority; there is no separate
// requeue orchestration writing the deprecated scalar status any more.
func (r *restoreReconciler) performOperation(
	ctx context.Context,
	restore *k8sv1.Restore,
	eventReason string,
	operationFn func(context.Context, *k8sv1.Restore) error,
) stageOutcome {
	logger := log.FromContext(ctx)

	operationError := operationFn(ctx, restore)
	eventType := corev1.EventTypeNormal
	message := fmt.Sprintf("%s successful", eventReason)
	if operationError != nil {
		eventType = corev1.EventTypeWarning
		printError := strings.ReplaceAll(operationError.Error(), "\n", "")
		message = fmt.Sprintf("%s failed. Reason: %s", eventReason, printError)
		logger.Error(operationError, message)
	}

	r.recorder.Event(restore, eventType, eventReason, message)

	if operationError != nil {
		return retryOnError(fmt.Errorf("%s of restore %s failed: %w", eventReason, restore.Name, operationError))
	}

	return abort()
}

// SetupWithManager sets up the controller with the Manager.
//
// Owns Velero restore resources to get notified when restore finishes or fails.
//
// There is deliberately no controller-wide event filter and no predicate on either source: a
// GenerationChangedPredicate would drop the provider restore's phase transitions and the parent's own
// status, finalizer and deletion events, all of which this level-triggered workflow reconciles on.
func (r *restoreReconciler) SetupWithManager(mgr controllerManager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8sv1.Restore{}).
		Owns(&velerov1.Restore{}).
		Complete(r)
}
