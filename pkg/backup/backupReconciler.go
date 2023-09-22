package backup

import (
	"context"
	"fmt"
	"github.com/cloudogu/k8s-backup-operator/pkg/api/ecosystem"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	k8sv1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
)

type operation string

const (
	operationCreate = operation("create")
	operationDelete = operation("delete")
	operationIgnore = operation("ignore")
)

// backupReconciler reconciles a Backup object
type backupReconciler struct {
	clientSet ecosystem.Interface
	recorder  eventRecorder
	namespace string
	manager   backupControllerManager
}

// NewBackupReconciler creates a new instance of backupReconciler.
func NewBackupReconciler(clientSet ecosystemInterface, recorder eventRecorder, namespace string, manager backupControllerManager) *backupReconciler {
	return &backupReconciler{clientSet: clientSet, recorder: recorder, namespace: namespace, manager: manager}
}

// +kubebuilder:rbac:groups=k8s.cloudogu.com,resources=backups,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=k8s.cloudogu.com,resources=backups/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=k8s.cloudogu.com,resources=backups/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.15.0/pkg/reconcile
func (r *backupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	backup, err := r.clientSet.EcosystemV1Alpha1().Backups(r.namespace).Get(ctx, req.Name, metav1.GetOptions{})
	if err != nil {
		// TODO Throws not found on delete. Will be fixed with finalizers.
		return ctrl.Result{}, fmt.Errorf("failed to get backup resource %s: %w", req.NamespacedName, err)
	}

	logger.Info(fmt.Sprintf("found backup resource %s", req.NamespacedName))

	requiredOperation := evaluateRequiredOperation(backup)
	logger.Info(fmt.Sprintf("required operation for backup %s is %s", req.NamespacedName, requiredOperation))

	switch requiredOperation {
	case operationCreate:
		return ctrl.Result{}, r.manager.create(ctx, backup)
	case operationDelete:
		return ctrl.Result{}, r.manager.delete(ctx, backup)
	case operationIgnore:
		return ctrl.Result{}, nil
	default:
		return ctrl.Result{}, fmt.Errorf("unknown operation: %s", requiredOperation)
	}
}

func evaluateRequiredOperation(backup *k8sv1.Backup) operation {
	if backup.DeletionTimestamp != nil && !backup.DeletionTimestamp.IsZero() {
		return operationDelete
	}

	if backup.Status.Status == k8sv1.BackupStatusNew {
		return operationCreate
	}

	return operationIgnore
}

// SetupWithManager sets up the controller with the Manager.
func (r *backupReconciler) SetupWithManager(mgr controllerManager) error {
	return ctrl.NewControllerManagedBy(mgr).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		For(&k8sv1.Backup{}).
		Complete(r)
}
