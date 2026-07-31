package restore

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudogu/k8s-backup-operator/pkg/metrics"
	corev1 "k8s.io/api/core/v1"
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

func NewRestoreReconciler(k8sClient k8sClient, recorder eventRecorder, namespace string, manager restoreManager) *restoreReconciler {
	return &restoreReconciler{k8sClient: k8sClient, recorder: recorder, namespace: namespace, manager: manager}
}

// restoreReconciler reconciles a Restore object
type restoreReconciler struct {
	k8sClient k8sClient
	recorder  eventRecorder
	namespace string
	manager   restoreManager
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.15.0/pkg/reconcile
func (r *restoreReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	metrics.UpdateRestoreReconcileTotalMetric()

	restore := &k8sv1.Restore{}
	err := r.k8sClient.Get(ctx, client.ObjectKey{Namespace: r.namespace, Name: req.Name}, restore)
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

	updater := newConditionUpdater(r.k8sClient)

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

	return restore, r.performOperation(ctx, restore, k8sv1.DeleteEventReason, r.manager.delete)
}

func (r *restoreReconciler) ensureRestoreCreated(ctx context.Context, restore *k8sv1.Restore) (*k8sv1.Restore, stageOutcome) {
	op := requiredOperation(restore)
	log.FromContext(ctx).Info(fmt.Sprintf("required operation for restore %s/%s is %s", r.namespace, restore.Name, op))

	if op != operationCreate {
		return restore, abort()
	}

	return restore, r.performOperation(ctx, restore, k8sv1.CreateEventReason, r.manager.create)
}

// performOperation executes the given operationFn, reports its outcome as an event and translates a
// failure into a retry. The stage outcome is the only requeue authority; there is no separate
// requeue orchestration writing the deprecated scalar status any more.
func (r *restoreReconciler) performOperation(
	ctx context.Context,
	restore *k8sv1.Restore,
	eventReason string,
	operationFn func(context.Context, *k8sv1.Restore) error,
) stageOutcome {
	logger := log.FromContext(ctx)

	operationError := operationFn(ctx, restore)
	eventType := corev1.EventTypeNormal
	message := fmt.Sprintf("%s successful", eventReason)
	if operationError != nil {
		eventType = corev1.EventTypeWarning
		printError := strings.ReplaceAll(operationError.Error(), "\n", "")
		message = fmt.Sprintf("%s failed. Reason: %s", eventReason, printError)
		logger.Error(operationError, message)
	}

	r.recorder.Event(restore, eventType, eventReason, message)

	if operationError != nil {
		return retryOnError(fmt.Errorf("%s of restore %s failed: %w", eventReason, restore.Name, operationError))
	}

	return abort()
}

// SetupWithManager sets up the controller with the Manager.
func (r *restoreReconciler) SetupWithManager(mgr controllerManager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8sv1.Restore{}).
		Complete(r)
}
