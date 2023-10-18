package velero

import (
	"context"
	"errors"
	"fmt"
	v1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
	"github.com/cloudogu/k8s-backup-operator/pkg/requeue"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const veleroBackupDeleteRequestKind = "DeleteBackupRequest"

const defaultStorageLocation = "default"

var deleteWaitTimeout = int64(300)

type defaultBackupManager struct {
	veleroClientSet veleroClientSet
	recorder        eventRecorder
}

// NewDefaultBackupManager creates a new instance of defaultBackupManager.
func NewDefaultBackupManager(veleroClientSet veleroClientSet, recorder eventRecorder) *defaultBackupManager {
	return &defaultBackupManager{veleroClientSet: veleroClientSet, recorder: recorder}
}

// CreateBackup triggers a velero backup and waits for its completion.
func (bm *defaultBackupManager) CreateBackup(ctx context.Context, backup *v1.Backup) error {
	bm.recorder.Event(backup, corev1.EventTypeNormal, v1.CreateEventReason, "Using velero as backup provider")

	volumeFsBackup := false
	veleroBackup := &velerov1.Backup{
		ObjectMeta: metav1.ObjectMeta{Name: backup.Name, Namespace: backup.Namespace},
		Spec: velerov1.BackupSpec{
			IncludedNamespaces:       []string{backup.Namespace},
			StorageLocation:          defaultStorageLocation,
			DefaultVolumesToFsBackup: &volumeFsBackup,
		},
	}

	veleroBackupClient := bm.veleroClientSet.VeleroV1().Backups(backup.Namespace)
	_, err := veleroBackupClient.Create(ctx, veleroBackup, metav1.CreateOptions{})
	if err != nil {
		return bm.handleFailedBackup(backup, fmt.Errorf("failed to apply velero backup '%s/%s' to cluster: %w", veleroBackup.Namespace, veleroBackup.Name, err))
	}

	watcher, err := veleroBackupClient.Watch(ctx, metav1.ListOptions{FieldSelector: backup.GetFieldSelectorWithName()})
	if err != nil {
		return bm.handleFailedBackup(backup, fmt.Errorf("failed to create watch for velero backup '%s/%s': %w", veleroBackup.Namespace, veleroBackup.Name, err))
	}

	veleroBackupChan := watcher.ResultChan()
	defer watcher.Stop()

	err = waitForBackupCompletionOrFailure(veleroBackupChan)
	if err != nil {
		return bm.handleFailedBackup(backup, err)
	}

	bm.recorder.Eventf(backup, corev1.EventTypeNormal, v1.CreateEventReason, "Successfully completed velero backup '%s/%s'", veleroBackup.Namespace, veleroBackup.Name)
	return nil
}

func (bm *defaultBackupManager) handleFailedBackup(backup *v1.Backup, err error) error {
	bm.recorder.Event(backup, corev1.EventTypeWarning, v1.ErrorOnCreateEventReason, err.Error())
	return err
}

func waitForBackupCompletionOrFailure(veleroBackupChan <-chan watch.Event) error {
	for veleroChange := range veleroBackupChan {
		changedBackup, ok := veleroChange.Object.(*velerov1.Backup)
		if !ok {
			return fmt.Errorf("got event with wrong object type when watching velero backup")
		}

		switch veleroChange.Type {
		case watch.Deleted:
			return fmt.Errorf("failed to complete velero backup '%s/%s': the backup got deleted", changedBackup.Namespace, changedBackup.Name)
		case watch.Modified:
			switch changedBackup.Status.Phase {
			case velerov1.BackupPhaseFailedValidation:
				fallthrough
			case velerov1.BackupPhaseWaitingForPluginOperationsPartiallyFailed:
				fallthrough
			case velerov1.BackupPhaseFinalizingPartiallyFailed:
				fallthrough
			case velerov1.BackupPhasePartiallyFailed:
				fallthrough
			case velerov1.BackupPhaseFailed:
				return fmt.Errorf("failed to complete velero backup '%s/%s': has status phase '%s'", changedBackup.Namespace, changedBackup.Name, changedBackup.Status.Phase)
			case velerov1.BackupPhaseDeleting:
				return fmt.Errorf("failed to complete velero backup '%s/%s': invalid status phase 'Deleting'", changedBackup.Namespace, changedBackup.Name)
			case velerov1.BackupPhaseCompleted:
				return nil
			}
		}
	}

	return nil
}

// DeleteBackup deletes a velero backup with a delete backup request.
// Potential errors from the request will be returned. Nil if no occur.
func (bm *defaultBackupManager) DeleteBackup(ctx context.Context, backup *v1.Backup) error {
	bm.recorder.Event(backup, corev1.EventTypeNormal, v1.ProviderDeleteEventReason, "Trigger velero provider to delete backup.")
	requestCR := getDeleteBackupRequestCR(backup.Name, backup.Namespace)
	_, err := bm.veleroClientSet.VeleroV1().DeleteBackupRequests(backup.Namespace).Create(ctx, requestCR, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create backup delete request %s: %w", requestCR.Name, err)
	}

	watcher, err := bm.veleroClientSet.VeleroV1().DeleteBackupRequests(backup.Namespace).Watch(
		ctx, metav1.ListOptions{FieldSelector: backup.GetFieldSelectorWithName(), TimeoutSeconds: &deleteWaitTimeout})
	if err != nil {
		return fmt.Errorf("failed to create watch for delete backup request %s: %w", backup.Name, err)
	}

	for event := range watcher.ResultChan() {
		if event.Type == watch.Modified {
			cr, ok := event.Object.(*velerov1.DeleteBackupRequest)
			if ok && cr.Status.Phase == velerov1.DeleteBackupRequestPhaseProcessed {
				watcher.Stop()
				break
			}
		}
	}

	requestCR, err = bm.veleroClientSet.VeleroV1().DeleteBackupRequests(backup.Namespace).Get(ctx, backup.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get delete backup request %s: %w", backup.Name, err)
	}

	if requestCR.Status.Phase != velerov1.DeleteBackupRequestPhaseProcessed {
		bm.cleanUpDeleteRequest(ctx, backup, requestCR)
		return fmt.Errorf("failed to delete backup %s: timout waiting for backup delete request %s", backup.Name, requestCR.Name)
	}

	backUpErrors := requestCR.Status.Errors
	if len(backUpErrors) == 0 {
		bm.recorder.Event(backup, corev1.EventTypeNormal, v1.ProviderDeleteEventReason, "Provider delete request successful.")
		return nil
	}

	var multiErr error = nil
	for _, errStr := range backUpErrors {
		multiErr = errors.Join(multiErr, fmt.Errorf("velero backup delete request error: %s", errStr))
	}
	bm.recorder.Event(backup, corev1.EventTypeWarning, v1.ErrorOnProviderDeleteEventReason, multiErr.Error())
	bm.cleanUpDeleteRequest(ctx, backup, requestCR)

	return &requeue.GenericRequeueableError{
		ErrMsg: "failed to delete backup",
		Err:    multiErr,
	}
}

func (bm *defaultBackupManager) cleanUpDeleteRequest(ctx context.Context, backup *v1.Backup, request *velerov1.DeleteBackupRequest) {
	bm.recorder.Event(backup, corev1.EventTypeWarning, v1.ProviderDeleteEventReason, "Cleanup velero delete request.")
	err := bm.veleroClientSet.VeleroV1().DeleteBackupRequests(request.Namespace).Delete(ctx, request.Name, metav1.DeleteOptions{})
	if err != nil {
		bm.recorder.Event(backup, corev1.EventTypeWarning, v1.ProviderDeleteEventReason, "Error cleanup velero delete request.")
		log.FromContext(ctx).Error(err, "velero backup delete error")
	}
}

func getDeleteBackupRequestCR(name, namespace string) *velerov1.DeleteBackupRequest {
	return &velerov1.DeleteBackupRequest{
		TypeMeta: metav1.TypeMeta{
			APIVersion: velerov1.SchemeGroupVersion.String(),
			Kind:       veleroBackupDeleteRequestKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Spec: velerov1.DeleteBackupRequestSpec{
			BackupName: name,
		},
	}
}
