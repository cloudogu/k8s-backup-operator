package backup

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

// backupRequeueHandler is responsible to requeue a backup resource after it failed.
type backupRequeueHandler struct {
	clientSet ecosystemInterface
	namespace string
	recorder  record.EventRecorder
}

// NewBackupRequeueHandler creates a new component requeue handler.
func NewBackupRequeueHandler(clientSet ecosystemInterface, recorder record.EventRecorder, namespace string) *backupRequeueHandler {
	return &backupRequeueHandler{
		clientSet: clientSet,
		namespace: namespace,
		recorder:  recorder,
	}
}

// Handle takes an error and handles the requeue process for the current backup operation.
func (brh *backupRequeueHandler) Handle(ctx context.Context, contextMessage string, backup *k8sv1.Backup, originalErr error, requeueStatus string) (ctrl.Result, error) {
	requeueable, requeueableErr := shouldRequeue(originalErr)
	if !requeueable {
		return brh.noLongerHandleRequeueing(ctx, backup)
	}

	requeueTime := requeueableErr.GetRequeueTime(backup.Status.RequeueTimeNanos)

	updateError := retry.OnConflict(func() error {
		backupClient := brh.clientSet.EcosystemV1Alpha1().Backups(brh.namespace)

		updatedBackup, err := backupClient.Get(ctx, backup.GetName(), metav1.GetOptions{})
		if err != nil {
			return err
		}

		updatedBackup.Status.Status = requeueStatus
		updatedBackup.Status.RequeueTimeNanos = requeueTime
		backup, err = backupClient.UpdateStatus(ctx, updatedBackup, metav1.UpdateOptions{})
		return err
	})
	if updateError != nil {
		return ctrl.Result{}, fmt.Errorf("failed to update backup status while requeueing: %w", updateError)
	}

	result := ctrl.Result{Requeue: true, RequeueAfter: requeueTime}
	brh.fireRequeueEvent(backup, result)

	log.FromContext(ctx).Info(fmt.Sprintf("%s: requeue in %s seconds because of: %s", contextMessage, requeueTime, originalErr.Error()))

	return result, nil
}

// noLongerHandleRequeueing returns values so the backup will no longer be requeued. This will occur either on a
// successful reconciliation or errors which cannot be handled and thus not be requeued. The component may reset the
// requeue backoff if necessary in order to avoid a wrong backoff baseline time for future reconciliations.
func (brh *backupRequeueHandler) noLongerHandleRequeueing(ctx context.Context, backup *k8sv1.Backup) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if backup.Status.RequeueTimeNanos == 0 {
		logger.Info("Skipping backoff time reset")
		return ctrl.Result{}, nil
	}

	backupClient := brh.clientSet.EcosystemV1Alpha1().Backups(brh.namespace)

	err := retry.OnConflict(func() error {
		updatedBackup, err := backupClient.Get(ctx, backup.GetName(), metav1.GetOptions{})
		if err != nil {
			return err
		}

		logger.Info("Reset backoff time to 0")
		updatedBackup.Status.RequeueTimeNanos = 0
		_, err = backupClient.UpdateStatus(ctx, updatedBackup, metav1.UpdateOptions{})
		return err
	})

	return ctrl.Result{}, err
}

func shouldRequeue(err error) (bool, requeuableError) {
	var requeueableError requeuableError
	return errors.As(err, &requeueableError), requeueableError
}

func (brh *backupRequeueHandler) fireRequeueEvent(backup *k8sv1.Backup, result ctrl.Result) {
	brh.recorder.Eventf(backup, corev1.EventTypeNormal, RequeueEventReason, "Falling back to backup status %s: Trying again in %s.", backup.Status.Status, result.RequeueAfter.String())
}
