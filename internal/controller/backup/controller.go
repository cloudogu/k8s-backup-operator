package backup

import (
	"context"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	blueprintv3 "github.com/cloudogu/k8s-blueprint-lib/v3/api/v3"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type service interface {
	configureBackup(context context.Context, backup *backupv1.Backup)
	addBlueprintAnnotation(context context.Context, backup *backupv1.Backup, displayName string, dogus []blueprintv3.Dogu) error
	reconcileBackup(context context.Context, backup *backupv1.Backup) error
	cancelBackup(context context.Context, backup *backupv1.Backup) error
	deleteBackup(context context.Context, backup *backupv1.Backup) error
}

func NewController(client client.Client, service service) *Controller {
	return &Controller{
		client:  client,
		service: service,
	}
}

type Controller struct {
	client  client.Client
	service service
}

func (c *Controller) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var backup = backupv1.Backup{}
	if err := c.client.Get(ctx, req.NamespacedName, &backup); err != nil {
		return reconcile.Result{}, client.IgnoreNotFound(err)
	}

	if backup.DeletionTimestamp != nil && !backup.DeletionTimestamp.IsZero() {
		err := c.service.deleteBackup(ctx, &backup)
		if err != nil {
			logger.Error(err, "failed to delete provider backup", "namespace", req.NamespacedName.Namespace, "name", req.Name)
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	c.service.configureBackup(ctx, &backup)

	var blueprintList = blueprintv3.BlueprintList{}
	if err := c.client.List(ctx, &blueprintList, client.InNamespace(req.Namespace)); err != nil {
		logger.Error(err, "failed to list blueprints")
		return reconcile.Result{}, client.IgnoreNotFound(err)
	}

	if len(blueprintList.Items) > 0 {
		blueprint := blueprintList.Items[0]
		err := c.service.addBlueprintAnnotation(ctx, &backup, blueprint.Spec.DisplayName, blueprint.Spec.Blueprint.Dogus)
		if err != nil {
			logger.Error(err, "failed to add annotations for blueprint infos")
			return reconcile.Result{}, err
		}
	}

	err := c.client.Update(ctx, &backup)
	if err != nil {
		logger.Error(err, "failed to update backup to set labels and annotations")
		return reconcile.Result{}, err
	}

	return ctrl.Result{}, c.service.reconcileBackup(ctx, &backup)
}

// SetupWithManager sets up the controller with the Manager.
func (c *Controller) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		For(&backupv1.Backup{}).
		Complete(c)
}
