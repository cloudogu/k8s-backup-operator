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

	err := c.doSomething(ctx, &backupschedule, req.NamespacedName.Namespace, logger)

	logger.Info("Reconcile ran")

	return ctrl.Result{}, err
}

func (c *Controller) doSomething(ctx context.Context, b *backupv1.BackupSchedule, namespace string, logger logr.Logger) error {

	return nil

}
