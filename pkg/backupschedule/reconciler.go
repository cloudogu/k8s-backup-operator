package backupschedule

import (
	"context"
	"fmt"
	"github.com/cloudogu/k8s-backup-operator/pkg/additionalimages"
	"strings"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	k8sv1 "github.com/cloudogu/k8s-backup-lib/pkg/api/v1"
	"github.com/cloudogu/k8s-backup-operator/pkg/requeue"
	"github.com/cloudogu/retry-lib/retry"
)

type operation string

const (
	operationCreate = operation("create")
	operationUpdate = operation("update")
	operationDelete = operation("delete")
	operationIgnore = operation("ignore")
)

// backupScheduleReconciler reconciles a BackupSchedule object
type backupScheduleReconciler struct {
	clientSet      ecosystemInterface
	recorder       eventRecorder
	namespace      string
	manager        Manager
	requeueHandler requeueHandler
}

func NewReconciler(clientSet ecosystemInterface, recorder eventRecorder, namespace string, requeueHandler requeueHandler, imageConfig additionalimages.ImageConfig) *backupScheduleReconciler {
	return &backupScheduleReconciler{
		clientSet:      clientSet,
		recorder:       recorder,
		requeueHandler: requeueHandler,
		manager:        NewManager(clientSet, recorder, namespace, imageConfig),
		namespace:      namespace,
	}
}

//+kubebuilder:rbac:groups=k8s.cloudogu.com,resources=backupschedules,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=k8s.cloudogu.com,resources=backupschedules/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=k8s.cloudogu.com,resources=backupschedules/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.15.0/pkg/reconcile
func (r *backupScheduleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	backupSchedule, err := r.clientSet.EcosystemV1Alpha1().BackupSchedules(r.namespace).Get(ctx, req.Name, metav1.GetOptions{})
	if err != nil {
		logger.Info(fmt.Sprintf("failed to get backup schedule resource %s/%s: %s", r.namespace, req.Name, err))
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logger.Info(fmt.Sprintf("found backup schedule resource %s", req.NamespacedName))

	requiredOperation, err := r.evaluateRequiredOperation(ctx, backupSchedule)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to evaluate required operation for backup schedule %s: %w", backupSchedule.Name, err)
	}

	logger.Info(fmt.Sprintf("required operation for backup schedule %s is %s", req.NamespacedName, requiredOperation))

	switch requiredOperation {
	case operationCreate:
		return r.performCreateOperation(ctx, backupSchedule)
	case operationUpdate:
		return r.performUpdateOperation(ctx, backupSchedule)
	case operationDelete:
		return r.performDeleteOperation(ctx, backupSchedule)
	case operationIgnore:
		return ctrl.Result{}, nil
	default:
		return ctrl.Result{}, fmt.Errorf("unknown operation: %s", requiredOperation)
	}
}

func (r *backupScheduleReconciler) evaluateRequiredOperation(ctx context.Context, backupSchedule *k8sv1.BackupSchedule) (operation, error) {
	logger := log.FromContext(ctx)

	if backupSchedule.DeletionTimestamp != nil && !backupSchedule.DeletionTimestamp.IsZero() {
		return operationDelete, nil
	}

	switch backupSchedule.Status.Status {
	case k8sv1.BackupScheduleStatusFailed:
		return operationIgnore, nil
	case k8sv1.BackupScheduleStatusNew:
		return operationCreate, nil
	case k8sv1.BackupScheduleStatusCreated:
		var cronJob *batchv1.CronJob
		op := operationIgnore
		err := retry.OnError(maxTries, retry.AlwaysRetryFunc, func() error {
			var err error
			cronJob, err = r.clientSet.BatchV1().CronJobs(r.namespace).Get(ctx, backupSchedule.CronJobName(), metav1.GetOptions{})
			if errors.IsNotFound(err) {
				logger.Error(err, "backup schedule has status 'created' but its cron job does not exist. creating cron job...")
				op = operationCreate
				return nil
			} else if err != nil {
				return err
			}

			if cronJob.Spec.Schedule != backupSchedule.Spec.Schedule || getCronJobProvider(cronJob) != string(backupSchedule.Spec.Provider) {
				op = operationUpdate
			}

			return nil
		})
		if err != nil {
			return "", fmt.Errorf("failed to find cron job for backup schedule %s: %w", backupSchedule.Name, err)
		}

		return op, nil
	default:
		return operationIgnore, nil
	}
}

func getCronJobProvider(cronJob *batchv1.CronJob) string {
	// TODO: Index out of range abfangen?
	argList := cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Args
	for _, arg := range argList {
		if strings.HasPrefix(arg, k8sv1.ProviderArgFlag) {
			provider := strings.TrimPrefix(arg, k8sv1.ProviderArgFlag+"=")
			return provider
		}
	}
	return ""
}

func (r *backupScheduleReconciler) performCreateOperation(ctx context.Context, backupSchedule *k8sv1.BackupSchedule) (ctrl.Result, error) {
	return r.performOperation(ctx, backupSchedule, k8sv1.CreateEventReason, k8sv1.BackupScheduleStatusNew, r.manager.create)
}

func (r *backupScheduleReconciler) performUpdateOperation(ctx context.Context, backupSchedule *k8sv1.BackupSchedule) (ctrl.Result, error) {
	return r.performOperation(ctx, backupSchedule, k8sv1.UpdateEventReason, k8sv1.BackupScheduleStatusCreated, r.manager.update)
}

func (r *backupScheduleReconciler) performDeleteOperation(ctx context.Context, backupSchedule *k8sv1.BackupSchedule) (ctrl.Result, error) {
	return r.performOperation(ctx, backupSchedule, k8sv1.DeleteEventReason, backupSchedule.Status.Status, r.manager.delete)
}

// performOperation executes the given operationFn and requeues if necessary.
// When requeueing, the requeueStatus is set as the restore status.
func (r *backupScheduleReconciler) performOperation(
	ctx context.Context,
	backupSchedule *k8sv1.BackupSchedule,
	eventReason string,
	requeueStatus string,
	operationFn func(context.Context, *k8sv1.BackupSchedule) error,
) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	operationError := operationFn(ctx, backupSchedule)
	contextMessageOnError := fmt.Sprintf("%s of backup schedule %s failed", eventReason, backupSchedule.Name)
	eventType := corev1.EventTypeNormal
	message := fmt.Sprintf("%s successful", eventReason)
	if operationError != nil {
		eventType = corev1.EventTypeWarning
		printError := strings.ReplaceAll(operationError.Error(), "\n", "")
		message = fmt.Sprintf("%s failed. Reason: %s", eventReason, printError)
		logger.Error(operationError, message)
	}

	r.recorder.Event(backupSchedule, eventType, eventReason, message)

	result, handleErr := r.requeueHandler.Handle(ctx, contextMessageOnError, backupSchedule, operationError, requeueStatus)
	if handleErr != nil {
		r.recorder.Eventf(backupSchedule, corev1.EventTypeWarning, requeue.RequeueEventReason,
			"Failed to requeue the %s.", strings.ToLower(eventReason))
		return ctrl.Result{}, fmt.Errorf("failed to handle requeue: %w", handleErr)
	}

	return result, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *backupScheduleReconciler) SetupWithManager(mgr controllerManager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8sv1.BackupSchedule{}).
		Complete(r)
}
