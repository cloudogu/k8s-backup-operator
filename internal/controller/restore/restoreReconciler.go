package restore

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudogu/k8s-backup-operator/internal/provider/velero"
	"github.com/cloudogu/k8s-backup-operator/pkg/metrics"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
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
		r.ensureConditionsInitialized,
		r.ensureMetadata,
		r.ensureProviderChildState,
		r.ensureRestoreCreated,
	)
}

// requiredOperation decides what this reconciliation has to do. The decision is derived from the
// deletion timestamp and from the effective Successful condition.
func requiredOperation(restore *k8sv1.Restore) operation {
	if restore.DeletionTimestamp != nil && !restore.DeletionTimestamp.IsZero() {
		return operationDelete
	}

	if isTerminal(restore) {
		return operationIgnore
	}

	return operationCreate
}

// ensureConditionsInitialized makes a restore that is about to run observable: all conditions of the
// workflow are written as Unknown in one status write, before any destructive stage, so that a
// running restore shows the milestones it has not reached yet instead of hiding them. Only absent
// conditions are written, so a milestone a later stage resolved is never reset to Unknown.
func (r *restoreReconciler) ensureConditionsInitialized(ctx context.Context, restore *k8sv1.Restore) (*k8sv1.Restore, stageOutcome) {
	if requiredOperation(restore) != operationCreate {
		return restore, next()
	}

	missing := missingWorkflowConditions(restore)
	if len(missing) == 0 {
		return restore, next()
	}

	initialized, err := newConditionUpdater(r.k8sClient).setConditions(ctx, restore, missing...)
	if err != nil {
		return restore, retryOnError(fmt.Errorf("failed to initialize the conditions of restore %s: %w", restore.Name, err))
	}

	// retry after defaultDelay is a fallback since the status write triggers an instant requeue anyway
	return initialized, retryAfter(defaultRequeueDelay)
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

// ensureMetadata adds the finalizer and the labels of a Restore that is about to be worked on,
// in one no-op-aware write. A write ends the reconciliation and asks for another one, so that at most
// one mutating stage runs per invocation.
func (r *restoreReconciler) ensureMetadata(ctx context.Context, restore *k8sv1.Restore) (*k8sv1.Restore, stageOutcome) {
	if requiredOperation(restore) != operationCreate {
		return restore, next()
	}

	written, err := ensureRestoreMetadata(ctx, r.k8sClient, restore)
	if err != nil {
		return restore, retryOnError(fmt.Errorf("failed to converge the metadata of restore %s: %w", restore.Name, err))
	}

	if written {
		// retry after defaultDelay is a fallback since the metadata write triggers an instant requeue anyway
		return restore, retryAfter(defaultRequeueDelay)
	}

	return restore, next()
}

// ensureProviderChildState reads the owned provider restore before any destructive preparation can
// run, which is the workflow's safety barrier: the crash window between child creation and parent
// status persistence must not repeat cleanup or scale-down. The stage never writes the child. An
// existing child that this Restore may not use is a terminal failure; an existing child that is ours
// means the preparation has already happened and must not run again.
func (r *restoreReconciler) ensureProviderChildState(ctx context.Context, restore *k8sv1.Restore) (*k8sv1.Restore, stageOutcome) {
	if requiredOperation(restore) != operationCreate {
		return restore, next()
	}

	child, err := velero.GetRestore(ctx, r.k8sClient, restore)
	if err != nil {
		return restore, retryOnError(fmt.Errorf("failed to read the provider restore of restore %s: %w", restore.Name, err))
	}

	if child == nil {
		return restore, next()
	}

	if conflictErr := velero.CheckRestoreForConflicts(restore, child); conflictErr != nil {
		return r.failOnProviderChildConflict(ctx, restore, conflictErr)
	}

	// Our child already exists, so preparation is done and repeating it would destroy resources while
	// a provider restore is in flight. Observing the child and finishing the workflow from here is the
	// provider completion stage's job; until that stage exists this restore makes no further progress,
	// which is deliberately preferred over repeating cleanup.
	log.FromContext(ctx).Info(fmt.Sprintf("provider restore of restore %s/%s already exists: skip preparation", r.namespace, restore.Name))

	return restore, abort()
}

// failOnProviderChildConflict reports an existing provider restore that this Restore may not adopt as
// a terminal failure, before any preparation ran.
func (r *restoreReconciler) failOnProviderChildConflict(ctx context.Context, restore *k8sv1.Restore, conflictErr error) (*k8sv1.Restore, stageOutcome) {
	r.recorder.Event(restore, corev1.EventTypeWarning, k8sv1.ErrorOnCreateEventReason, conflictErr.Error())
	metrics.InitRestoreStatusMetrics(r.namespace, restore.Name, restore.Spec.BackupName)

	updated, err := newConditionUpdater(r.k8sClient).setConditions(ctx, restore,
		metav1.Condition{
			Type:    k8sv1.ConditionProviderRestoreSuccessful,
			Status:  metav1.ConditionFalse,
			Reason:  ReasonProviderRestoreConflict,
			Message: conflictErr.Error(),
		},
		metav1.Condition{
			Type:    k8sv1.ConditionSuccessful,
			Status:  metav1.ConditionFalse,
			Reason:  ReasonProviderRestoreConflict,
			Message: fmt.Sprintf("The restore was not started: %v", conflictErr),
		},
	)
	if err != nil {
		return restore, retryOnError(fmt.Errorf("failed to report the provider restore conflict of restore %s: %w", restore.Name, err))
	}

	metrics.UpdateRestoreStatusMetrics(r.namespace, restore.Name, restore.Spec.BackupName, k8sv1.RestoreStatusFailed)

	return updated, abort()
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
//
// Owns Velero restore resources to get notified when restore finishes or fails.
//
// There is deliberately no controller-wide event filter and no predicate on either source: a
// GenerationChangedPredicate would drop the provider restore's phase transitions and the parent's own
// status, finalizer and deletion events, all of which this level-triggered workflow reconciles on.
func (r *restoreReconciler) SetupWithManager(mgr controllerManager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8sv1.Restore{}).
		Owns(&velerov1.Restore{}).
		Complete(r)
}
