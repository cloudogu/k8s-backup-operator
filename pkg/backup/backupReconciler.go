package backup

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/cloudogu/k8s-backup-operator/pkg/api/ecosystem"
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
	clientSet      ecosystem.Interface
	recorder       eventRecorder
	namespace      string
	manager        backupControllerManager
	requeueHandler requeueHandler
}

// NewBackupReconciler creates a new instance of backupReconciler.
func NewBackupReconciler(clientSet ecosystemInterface, recorder eventRecorder, namespace string, manager backupControllerManager, handler requeueHandler) *backupReconciler {
	return &backupReconciler{clientSet: clientSet, recorder: recorder, namespace: namespace, manager: manager, requeueHandler: handler}
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
	if client.IgnoreNotFound(err) != nil {
		return ctrl.Result{}, fmt.Errorf("failed to get backup resource %s/%s: %w", r.namespace, req.Name, err)
	}

	logger.Info(fmt.Sprintf("found backup resource %s", req.NamespacedName))

	requiredOperation := evaluateRequiredOperation(backup)
	logger.Info(fmt.Sprintf("required operation for backup %s is %s", req.NamespacedName, requiredOperation))

	switch requiredOperation {
	case operationCreate:
		return ctrl.Result{}, r.manager.create(ctx, backup)
	case operationDelete:
		return r.performDeleteOperation(ctx, backup)
	case operationIgnore:
		return ctrl.Result{}, nil
	default:
		return ctrl.Result{}, fmt.Errorf("unknown operation: %s", requiredOperation)
	}
}

func (r *backupReconciler) performDeleteOperation(ctx context.Context, backup *k8sv1.Backup) (ctrl.Result, error) {
	return r.performOperation(ctx, backup, k8sv1.DeleteEventReason, k8sv1.BackupStatusCompleted, r.manager.delete)
}

// performOperation executes the given operationFn and requeues if necessary.
// When requeueing, the sourceComponentStatus is set as the backup status.
func (r *backupReconciler) performOperation(
	ctx context.Context,
	backup *k8sv1.Backup,
	eventReason string,
	requeueStatus string,
	operationFn func(context.Context, *k8sv1.Backup) error,
) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	operationError := operationFn(ctx, backup)
	contextMessageOnError := fmt.Sprintf("%s failed with backup %s", eventReason, backup.Name)
	eventType := corev1.EventTypeNormal
	message := fmt.Sprintf("%s successful", eventReason)
	if operationError != nil {
		eventType = corev1.EventTypeWarning
		printError := strings.ReplaceAll(operationError.Error(), "\n", "")
		message = fmt.Sprintf("%s failed. Reason: %s", eventReason, printError)
		logger.Error(operationError, message)
	}

	// on self-upgrade of the backup-operator this event might not get send, because the operator is already shutting down
	r.recorder.Event(backup, eventType, eventReason, message)

	result, handleErr := r.requeueHandler.Handle(ctx, contextMessageOnError, backup, operationError, requeueStatus)
	if handleErr != nil {
		r.recorder.Eventf(backup, corev1.EventTypeWarning, RequeueEventReason,
			"Failed to requeue the %s.", strings.ToLower(eventReason))
		return ctrl.Result{}, fmt.Errorf("failed to handle requeue: %w", handleErr)
	}

	return result, nil
}

func evaluateRequiredOperation(backup *k8sv1.Backup) operation {
	if backup.Status.Status == k8sv1.BackupStatusFailed {
		return operationIgnore
	}

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
