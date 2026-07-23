package schedule

import (
	"context"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type reconciler interface {
	checkCronJobSync(ctx context.Context, schedule *backupv1.BackupSchedule, namespace string, logger logr.Logger) (bool, error)
	markAsNotSyncedToCronJob(schedule *backupv1.BackupSchedule) error
	markAsSyncedToCronJob(schedule *backupv1.BackupSchedule) error
}

type Controller struct {
	client     client.Client
	reconciler reconciler
}

func (c *Controller) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var backupschedule = backupv1.BackupSchedule{}
	if err := c.client.Get(ctx, req.NamespacedName, &backupschedule); err != nil {
		return reconcile.Result{}, client.IgnoreNotFound(err)
	}

	isSynced, err := c.reconciler.checkCronJobSync(ctx, &backupschedule, req.NamespacedName.Namespace, logger)
	if isSynced {
		// check is ready
	} else {
		// sync
	}

	logger.Info("Reconcile ran")

	return ctrl.Result{}, err
}

func (c *Controller) doSomething(ctx context.Context, b *backupv1.BackupSchedule, namespace string, logger logr.Logger) error {

	return nil

}
