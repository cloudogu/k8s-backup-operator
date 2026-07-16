package backup

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"time"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	blueprintv3 "github.com/cloudogu/k8s-blueprint-lib/v3/api/v3"
	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var defaultRequeueAfterTime = 2 * time.Second

type service interface {
	reconcileBackup(context context.Context, backup *backupv1.Backup) error
	cancelBackup(context context.Context, backup *backupv1.Backup) error
	deleteBackup(context context.Context, backup *backupv1.Backup) error
}

type action int

const (
	Next action = iota
	Retry
	Abort
)

type reconciler interface {
	checkVeleroBackupStorage(ctx context.Context, backup *backupv1.Backup, namespace string, logger logr.Logger) (action, error)
	checkMaintenanceModeIsActive(ctx context.Context, backup *backupv1.Backup, namespace string, logger logr.Logger) (action, error)
	checkVeleroBackup(ctx context.Context, backup *backupv1.Backup, namespace string, logger logr.Logger) (action, error)
}

func NewController(client client.Client, reconciler reconciler) *Controller {
	return &Controller{
		client:     client,
		reconciler: reconciler,
	}
}

type Controller struct {
	client     client.Client
	service    service
	reconciler reconciler
}

func (c *Controller) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var backup = backupv1.Backup{}
	if err := c.client.Get(ctx, req.NamespacedName, &backup); err != nil {
		return reconcile.Result{}, client.IgnoreNotFound(err)
	}

	if !backup.DeletionTimestamp.IsZero() {
		err := c.service.deleteBackup(ctx, &backup)
		if err != nil {
			logger.Error(err, "Failed to delete provider backup", "namespace", req.NamespacedName.Namespace, "name", req.Name)
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	err := c.setupBackup(ctx, &backup, req.NamespacedName.Namespace, logger)
	if err != nil {
		return reconcile.Result{}, err
	}

	nextAction, err := c.reconciler.checkVeleroBackupStorage(ctx, &backup, req.NamespacedName.Namespace, logger)
	if nextAction == Retry {
		return ctrl.Result{RequeueAfter: defaultRequeueAfterTime}, err
	}
	if nextAction == Abort {
		return ctrl.Result{}, err
	}

	nextAction, err = c.reconciler.checkMaintenanceModeIsActive(ctx, &backup, req.NamespacedName.Namespace, logger)
	if nextAction == Retry {
		return ctrl.Result{RequeueAfter: defaultRequeueAfterTime}, err
	}
	if nextAction == Abort {
		return ctrl.Result{}, err
	}

	//TODO
	_, _ = c.reconciler.checkVeleroBackup(ctx, &backup, req.NamespacedName.Namespace, logger)

	return ctrl.Result{}, nil
}

func (c *Controller) setupBackup(ctx context.Context, backup *backupv1.Backup, namespace string, logger logr.Logger) error {
	if backup.Labels == nil {
		backup.Labels = make(map[string]string)
	}
	maps.Copy(backup.Labels, defaultLabels)

	controllerutil.AddFinalizer(backup, backupv1.BackupFinalizer)

	var blueprintList = blueprintv3.BlueprintList{}
	if err := c.client.List(ctx, &blueprintList, client.InNamespace(namespace)); err != nil {
		logger.Error(err, "failed to list blueprints")
		return client.IgnoreNotFound(err)
	}

	if len(blueprintList.Items) > 0 {
		blueprint := blueprintList.Items[0]

		if backup.Annotations == nil {
			backup.Annotations = make(map[string]string)
		}

		dogusAsJson, err := json.Marshal(blueprint.Spec.Blueprint.Dogus)
		if err != nil {
			return fmt.Errorf("marshal blueprint dogus to json: %w", err)
		}

		annotations := map[string]string{
			blueprintIdAnnotation:    blueprint.Spec.DisplayName,
			blueprintDogusAnnotation: string(dogusAsJson),
		}
		maps.Copy(backup.Annotations, annotations)
	}

	err := c.client.Update(ctx, backup)
	if err != nil {
		logger.Error(err, "failed to update backup to set labels and annotations")
		return err
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (c *Controller) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		For(&backupv1.Backup{}).
		Complete(c)
}
