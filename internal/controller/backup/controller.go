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

type reconciler interface {
	ensureBackupLeaseReleased(ctx context.Context, backup *backupv1.Backup) (*backupv1.Backup, stageOutcome)
	ensureProviderBackupDeleted(ctx context.Context, backup *backupv1.Backup) (*backupv1.Backup, stageOutcome)
	ensureOrphanedBackupDeleted(ctx context.Context, backup *backupv1.Backup) (*backupv1.Backup, stageOutcome)
	ensureVeleroStatusSynced(ctx context.Context, backup *backupv1.Backup) (*backupv1.Backup, stageOutcome)
	ensureConditionsInitialized(ctx context.Context, backup *backupv1.Backup) (*backupv1.Backup, stageOutcome)
	ensureBackupSetup(ctx context.Context, backup *backupv1.Backup) (*backupv1.Backup, stageOutcome)
	ensureBackupIsCanceledAfterTimeWindowExpired(ctx context.Context, backup *backupv1.Backup) (*backupv1.Backup, stageOutcome)
	ensureBackupIsPrepared(ctx context.Context, backup *backupv1.Backup) (*backupv1.Backup, stageOutcome)
	ensureActiveBackupLease(ctx context.Context, backup *backupv1.Backup) (*backupv1.Backup, stageOutcome)
	ensureMaintenanceActivated(ctx context.Context, backup *backupv1.Backup) (*backupv1.Backup, stageOutcome)
	ensureProviderBackupCreated(ctx context.Context, backup *backupv1.Backup) (*backupv1.Backup, stageOutcome)
	ensureProviderBackupCompleted(ctx context.Context, backup *backupv1.Backup) (*backupv1.Backup, stageOutcome)
	ensureMaintenanceDeactivated(ctx context.Context, backup *backupv1.Backup) (*backupv1.Backup, stageOutcome)
	ensureBackupRunCompleted(ctx context.Context, backup *backupv1.Backup) (*backupv1.Backup, stageOutcome)
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

	return runStages(ctx, &backup, c.requeueAfter, c.getStagesForOperation(requiredOperation(&backup))...)
}

// getStagesForOperation returns the stages this reconciliation has to run, in order.
func (c *Controller) getStagesForOperation(op operation) []stage {
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
		return []stage{
			c.reconciler.ensureOrphanedBackupDeleted,
		}
	case operationFinalize:
		return c.finalizeStages()
	default: // operationCreate
		return append([]stage{
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
func (c *Controller) finalizeStages() []stage {
	return append(c.cleanupStages(),
		c.reconciler.ensureBackupRunCompleted,
	)
}

// cleanupStages returns resources owned by an active backup run in release order.
func (c *Controller) cleanupStages() []stage {
	return []stage{
		c.reconciler.ensureMaintenanceDeactivated,
		c.reconciler.ensureBackupLeaseReleased,
	}
}

// SetupWithManager sets up the controller with the Manager.
//
// The event filter is deliberate: a Velero backup can already exist before
// its Backup CR does, so the CR can never own it, and there are no owned children to watch. Provider
// progress is therefore awaited by requeueing, not by events.
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
