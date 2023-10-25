package restore

import (
	"context"
	"fmt"
	"github.com/cloudogu/k8s-backup-operator/pkg/requeue"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"strings"

	k8sv1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
)

type operation string

const (
	operationCreate = operation("create")
	operationDelete = operation("delete")
	operationIgnore = operation("ignore")
)

func NewRestoreReconciler(clientSet ecosystemInterface, recorder eventRecorder, namespace string, manager restoreManager, handler requeueHandler) *restoreReconciler {
	return &restoreReconciler{clientSet: clientSet, recorder: recorder, namespace: namespace, manager: manager, requeueHandler: handler}
}

// restoreReconciler reconciles a Restore object
type restoreReconciler struct {
	clientSet      ecosystemInterface
	recorder       eventRecorder
	namespace      string
	manager        restoreManager
	requeueHandler requeueHandler
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.15.0/pkg/reconcile
func (r *restoreReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	restore, err := r.clientSet.EcosystemV1Alpha1().Restores(r.namespace).Get(ctx, req.Name, metav1.GetOptions{})
	if err != nil {
		logger.Info(fmt.Sprintf("failed to get restore resource %s/%s: %s", r.namespace, req.Name, err))
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logger.Info(fmt.Sprintf("found restore resource %s", req.NamespacedName))

	requiredOperation := evaluateRequiredOperation(restore)
	logger.Info(fmt.Sprintf("required operation for restore %s is %s", req.NamespacedName, requiredOperation))

	switch requiredOperation {
	case operationCreate:
		return r.performCreateOperation(ctx, restore)
	case operationDelete:
		return r.performDeleteOperation(ctx, restore)
	case operationIgnore:
		return ctrl.Result{}, nil
	default:
		return ctrl.Result{}, fmt.Errorf("unknown operation: %s", requiredOperation)
	}
}

func evaluateRequiredOperation(restore *k8sv1.Restore) operation {
	if restore.DeletionTimestamp != nil && !restore.DeletionTimestamp.IsZero() {
		return operationDelete
	}

	switch restore.Status.Status {
	case k8sv1.RestoreStatusFailed:
		return operationIgnore
	case k8sv1.RestoreStatusNew:
		return operationCreate
	default:
		return operationIgnore
	}
}

func (r *restoreReconciler) performCreateOperation(ctx context.Context, restore *k8sv1.Restore) (ctrl.Result, error) {
	return r.performOperation(ctx, restore, k8sv1.CreateEventReason, k8sv1.RestoreStatusNew, r.manager.create)
}

func (r *restoreReconciler) performDeleteOperation(ctx context.Context, restore *k8sv1.Restore) (ctrl.Result, error) {
	return r.performOperation(ctx, restore, k8sv1.DeleteEventReason, restore.Status.Status, r.manager.delete)
}

// performOperation executes the given operationFn and requeues if necessary.
// When requeueing, the requeueStatus is set as the restore status.
func (r *restoreReconciler) performOperation(
	ctx context.Context,
	restore *k8sv1.Restore,
	eventReason string,
	requeueStatus string,
	operationFn func(context.Context, *k8sv1.Restore) error,
) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	operationError := operationFn(ctx, restore)
	contextMessageOnError := fmt.Sprintf("%s of restore %s failed", eventReason, restore.Name)
	eventType := corev1.EventTypeNormal
	message := fmt.Sprintf("%s successful", eventReason)
	if operationError != nil {
		eventType = corev1.EventTypeWarning
		printError := strings.ReplaceAll(operationError.Error(), "\n", "")
		message = fmt.Sprintf("%s failed. Reason: %s", eventReason, printError)
		logger.Error(operationError, message)
	}

	r.recorder.Event(restore, eventType, eventReason, message)

	result, handleErr := r.requeueHandler.Handle(ctx, contextMessageOnError, restore, operationError, requeueStatus)
	if handleErr != nil {
		r.recorder.Eventf(restore, corev1.EventTypeWarning, requeue.RequeueEventReason,
			"Failed to requeue the %s.", strings.ToLower(eventReason))
		return ctrl.Result{}, fmt.Errorf("failed to handle requeue: %w", handleErr)
	}

	return result, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *restoreReconciler) SetupWithManager(mgr controllerManager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8sv1.Restore{}).
		Complete(r)
}
