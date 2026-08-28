package schedule

import (
	"context"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/k8s-backup-operator/internal/config"
	"github.com/cloudogu/k8s-backup-operator/internal/logging"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type defaultReconciler struct {
	client   client.Client
	recorder eventRecorder

	metadata   metadataManager
	cronJobs   cronJobManager
	validator  validator
	conditions conditionManager
}

// newReconciler creates a new BackupSchedule reconciler instance.
func newReconciler(client client.Client, recorder eventRecorder, operatorImage string, imagePullSecrets []corev1.LocalObjectReference) *defaultReconciler {
	return &defaultReconciler{
		client:     client,
		recorder:   recorder,
		metadata:   defaultMetadataManager{client: client},
		conditions: defaultConditionManager{},
		validator:  defaultValidator{},
		cronJobs: defaultCronJobManager{
			Client:           client,
			recorder:         recorder,
			scheme:           client.Scheme(),
			operatorImage:    operatorImage,
			pullPolicy:       config.GetStagePullPolicy(),
			imagePullSecrets: imagePullSecrets,
		},
	}
}

// reconcile brings the requested BackupSchedule and its CronJob closer to their desired state as
// part of the reconcile loop
func (r *defaultReconciler) reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	logging.Debug(ctx, "starting BackupSchedule reconciliation")
	schedule, err := r.getBackupSchedule(ctx, req.NamespacedName)
	if err != nil {
		if apierrors.IsNotFound(err) {
			logging.Info(ctx, "BackupSchedule for reconcile no longer exists")
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	original := schedule.DeepCopy()

	if !schedule.DeletionTimestamp.IsZero() {
		logging.Info(ctx, "reconciling BackupSchedule: deletion")

		// set the deletion state before doing anything else
		// Removing the finalizer may delete the resource immediately
		// leading to an error when patching
		r.conditions.markNotReady(schedule, backupv1.ReasonDeleting, "BackupSchedule is being deleted.")
		if err := r.patchStatus(ctx, original, schedule); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, r.reconcileDelete(ctx, schedule)
	}

	reconcileErr := r.reconcileNormal(ctx, schedule)

	// Always update status if it changed.
	if err := r.patchStatus(ctx, original, schedule); err != nil {
		return ctrl.Result{}, err
	}

	if reconcileErr != nil {
		return ctrl.Result{}, reconcileErr
	}

	logging.Debug(ctx, "BackupSchedule reconciliation completed")
	return ctrl.Result{}, nil

}

// reconcileDelete deletes the BackupSchedule by first deleting the managed CronJob before removing the BackupSchedule finalizer
// which will allow Kubernetes to delete the BackupSchedule
func (r *defaultReconciler) reconcileDelete(ctx context.Context, schedule *backupv1.BackupSchedule) error {
	if err := r.cronJobs.delete(ctx, schedule); err != nil {
		return err
	}

	// only need to remove finalizers, not labels
	if err := r.metadata.remove(ctx, schedule); err != nil {
		r.recorder.Eventf(schedule, corev1.EventTypeWarning, backupv1.FinalizerRemovalFailedEventReason,
			"Failed to remove finalizer %q from BackupSchedule: %v", backupv1.BackupScheduleFinalizer, err,
		)
		return err
	}

	return nil
}

// reconcileNormal ensures metadata like labels and finalizers, validates that the schedule, is a valid cronjob schedule
// and synchronizes the managed CronJob to the BackupSchedule resource.
func (r *defaultReconciler) reconcileNormal(ctx context.Context, schedule *backupv1.BackupSchedule) error {

	logging.Debug(ctx, "BackupSchedule reconciliation for backupschedule")
	if err := r.metadata.ensure(ctx, schedule); err != nil {
		r.conditions.markAcceptanceNotEvaluated(schedule, err)
		r.conditions.markNotReady(schedule, backupv1.ReasonNotEvaluated, "Required metadata could not be persisted: "+err.Error())
		return err
	}

	if err := r.validator.validate(schedule); err != nil {
		r.conditions.markInvalid(schedule, err)
		r.conditions.markNotReady(schedule, backupv1.ReasonInvalidSpec, "BackupSchedule spec is invalid: "+err.Error())
		r.recorder.Eventf(
			schedule, corev1.EventTypeWarning, backupv1.InvalidScheduleEventReason,
			"BackupSchedule has an invalid schedule: %v", err,
		)
		logging.Error(ctx, err, "BackupSchedule spec is invalid, skipping CronJob synchronization")

		// invalid spec should not be reconciled again before it is edited
		return nil
	}

	r.conditions.markAccepted(schedule)

	if err := r.cronJobs.ensure(ctx, schedule); err != nil {
		r.conditions.markNotReady(schedule, backupv1.ReasonSyncFailed, "CronJob synchronization failed: "+err.Error())

		return err
	}

	r.conditions.markReady(schedule)

	return nil
}

// patchStatus persists the BackupSchedule status when it differs from the original status.
func (r *defaultReconciler) patchStatus(ctx context.Context, before, after *backupv1.BackupSchedule) error {

	if equality.Semantic.DeepEqual(before.Status, after.Status) {
		return nil
	}

	if err := r.client.Status().Patch(ctx, after, client.MergeFrom(before)); err != nil {
		return err
	}

	logging.Debug(ctx, "patched BackupSchedule status")
	return nil
}

// getBackupSchedule retrieves a BackupSchedule specified by name.
func (r *defaultReconciler) getBackupSchedule(ctx context.Context, name types.NamespacedName) (*backupv1.BackupSchedule, error) {
	schedule := &backupv1.BackupSchedule{}

	if err := r.client.Get(ctx, name, schedule); err != nil {
		return nil, err
	}

	return schedule, nil
}
