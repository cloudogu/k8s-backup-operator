package restore

import (
	"context"
	"errors"
	"fmt"
	k8sv1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
	"github.com/cloudogu/k8s-backup-operator/pkg/retry"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// RequeueEventReason The name of the requeue event
const RequeueEventReason = "Requeue"

// defaultRequeueHandler is responsible to requeue a restore resource after it failed.
type defaultRequeueHandler struct {
	clientSet ecosystemInterface
	namespace string
	recorder  record.EventRecorder
}

// NewRequeueHandler creates a new restore requeue handler.
func NewRequeueHandler(clientSet ecosystemInterface, recorder record.EventRecorder, namespace string) *defaultRequeueHandler {
	return &defaultRequeueHandler{
		clientSet: clientSet,
		namespace: namespace,
		recorder:  recorder,
	}
}

// Handle takes an error and handles the requeue process for the current restore operation.
func (brh *defaultRequeueHandler) Handle(ctx context.Context, contextMessage string, restore *k8sv1.Restore, originalErr error, requeueStatus string) (ctrl.Result, error) {
	requeueable, requeueableErr := shouldRequeue(originalErr)
	if !requeueable {
		return brh.noLongerHandleRequeueing(ctx, restore)
	}

	requeueTime := requeueableErr.GetRequeueTime(restore.Status.RequeueTimeNanos)

	updateError := retry.OnConflict(func() error {
		restoreClient := brh.clientSet.EcosystemV1Alpha1().Restores(brh.namespace)

		updatedRestore, err := restoreClient.Get(ctx, restore.GetName(), metav1.GetOptions{})
		if err != nil {
			return err
		}

		updatedRestore.Status.Status = requeueStatus
		updatedRestore.Status.RequeueTimeNanos = requeueTime
		restore, err = restoreClient.UpdateStatus(ctx, updatedRestore, metav1.UpdateOptions{})
		return err
	})
	if updateError != nil {
		return ctrl.Result{}, fmt.Errorf("failed to update restore status while requeueing: %w", updateError)
	}

	result := ctrl.Result{Requeue: true, RequeueAfter: requeueTime}
	brh.fireRequeueEvent(restore, result)

	log.FromContext(ctx).Info(fmt.Sprintf("%s: requeue in %s seconds because of: %s", contextMessage, requeueTime, originalErr.Error()))

	return result, nil
}

// noLongerHandleRequeueing returns values so the restore will no longer be requeued. This will occur either on a
// successful reconciliation or errors which cannot be handled and thus not be requeued. The component may reset the
// requeue backoff if necessary in order to avoid a wrong backoff baseline time for future reconciliations.
func (brh *defaultRequeueHandler) noLongerHandleRequeueing(ctx context.Context, restore *k8sv1.Restore) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if restore.Status.RequeueTimeNanos == 0 {
		logger.Info("Skipping backoff time reset")
		return ctrl.Result{}, nil
	}

	restoreClient := brh.clientSet.EcosystemV1Alpha1().Restores(brh.namespace)

	err := retry.OnConflict(func() error {
		updatedRestore, err := restoreClient.Get(ctx, restore.GetName(), metav1.GetOptions{})
		if err != nil {
			return err
		}

		logger.Info("Reset backoff time to 0")
		updatedRestore.Status.RequeueTimeNanos = 0
		_, err = restoreClient.UpdateStatus(ctx, updatedRestore, metav1.UpdateOptions{})
		return err
	})

	return ctrl.Result{}, err
}

func shouldRequeue(err error) (bool, requeuableError) {
	var requeueableError requeuableError
	return errors.As(err, &requeueableError), requeueableError
}

func (brh *defaultRequeueHandler) fireRequeueEvent(restore *k8sv1.Restore, result ctrl.Result) {
	brh.recorder.Eventf(restore, corev1.EventTypeNormal, RequeueEventReason, "Falling back to restore status %s: Trying again in %s.", restore.Status.Status, result.RequeueAfter.String())
}
