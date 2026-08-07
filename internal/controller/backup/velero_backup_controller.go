package backup

import (
	"context"
	"fmt"

	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type veleroBackupReconciler interface {
	ensureBackupExists(ctx context.Context, veleroBackup *velerov1.Backup) error
	deleteBackupIfExists(ctx context.Context, key client.ObjectKey) error
}

// VeleroBackupController reconciles Velero backups with their cloudogu Backup counterpart.
type VeleroBackupController struct {
	client     client.Client
	reconciler veleroBackupReconciler
}

func NewVeleroBackupController(client client.Client, reconciler veleroBackupReconciler) *VeleroBackupController {
	return &VeleroBackupController{client: client, reconciler: reconciler}
}

func (c *VeleroBackupController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// get velero backup from request
	veleroBackup := &velerov1.Backup{}
	if err := c.client.Get(ctx, req.NamespacedName, veleroBackup); err != nil {
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, fmt.Errorf("get velero backup %s: %w", req.NamespacedName, err)
		}
		if err = c.reconciler.deleteBackupIfExists(ctx, req.NamespacedName); err != nil {
			return ctrl.Result{}, fmt.Errorf("delete cloudogu backup for deleted velero backup %s: %w", req.NamespacedName, err)
		}
		return ctrl.Result{}, nil
	}

	// check backup for velero backup
	if err := c.reconciler.ensureBackupExists(ctx, veleroBackup); err != nil {
		if apierrors.IsAlreadyExists(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf("ensure cloudogu backup for velero backup %s: %w", req.NamespacedName, err)
	}

	return ctrl.Result{}, nil
}

func (c *VeleroBackupController) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named("velero-backup-sync").
		For(&velerov1.Backup{}).
		Complete(c)
}
