package restore

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cloudogu/k8s-backup-operator/pkg/metrics"
	"github.com/cloudogu/k8s-backup-operator/pkg/requeue"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	k8sv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
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
	metrics.UpdateRestoreReconcileTotalMetric()

	restore, err := r.clientSet.EcosystemV1Alpha1().Restores(r.namespace).Get(ctx, req.Name, metav1.GetOptions{})
	if err != nil {
		logger.Info(fmt.Sprintf("failed to get restore resource %s/%s: %s", r.namespace, req.Name, err))
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logger.Info(fmt.Sprintf("found restore resource %s", req.NamespacedName))

	return runStages(ctx, restore,
		r.ensureLegacyConditionsMigrated,
		r.ensureDeletionHandled,
		r.ensureRestoreCreated,
	)
}

// requiredOperation decides what this reconciliation has to do. The decision is derived from the
// deletion timestamp and from the effective Successful condition; the deprecated scalar status is
// only consulted to keep a legacy value this operator cannot interpret out of the workflow.
func requiredOperation(restore *k8sv1.Restore) operation {
	if restore.DeletionTimestamp != nil && !restore.DeletionTimestamp.IsZero() {
		return operationDelete
	}

	successful := effectiveSuccessfulCondition(restore)
	switch {
	case successful == nil && restore.Status.Status == k8sv1.RestoreStatusNew:
		return operationCreate
	case successful == nil:
		// An already started status must not start a destructive restore.
		return operationIgnore
	default:
		return operationIgnore
	}
}

// ensureLegacyConditionsMigrated persists the Successful condition derived from the
// status phase of a Restore created before conditions existed. A Restore that is being deleted is left alone:
// its outcome no longer matters and writing conditions would only fight the deletion.
func (r *restoreReconciler) ensureLegacyConditionsMigrated(ctx context.Context, restore *k8sv1.Restore) (*k8sv1.Restore, stageOutcome) {
	if requiredOperation(restore) == operationDelete {
		return restore, next()
	}

	updater := newConditionUpdater(r.clientSet.EcosystemV1Alpha1().Restores(r.namespace))

	migrated, err := updater.setConditionsFromLegacyStatus(ctx, restore)
	if err != nil {
		return restore, retryOnError(err)
	}

	return migrated, next()
}

func (r *restoreReconciler) ensureDeletionHandled(ctx context.Context, restore *k8sv1.Restore) (*k8sv1.Restore, stageOutcome) {
	if requiredOperation(restore) != operationDelete {
		return restore, next()
	}

	log.FromContext(ctx).Info(fmt.Sprintf("required operation for restore %s/%s is %s", r.namespace, restore.Name, operationDelete))

	return restore, r.performOperation(ctx, restore, k8sv1.DeleteEventReason, restore.Status.Status, r.manager.delete)
}

func (r *restoreReconciler) ensureRestoreCreated(ctx context.Context, restore *k8sv1.Restore) (*k8sv1.Restore, stageOutcome) {
	op := requiredOperation(restore)
	log.FromContext(ctx).Info(fmt.Sprintf("required operation for restore %s/%s is %s", r.namespace, restore.Name, op))

	if op != operationCreate {
		return restore, abort()
	}

	return restore, r.performOperation(ctx, restore, k8sv1.CreateEventReason, k8sv1.RestoreStatusNew, r.manager.create)
}

// performOperation executes the given operationFn and requeues if necessary.
// When requeueing, the requeueStatus is set as the restore status.
func (r *restoreReconciler) performOperation(
	ctx context.Context,
	restore *k8sv1.Restore,
	eventReason string,
	requeueStatus string,
	operationFn func(context.Context, *k8sv1.Restore) error,
) stageOutcome {
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
		return retryOnError(fmt.Errorf("failed to handle requeue: %w", handleErr))
	}

	if result.Requeue || result.RequeueAfter > 0 {
		// if RequeueAfter is 0, use 1 second - Requeue = true is deprecated, and RequeueAfter needs a positive value to requeue at all.
		return retryAfter(max(time.Second, result.RequeueAfter))
	}
	return abort()
}

// SetupWithManager sets up the controller with the Manager.
func (r *restoreReconciler) SetupWithManager(mgr controllerManager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8sv1.Restore{}).
		Complete(r)
}
