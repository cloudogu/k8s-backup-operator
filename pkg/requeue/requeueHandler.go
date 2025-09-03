package requeue

import (
	"context"
	"errors"
	"fmt"
	k8sv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/retry-lib/retry"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"time"
)

// RequeueEventReason The name of the requeue event
const RequeueEventReason = "Requeue"

// defaultRequeueHandler is responsible to requeue a backup resource after it failed.
type defaultRequeueHandler struct {
	clientSet ecosystemInterface
	namespace string
	recorder  eventRecorder
}

// NewRequeueHandler creates a new component requeue handler.
func NewRequeueHandler(clientSet ecosystemInterface, recorder record.EventRecorder, namespace string) *defaultRequeueHandler {
	return &defaultRequeueHandler{
		clientSet: clientSet,
		namespace: namespace,
		recorder:  recorder,
	}
}

// Handle takes an error and handles the requeue process for the current backup operation.
func (brh *defaultRequeueHandler) Handle(ctx context.Context, contextMessage string, requeuableObject k8sv1.RequeuableObject, originalErr error, requeueStatus string) (ctrl.Result, error) {
	requeueable, requeueableErr := shouldRequeue(originalErr)
	if !requeueable {
		return brh.noLongerHandleRequeueing(ctx, requeuableObject)
	}

	requeueTime := requeueableErr.GetRequeueTime(requeuableObject.GetStatus().GetRequeueTimeNanos())

	updateError := brh.getAndUpdateObject(ctx, requeuableObject, requeueStatus, requeueTime)
	if updateError != nil {
		return ctrl.Result{}, fmt.Errorf("failed to update status from requeue object %s (type: %T) while requeueing: %w", requeuableObject.GetName(), requeuableObject, updateError)
	}

	result := ctrl.Result{Requeue: true, RequeueAfter: requeueTime}
	brh.fireRequeueEvent(requeuableObject, result)

	log.FromContext(ctx).Info(fmt.Sprintf("%s: requeue in %s seconds because of: %s", contextMessage, requeueTime, originalErr.Error()))

	return result, nil
}

func (brh *defaultRequeueHandler) getAndUpdateObject(ctx context.Context, requeuableObject k8sv1.RequeuableObject, requeueStatus string, requeueTime time.Duration) error {
	return retry.OnConflict(func() error {
		switch objectType := requeuableObject.(type) {
		case *k8sv1.Backup:
			backupClient := brh.clientSet.EcosystemV1Alpha1().Backups(brh.namespace)

			updatedBackup, err := backupClient.Get(ctx, requeuableObject.GetName(), metav1.GetOptions{})
			if err != nil {
				return err
			}
			updatedBackup.Status.Status = requeueStatus
			updatedBackup.Status.RequeueTimeNanos = requeueTime
			_, err = backupClient.UpdateStatus(ctx, updatedBackup, metav1.UpdateOptions{})
			return err
		case *k8sv1.Restore:
			restoreClient := brh.clientSet.EcosystemV1Alpha1().Restores(brh.namespace)

			updatedBackup, err := restoreClient.Get(ctx, requeuableObject.GetName(), metav1.GetOptions{})
			if err != nil {
				return err
			}
			updatedBackup.Status.Status = requeueStatus
			updatedBackup.Status.RequeueTimeNanos = requeueTime
			_, err = restoreClient.UpdateStatus(ctx, updatedBackup, metav1.UpdateOptions{})
			return err
		default:
			return fmt.Errorf("wrong requeueable object type %T", objectType)
		}
	})
}

// noLongerHandleRequeueing returns values so the backup will no longer be requeued. This will occur either on a
// successful reconciliation or errors which cannot be handled and thus not be requeued. The component may reset the
// requeue backoff if necessary in order to avoid a wrong backoff baseline time for future reconciliations.
func (brh *defaultRequeueHandler) noLongerHandleRequeueing(ctx context.Context, requeuableObject k8sv1.RequeuableObject) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if requeuableObject.GetStatus().GetRequeueTimeNanos() == 0 {
		logger.Info("Skipping backoff time reset")
		return ctrl.Result{}, nil
	}

	logger.Info("Reset backoff time to 0")
	err := brh.getAndUpdateObject(ctx, requeuableObject, requeuableObject.GetStatus().GetStatus(), 0)

	return ctrl.Result{}, err
}

func shouldRequeue(err error) (bool, requeuableError) {
	var requeueableError requeuableError
	return errors.As(err, &requeueableError), requeueableError
}

func (brh *defaultRequeueHandler) fireRequeueEvent(backup k8sv1.RequeuableObject, result ctrl.Result) {
	brh.recorder.Eventf(backup, corev1.EventTypeNormal, RequeueEventReason, "Falling back to backup status %s: Trying again in %s.", backup.GetStatus().GetStatus(), result.RequeueAfter.String())
}
