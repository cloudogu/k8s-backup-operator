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

func NewReconciler(backupApi ecosystemBackupInterface, backupProvider backupProvider) *Reconciler {
	return &Reconciler{
		backupApi:      backupApi,
		backupProvider: backupProvider,
	}
}

type Reconciler struct {
	backupApi      ecosystemBackupInterface
	backupProvider backupProvider
}

/*
func (b *DefaultBackup) create(ctx context.Context, backup *backupv1.Backup) (ctrl.Result, error) {
	// Update Conditions
	// add Finalizer
	// add Labels
	// add Annotations
	// maintenance on

	return b.backup.create(ctx, backup)

	//return ctrl.Result{}, nil
}


type VeleroBackup struct {
}

func (b *VeleroBackup) create(ctx context.Context, backup *backupv1.Backup) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}
*/

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	backup, err := r.backupApi.Get(ctx, req.Name, metav1.GetOptions{})
	if err != nil {
		logger.Info(fmt.Sprintf("failed to get backup resource %s/%s: %s", req.NamespacedName.Namespace, req.Name, err))
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if backup.DeletionTimestamp != nil && !backup.DeletionTimestamp.IsZero() {
		err = r.backupProvider.DeleteBackup(ctx, backup)
		if err != nil {
			logger.Error(err, "failed to delete provider backup", "namespace", req.NamespacedName.Namespace, "name", req.Name)
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	err = r.backupProvider.CreateBackup(ctx, backup)
	if err != nil {
		logger.Error(err, "failed to create provider backup", "namespace", req.NamespacedName.Namespace, "name", req.Name)
		return ctrl.Result{}, nil
	}

	// err := r.client.Get(ctx, req.NamespacedName, ...)

	// backup not found -> provider.Delete VeleroBackup

	// provider.exists(namespace, name) bool
	// if not -> Create Velero Backup
	// else Do Nothing

	// backup is Deleting -> provider.Delete(namespace, name)

	// action =
	// switch action
	//   return r.backup.create()
	//	 return r.backup.delete()

	//veleroBackup := &velerov1.Backup{}
	// get

	//return r.actionMapper.handle(ctx, backup)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *Reconciler) SetupWithManager(mgr controllerManager) error {
	return builder.ControllerManagedBy(mgr).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		For(&backupv1.Backup{}).
		Complete(r)
}

/*
type RequestHandler interface {
	handle(ctx context.Context, backup *backupv1.Backup, veleroBackup *velerov1.Backup) (ctrl.Result, error)
}

*/
