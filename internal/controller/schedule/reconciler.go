package schedule

import (
	"context"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"k8s.io/apimachinery/pkg/api/equality"
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

	metadata   MetadataManager
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

type MetadataManager interface {
	Ensure(ctx context.Context, schedule *backupv1.BackupSchedule) error
	Remove(ctx context.Context, schedule *backupv1.BackupSchedule) error
}

const (
	LabelApp    = "app"
	LabelPartOf = "k8s.cloudogu.com/part-of"

	LabelValueApp    = "ces"
	LabelValuePartOf = "backup"
)

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

	// only need to remove finalizers, not labels
	if err := r.metadata.Remove(ctx, schedule); err != nil {
		return err
	}

	return nil
}

func (r *defaultReconciler) reconcileNormal(ctx context.Context, schedule *backupv1.BackupSchedule) error {
	if err := r.metadata.Ensure(ctx, schedule); err != nil {
		r.conditions.MarkInvalid(schedule, err)
		r.conditions.MarkCronJobNotSynced(schedule, err)
		r.conditions.ComputeReady(schedule)
		return err
	}

	if err := r.validator.Validate(schedule); err != nil {
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

	// ready is computed from the other values
	r.conditions.ComputeReady(schedule)

	return nil
}

func (r *defaultReconciler) patchStatus(ctx context.Context, before, after *backupv1.BackupSchedule) error {
	if equality.Semantic.DeepEqual(before.Status, after.Status) {
		return nil
	}

	return r.client.Status().Patch(ctx, after, client.MergeFrom(before))
}

func (r *defaultReconciler) ensureMetadataSet(ctx context.Context, schedule *backupv1.BackupSchedule) error {
	before := schedule.DeepCopy()
	changed := false

	if !controllerutil.ContainsFinalizer(schedule, backupv1.BackupScheduleFinalizer) {
		controllerutil.AddFinalizer(schedule, backupv1.BackupScheduleFinalizer)
		changed = true
	}

	// no labels at all, initialize labels
	if schedule.Labels == nil {
		schedule.Labels = map[string]string{}
	}

	changed = addLabelsIfNecessary(schedule, changed)

	if !changed {
		return nil
	}

	return r.client.Patch(ctx, schedule, client.MergeFrom(before))
}

func addLabelsIfNecessary(schedule *backupv1.BackupSchedule, changed bool) bool {
	if schedule.Labels[LabelApp] != LabelValueApp {
		schedule.Labels[LabelApp] = LabelValueApp
		changed = true
	}

	if schedule.Labels[LabelPartOf] != LabelValuePartOf {
		schedule.Labels[LabelPartOf] = LabelValuePartOf
		changed = true
	}
	return changed
}

func (r *defaultReconciler) removeFinalizer(ctx context.Context, schedule *backupv1.BackupSchedule) error {

	if !controllerutil.ContainsFinalizer(schedule, backupv1.BackupScheduleFinalizer) {
		return nil
	}

	before := schedule.DeepCopy()

	controllerutil.RemoveFinalizer(schedule, backupv1.BackupScheduleFinalizer)

	return r.client.Patch(ctx, schedule, client.MergeFrom(before))
}

func (r *defaultReconciler) getBackupSchedule(ctx context.Context, name types.NamespacedName) (*backupv1.BackupSchedule, error) {
	schedule := &backupv1.BackupSchedule{}

	if err := r.client.Get(ctx, name, schedule); err != nil {
		return nil, err
	}

	return schedule, nil
}
