package schedule

import (
	"context"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	batchv1 "k8s.io/api/batch/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type reconciler interface {
	Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error)
}

type Controller struct {
	client     client.Client
	reconciler reconciler
}

func (c *Controller) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return c.reconciler.Reconcile(ctx, req)
}

// SetupWithManager sets up the controller with the Manager and registers the watcher
func (c *Controller) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&backupv1.BackupSchedule{}).
		Owns(&batchv1.CronJob{}). // Watch CronJobs owned by BackupSchedule
		Complete(c)
}
