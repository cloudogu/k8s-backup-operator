package schedule

import (
	"context"
	"reflect"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// BackupScheduleReconciler handles reconciliation logic for BackupSchedule resources.
type BackupScheduleReconciler interface {
	markAsSyncedToCronJob(schedule *backupv1.BackupSchedule) error
}

type defaultReconciler struct {
	client client.Client

	cronJobs   CronJobManager
	validator  Validator
	conditions ConditionManager
}

type CronJobManager interface {
	Ensure(ctx context.Context, schedule *backupv1.BackupSchedule) error
	Delete(ctx context.Context, schedule *backupv1.BackupSchedule) error
}

type Validator interface {
	Validate(*backupv1.BackupSchedule) error
}

type ConditionManager interface {
	MarkAccepted(schedule *backupv1.BackupSchedule)
	MarkInvalid(schedule *backupv1.BackupSchedule, err error)
	MarkCronJobSynced(schedule *backupv1.BackupSchedule)
	MarkCronJobNotSynced(schedule *backupv1.BackupSchedule, err error)
	MarkDeleting(schedule *backupv1.BackupSchedule)
	ComputeReady(schedule *backupv1.BackupSchedule)
}

// NewReconciler creates a new BackupScheduleReconciler instance.
func NewReconciler(client client.Client) *defaultReconciler {
	return &defaultReconciler{
		client: client,
	}
}

func (r *defaultReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	schedule, err := r.getBackupSchedule(ctx, req.NamespacedName)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	original := schedule.DeepCopy()

	var reconcileErr error

	switch {
	// deletion
	case !schedule.DeletionTimestamp.IsZero():
		reconcileErr = r.reconcileDelete(ctx, schedule)
	// create/update
	default:
		reconcileErr = r.reconcileNormal(ctx, schedule)
	}

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
	r.conditions.MarkDeleting(schedule)

	if err := r.cronJobs.Delete(ctx, schedule); err != nil {
		return err
	}

	if _, err := r.removeFinalizer(ctx, schedule); err != nil {
		return err
	}

	return nil
}

func (r *defaultReconciler) reconcileNormal(ctx context.Context, schedule *backupv1.BackupSchedule) error {
	if _, err := r.ensureFinalizerSet(ctx, schedule); err != nil {
		return err
	}

	if err := r.validate(schedule); err != nil {
		r.conditions.MarkInvalid(schedule, err)
		r.conditions.MarkCronJobNotSynced(schedule, err)
		r.conditions.ComputeReady(schedule)

		// Spec is invalid.
		// Don't return an error because retrying won't help.
		return nil
	}

	r.conditions.MarkAccepted(schedule)

	if err := r.cronJobs.Ensure(ctx, schedule); err != nil {
		r.conditions.MarkCronJobNotSynced(schedule, err)
		r.conditions.ComputeReady(schedule)

		return err
	}

	r.conditions.MarkCronJobSynced(schedule)
	r.conditions.ComputeReady(schedule)

	return nil
}

func (r *defaultReconciler) patchStatus(ctx context.Context, before, after *backupv1.BackupSchedule) error {
	if reflect.DeepEqual(before.Status, after.Status) {
		return nil
	}

	return r.client.Status().Patch(ctx, after, client.MergeFrom(before))
}

func (r *defaultReconciler) ensureFinalizerSet(ctx context.Context, schedule *backupv1.BackupSchedule) (bool, error) {
	if controllerutil.ContainsFinalizer(schedule, backupv1.BackupScheduleFinalizer) {
		return false, nil

	}

	before := schedule.DeepCopy()

	controllerutil.AddFinalizer(schedule, backupv1.BackupScheduleFinalizer)

	return true, r.patchStatus(ctx, schedule, before)
}

func (r *defaultReconciler) removeFinalizer(ctx context.Context, schedule *backupv1.BackupSchedule) (bool, error) {

	if !controllerutil.ContainsFinalizer(schedule, backupv1.BackupScheduleFinalizer) {
		return false, nil
	}

	before := schedule.DeepCopy()

	controllerutil.RemoveFinalizer(schedule, backupv1.BackupScheduleFinalizer)

	return true, r.patchStatus(ctx, schedule, before)
}

func (r *defaultReconciler) validate(schedule *backupv1.BackupSchedule) error {
	return nil
}

func (r *defaultReconciler) getBackupSchedule(ctx context.Context, name types.NamespacedName) (*backupv1.BackupSchedule, error) {
	schedule := &backupv1.BackupSchedule{}

	if err := r.client.Get(ctx, name, schedule); err != nil {
		return nil, err
	}

	return schedule, nil
}
