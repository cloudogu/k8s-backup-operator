package backup

import (
	"context"
	"time"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/k8s-backup-operator/pkg/metrics"
	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
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
	ensureProviderBackupDeleted(ctx context.Context, backup *backupv1.Backup, logger logr.Logger) (action, error)
	ensureVeleroStatusSynced(ctx context.Context, backup *backupv1.Backup, logger logr.Logger) (action, error)
	ensureCompletedBackupIsIgnored(ctx context.Context, backup *backupv1.Backup, logger logr.Logger) (action, error)
	ensureBackupSetup(ctx context.Context, backup *backupv1.Backup, logger logr.Logger) (action, error)
	ensureBackupIsCanceledAfterTimeWindowExpired(ctx context.Context, backup *backupv1.Backup, logger logr.Logger) (action, error)
	ensureBackupIsPrepared(ctx context.Context, backup *backupv1.Backup, logger logr.Logger) (action, error)
	ensureMaintenanceActivated(ctx context.Context, backup *backupv1.Backup, logger logr.Logger) (action, error)
	ensureProviderBackupCreated(ctx context.Context, backup *backupv1.Backup, logger logr.Logger) (action, error)
	ensureProviderBackupCompleted(ctx context.Context, backup *backupv1.Backup, logger logr.Logger) (action, error)
	ensureMaintenanceDeactivated(ctx context.Context, backup *backupv1.Backup, logger logr.Logger) (action, error)
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
	logger := log.FromContext(ctx)
	metrics.UpdateBackupReconcileTotalMetric()

	var backup = backupv1.Backup{}
	if err := c.client.Get(ctx, req.NamespacedName, &backup); err != nil {
		return reconcile.Result{}, client.IgnoreNotFound(err)
	}

	nextAction, err := c.reconciler.ensureProviderBackupDeleted(ctx, &backup, logger)
	if nextAction == Retry {
		return ctrl.Result{RequeueAfter: c.requeueAfter}, err
	}
	if nextAction == Abort {
		return ctrl.Result{}, err
	}

	nextAction, err = c.reconciler.ensureVeleroStatusSynced(ctx, &backup, logger)
	if nextAction == Retry {
		return ctrl.Result{RequeueAfter: c.requeueAfter}, err
	}
	if nextAction == Abort {
		return ctrl.Result{}, err
	}

	nextAction, err = c.reconciler.ensureCompletedBackupIsIgnored(ctx, &backup, logger)
	if nextAction == Retry {
		return ctrl.Result{RequeueAfter: c.requeueAfter}, err
	}
	if nextAction == Abort {
		return ctrl.Result{}, err
	}

	nextAction, err = c.reconciler.ensureBackupSetup(ctx, &backup, logger)
	if nextAction == Abort {
		return ctrl.Result{}, err
	}

	nextAction, err = c.reconciler.ensureBackupIsCanceledAfterTimeWindowExpired(ctx, &backup, logger)
	if nextAction == Abort {
		return ctrl.Result{}, err
	}

	nextAction, err = c.reconciler.ensureBackupIsPrepared(ctx, &backup, logger)
	if nextAction == Retry {
		return ctrl.Result{RequeueAfter: c.requeueAfter}, err
	}
	if nextAction == Abort {
		return ctrl.Result{}, err
	}

	nextAction, err = c.reconciler.ensureMaintenanceActivated(ctx, &backup, logger)
	if nextAction == Retry {
		return ctrl.Result{RequeueAfter: c.requeueAfter}, err
	}
	if nextAction == Abort {
		return ctrl.Result{}, err
	}

	nextAction, err = c.reconciler.ensureProviderBackupCreated(ctx, &backup, logger)
	if nextAction == Retry {
		return ctrl.Result{RequeueAfter: c.requeueAfter}, err
	}
	if nextAction == Abort {
		return ctrl.Result{}, err
	}

	nextAction, err = c.reconciler.ensureProviderBackupCompleted(ctx, &backup, logger)
	if nextAction == Retry {
		return ctrl.Result{RequeueAfter: c.requeueAfter}, err
	}
	if nextAction == Abort {
		return ctrl.Result{}, err
	}

	// Since this is the last step, the "Next" and "Abort" actions lead to the same return statement. So we don't need
	// to check the "Abort" action.
	nextAction, err = c.reconciler.ensureMaintenanceDeactivated(ctx, &backup, logger)
	if nextAction == Retry {
		return ctrl.Result{RequeueAfter: c.requeueAfter}, err
	}

	return ctrl.Result{}, err
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
