package backup

import (
	"context"
	"fmt"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

func NewReconciler(backupApi ecosystemBackupInterface, service Service, configGateway configGateway) *Reconciler {
	return &Reconciler{
		backupApi:     backupApi,
		service:       service,
		configGateway: configGateway,
	}
}

type Reconciler struct {
	backupApi     ecosystemBackupInterface
	service       Service
	configGateway configGateway
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	backup, err := r.backupApi.Get(ctx, req.Name, metav1.GetOptions{})
	if err != nil {
		logger.Info(fmt.Sprintf("failed to get backup resource %s/%s: %s", req.NamespacedName.Namespace, req.Name, err))
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if backup.DeletionTimestamp != nil && !backup.DeletionTimestamp.IsZero() {
		err = r.service.deleteBackup(ctx, Backup{Name: req.Name})
		if err != nil {
			logger.Error(err, "failed to delete provider backup", "namespace", req.NamespacedName.Namespace, "name", req.Name)
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, r.service.createBackup(ctx, Backup{Name: req.Name})
}

// SetupWithManager sets up the controller with the Manager.
func (r *Reconciler) SetupWithManager(mgr controllerManager) error {
	return builder.ControllerManagedBy(mgr).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		For(&backupv1.Backup{}).
		Complete(r)
}
