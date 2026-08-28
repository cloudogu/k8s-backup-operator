package backup

import (
	"context"
	"time"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/k8s-backup-operator/internal/metrics"
	"k8s.io/apimachinery/pkg/api/meta"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type action int

const (
	Next action = iota
	Retry
	Abort
)

type reconciler interface {
	ensureBackupLeaseReleased(ctx context.Context, backup *backupv1.Backup) (action, error)
	ensureProviderBackupDeleted(ctx context.Context, backup *backupv1.Backup) (action, error)
	ensureOrphanedBackupDeleted(ctx context.Context, backup *backupv1.Backup) (action, error)
	ensureVeleroStatusSynced(ctx context.Context, backup *backupv1.Backup) (action, error)
	ensureConditionsInitialized(ctx context.Context, backup *backupv1.Backup) (action, error)
	ensureBackupSetup(ctx context.Context, backup *backupv1.Backup) (action, error)
	ensureBackupIsCanceledAfterTimeWindowExpired(ctx context.Context, backup *backupv1.Backup) (action, error)
	ensureBackupIsPrepared(ctx context.Context, backup *backupv1.Backup) (action, error)
	ensureActiveBackupLease(ctx context.Context, backup *backupv1.Backup) (action, error)
	ensureMaintenanceActivated(ctx context.Context, backup *backupv1.Backup) (action, error)
	ensureProviderBackupCreated(ctx context.Context, backup *backupv1.Backup) (action, error)
	ensureProviderBackupCompleted(ctx context.Context, backup *backupv1.Backup) (action, error)
	ensureMaintenanceDeactivated(ctx context.Context, backup *backupv1.Backup) (action, error)
	ensureBackupRunCompleted(ctx context.Context, backup *backupv1.Backup) (action, error)
}

type operation int

const (
	operationCreate operation = iota
	operationFinalize
	operationIgnore
	operationDelete
)

// requiredOperation decides what this reconciliation has to do. This way we keep the heavy gating
// out of most stages.
func requiredOperation(backup *backupv1.Backup) operation {
	if !backup.DeletionTimestamp.IsZero() {
		return operationDelete
	}
	if hasBackupSucceededOrFailed(meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionSucceeded)) {
		return operationIgnore
	}
	if isPostProcessing(backup) {
		return operationFinalize
	}
	return operationCreate
}

type ensureFunction func(ctx context.Context, backup *backupv1.Backup) (action, error)

func NewController(client client.Client, reconciler reconciler, requeueAfter time.Duration) *Controller {
	return &Controller{
		client:       client,
		reconciler:   reconciler,
		requeueAfter: requeueAfter,
	}
}

type Controller struct {
	client       client.Client
	reconciler   reconciler
	requeueAfter time.Duration
}

func (c *Controller) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	metrics.UpdateBackupReconcileTotalMetric()

	var backup = backupv1.Backup{}
	if err := c.client.Get(ctx, req.NamespacedName, &backup); err != nil {
		return reconcile.Result{}, client.IgnoreNotFound(err)
	}
	// Initialize all possible condition transition time series without incrementing them.
	// - no increment, just create if not exists
	metrics.InitBackupConditionTransitionMetrics(backup.Namespace, backup.Name)
	metrics.InitBackupStatusMetrics(backup.Namespace, backup.Name)
	metrics.InitInvalidLeaseTotalMetric(backup.Namespace, backup.Name)

	for _, ensure := range c.getStagesForOperation(requiredOperation(&backup)) {
		nextAction, err := ensure(ctx, &backup)
		switch nextAction {
		case Retry:
			return ctrl.Result{RequeueAfter: c.requeueAfter}, err
		case Abort:
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (c *Controller) getStagesForOperation(op operation) []ensureFunction {
	switch op {
	case operationDelete:
		// MaintenanceModeDeactivation needs lease, so comes first. Lease second, because
		// after ensureProviderBackupDeleted there is no backup anymore to release the lease on.
		return append(c.cleanupStages(),
			c.reconciler.ensureProviderBackupDeleted,
		)
	case operationIgnore:
		// Succeeded is written only after cleanup, so a terminal backup has no work left but to
		// verify that the provider backup it mirrors is still there.
		return []ensureFunction{
			c.reconciler.ensureOrphanedBackupDeleted,
		}
	case operationFinalize:
		return c.finalizeStages()
	default: // operationCreate
		return append([]ensureFunction{
			c.reconciler.ensureVeleroStatusSynced,
			c.reconciler.ensureConditionsInitialized,
			c.reconciler.ensureBackupSetup,
			c.reconciler.ensureBackupIsCanceledAfterTimeWindowExpired,
			c.reconciler.ensureBackupIsPrepared,
			c.reconciler.ensureActiveBackupLease,
			c.reconciler.ensureMaintenanceActivated,
			c.reconciler.ensureProviderBackupCreated,
			c.reconciler.ensureProviderBackupCompleted,
		}, c.finalizeStages()...)
	}
}

// finalizeStages post-process a finished backup run
func (c *Controller) finalizeStages() []ensureFunction {
	return append(c.cleanupStages(),
		c.reconciler.ensureBackupRunCompleted,
	)
}

// cleanupStages returns resources owned by an active backup run in release order.
func (c *Controller) cleanupStages() []ensureFunction {
	return []ensureFunction{
		c.reconciler.ensureMaintenanceDeactivated,
		c.reconciler.ensureBackupLeaseReleased,
	}
}

// SetupWithManager sets up the controller with the Manager.
func (c *Controller) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		WithEventFilter(predicate.Or(
			predicate.GenerationChangedPredicate{},
			predicate.Funcs{
				UpdateFunc: func(event event.UpdateEvent) bool {
					oldDeletionTimestamp := event.ObjectOld.GetDeletionTimestamp()
					newDeletionTimestamp := event.ObjectNew.GetDeletionTimestamp()

					return oldDeletionTimestamp == nil && newDeletionTimestamp != nil
				},
			},
		)).
		For(&backupv1.Backup{}).
		Complete(c)
}
