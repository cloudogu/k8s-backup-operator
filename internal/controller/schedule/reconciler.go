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

	metadata   MetadataManager
	cronJobs   CronJobManager
	validator  Validator
	conditions ConditionManager
}

// NewReconciler creates a new BackupScheduleReconciler instance.
func NewReconciler(client client.Client, operatorImage string, imagePullSecrets []corev1.LocalObjectReference) *defaultReconciler {
	return &defaultReconciler{
		client:     client,
		metadata:   metadataManager{client: client},
		conditions: conditionManager{},
		validator:  validator{},
		cronJobs: cronJobManager{
			Client:           client,
			scheme:           client.Scheme(),
			operatorImage:    operatorImage,
			pullPolicy:       config.GetStagePullPolicy(),
			imagePullSecrets: imagePullSecrets,
		},
	}
}

func (r *defaultReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	schedule, err := r.getBackupSchedule(ctx, req.NamespacedName)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	original := schedule.DeepCopy()

	if !schedule.DeletionTimestamp.IsZero() {
		// set the deletion state before doing anything else
		// Removing the finalizer may delete the resource immediately
		// leading to an error when patching
		r.conditions.MarkNotReady(schedule, ReasonDeleting, "BackupSchedule is being deleted.")
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
	if err := r.cronJobs.Delete(ctx, schedule); err != nil {
		return err
	}

	// only need to remove finalizers, not labels
	if err := r.metadata.Remove(ctx, schedule); err != nil {
		return err
	}

	return nil
}

func (r *defaultReconciler) reconcileNormal(ctx context.Context, schedule *backupv1.BackupSchedule) error {
	if err := r.metadata.Ensure(ctx, schedule); err != nil {
		r.conditions.MarkAcceptanceNotEvaluated(schedule, err)
		r.conditions.MarkNotReady(schedule, ReasonNotEvaluated, "Required metadata could not be persisted: "+err.Error())
		return err
	}

	if err := r.validator.Validate(schedule); err != nil {
		r.conditions.MarkInvalid(schedule, err)
		r.conditions.MarkNotReady(schedule, ReasonInvalidSpec, "BackupSchedule spec is invalid: "+err.Error())

		// Spec is invalid.
		// Don't return an error because retrying won't help.
		return nil
	}

	r.conditions.MarkAccepted(schedule)

	if err := r.cronJobs.Ensure(ctx, schedule); err != nil {
		r.conditions.MarkNotReady(schedule, ReasonSyncFailed, "CronJob synchronization failed: "+err.Error())

		return err
	}

	r.conditions.MarkReady(schedule)

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
