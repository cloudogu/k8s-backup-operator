package schedule

import (
	"context"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/k8s-backup-operator/pkg/config"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type defaultReconciler struct {
	client client.Client

	metadata   metadataManager
	cronJobs   cronJobManager
	validator  validator
	conditions conditionManager
}

// newReconciler creates a new BackupSchedule reconciler instance.
func newReconciler(client client.Client, operatorImage string, imagePullSecrets []corev1.LocalObjectReference) *defaultReconciler {
	return &defaultReconciler{
		client:     client,
		metadata:   defaultMetadataManager{client: client},
		conditions: defaultConditionManager{},
		validator:  defaultValidator{},
		cronJobs: defaultCronJobManager{
			Client:           client,
			scheme:           client.Scheme(),
			operatorImage:    operatorImage,
			pullPolicy:       config.GetStagePullPolicy(),
			imagePullSecrets: imagePullSecrets,
		},
	}
}

func (r *defaultReconciler) reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	schedule, err := r.getBackupSchedule(ctx, req.NamespacedName)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	original := schedule.DeepCopy()

	if !schedule.DeletionTimestamp.IsZero() {
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

	return ctrl.Result{}, nil

}

func (r *defaultReconciler) reconcileDelete(ctx context.Context, schedule *backupv1.BackupSchedule) error {
	if err := r.cronJobs.delete(ctx, schedule); err != nil {
		return err
	}

	// only need to remove finalizers, not labels
	if err := r.metadata.remove(ctx, schedule); err != nil {
		return err
	}

	return nil
}

func (r *defaultReconciler) reconcileNormal(ctx context.Context, schedule *backupv1.BackupSchedule) error {
	if err := r.metadata.ensure(ctx, schedule); err != nil {
		r.conditions.markAcceptanceNotEvaluated(schedule, err)
		r.conditions.markNotReady(schedule, backupv1.ReasonNotEvaluated, "Required metadata could not be persisted: "+err.Error())
		return err
	}

	if err := r.validator.validate(schedule); err != nil {
		r.conditions.markInvalid(schedule, err)
		r.conditions.markNotReady(schedule, backupv1.ReasonInvalidSpec, "BackupSchedule spec is invalid: "+err.Error())

		// Spec is invalid.
		// Don't return an error because retrying won't help.
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

func (r *defaultReconciler) patchStatus(ctx context.Context, before, after *backupv1.BackupSchedule) error {
	if equality.Semantic.DeepEqual(before.Status, after.Status) {
		return nil
	}

	return r.client.Status().Patch(ctx, after, client.MergeFrom(before))
}

func (r *defaultReconciler) getBackupSchedule(ctx context.Context, name types.NamespacedName) (*backupv1.BackupSchedule, error) {
	schedule := &backupv1.BackupSchedule{}

	if err := r.client.Get(ctx, name, schedule); err != nil {
		return nil, err
	}

	return schedule, nil
}
