package velero

import (
	"context"
	"errors"
	"fmt"

	"github.com/cloudogu/k8s-backup-operator/pkg/api/ecosystem"
	v1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"

	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	veleroclient "github.com/vmware-tanzu/velero/pkg/client"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const veleroBackupDeleteRequestKind = "DeleteBackupRequest"

var deleteWaitTimeout = int64(300)

type provider struct {
	recorder        eventRecorder
	veleroClientSet veleroClientSet
}

func New(client ecosystem.BackupInterface, recorder eventRecorder, namespace string) (*provider, error) {
	factory := veleroclient.NewFactory("k8s-backup-operator", map[string]interface{}{"namespace": namespace})
	clientSet, err := factory.Client()
	if err != nil {
		return nil, fmt.Errorf("failed to create velero clientset: %w", err)
	}

	return &provider{recorder: recorder, veleroClientSet: clientSet}, nil
}

// CreateBackup triggers a velero backup.
func (p *provider) CreateBackup(ctx context.Context, backup *v1.Backup) error {
	p.recorder.Event(backup, corev1.EventTypeNormal, v1.CreateEventReason, "Using velero as backup provider")

	namespace := backup.Namespace
	volumeFsBackup := false
	veleroBackup := &velerov1.Backup{
		ObjectMeta: metav1.ObjectMeta{Name: backup.Name, Namespace: namespace},
		Spec: velerov1.BackupSpec{
			ExcludedNamespaces:       []string{namespace},
			StorageLocation:          "default",
			DefaultVolumesToFsBackup: &volumeFsBackup,
		},
	}

	backupsClient := p.veleroClientSet.VeleroV1().Backups(namespace)
	_, err := backupsClient.Create(ctx, veleroBackup, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to apply velero backup '%s/%s' to cluster: %w", namespace, veleroBackup.Name, err)
	}

	watcher, err := backupsClient.Watch(ctx, metav1.ListOptions{FieldSelector: backup.GetFieldSelectorWithName()})
	if err != nil {
		return fmt.Errorf("failed to create watch for velero backup '%s/%s': %w", namespace, veleroBackup.Name, err)
	}

	veleroBackupChan := watcher.ResultChan()
	err = waitForBackupCompletionOrFailure(veleroBackupChan, namespace, veleroBackup, p, backup, watcher)
	if err != nil {
		return err
	}

	return nil
}

func waitForBackupCompletionOrFailure(veleroBackupChan <-chan watch.Event, namespace string, veleroBackup *velerov1.Backup, p *provider, backup *v1.Backup, watcher watch.Interface) error {
	for veleroChange := range veleroBackupChan {
		switch veleroChange.Type {
		case watch.Error:
			message := fmt.Sprintf("failed to complete velero backup '%s/%s': watch returned error", namespace, veleroBackup.Name)
			p.recorder.Event(backup, corev1.EventTypeWarning, v1.ErrorOnCreateEventReason, message)
			watcher.Stop()
			return fmt.Errorf(message)
		case watch.Deleted:
			message := fmt.Sprintf("failed to complete velero backup '%s/%s': the backup is being deleted", namespace, veleroBackup.Name)
			watcher.Stop()
			return fmt.Errorf(message)
		case watch.Modified:
			modifiedBackup := veleroChange.Object.(*velerov1.Backup)

			switch modifiedBackup.Status.Phase {
			case velerov1.BackupPhaseFailedValidation:
				fallthrough
			case velerov1.BackupPhaseWaitingForPluginOperationsPartiallyFailed:
				fallthrough
			case velerov1.BackupPhaseFinalizingPartiallyFailed:
				fallthrough
			case velerov1.BackupPhasePartiallyFailed:
				fallthrough
			case velerov1.BackupPhaseFailed:
				message := fmt.Sprintf("failed to complete velero backup '%s/%s': has status phase '%s'", namespace, modifiedBackup.Name, modifiedBackup.Status.Phase)
				p.recorder.Event(backup, corev1.EventTypeWarning, v1.ErrorOnCreateEventReason, message)
				watcher.Stop()
				return fmt.Errorf(message)
			case velerov1.BackupPhaseDeleting:
				message := fmt.Sprintf("failed to complete velero backup '%s/%s': invalid status phase 'Deleting'", namespace, modifiedBackup.Name)
				p.recorder.Event(backup, corev1.EventTypeWarning, v1.ErrorOnCreateEventReason, message)
				watcher.Stop()
				return fmt.Errorf(message)
			case velerov1.BackupPhaseCompleted:
				p.recorder.Eventf(backup, corev1.EventTypeNormal, v1.CreateEventReason, "Successfully completed velero backup '%s/%s'", namespace, modifiedBackup.Name)
				watcher.Stop()
				break
			}
		}
	}
	return nil
}

// DeleteBackup deletes a velero backup with a delete backup request.
// Potential errors from the request will be returned. Nil if no occur.
// TODO maybe the delete process should become an own CRD.
func (p *provider) DeleteBackup(ctx context.Context, backup *v1.Backup) error {
	p.recorder.Event(backup, corev1.EventTypeNormal, v1.ProviderDeleteEventReason, "Trigger velero provider to delete backup.")
	requestCR := getDeleteBackupRequestCR(backup.Name, backup.Namespace)
	_, err := p.veleroClientSet.VeleroV1().DeleteBackupRequests(backup.Namespace).Create(ctx, requestCR, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create backup delete request %s: %w", requestCR.Name, err)
	}

	watcher, err := p.veleroClientSet.VeleroV1().DeleteBackupRequests(backup.Namespace).Watch(
		ctx, metav1.ListOptions{FieldSelector: backup.GetFieldSelectorWithName(), TimeoutSeconds: &deleteWaitTimeout})
	if err != nil {
		return fmt.Errorf("failed to create watch for delete backup request %s: %w", backup.Name, err)
	}

	for event := range watcher.ResultChan() {
		if event.Type == watch.Modified {
			cr, ok := event.Object.(*velerov1.DeleteBackupRequest)
			if ok && cr.Status.Phase == velerov1.DeleteBackupRequestPhaseProcessed {
				break
			}
		}
	}

	requestCR, err = p.veleroClientSet.VeleroV1().DeleteBackupRequests(backup.Namespace).Get(ctx, backup.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get delete backup request %s: %w", backup.Name, err)
	}

	if requestCR.Status.Phase != velerov1.DeleteBackupRequestPhaseProcessed {
		p.cleanUpDeleteRequest(ctx, backup, requestCR)
		return fmt.Errorf("failed to delete backup %s: timout waiting for backup delete request %s", backup.Name, requestCR.Name)
	}

	backUpErrors := requestCR.Status.Errors
	if len(backUpErrors) == 0 {
		p.recorder.Event(backup, corev1.EventTypeNormal, v1.ProviderDeleteEventReason, "Provider delete request successful.")
		return nil
	}

	var multiErr error = nil
	for _, errStr := range backUpErrors {
		multiErr = errors.Join(multiErr, fmt.Errorf("velero backup delete request error: %s", errStr))
	}
	p.recorder.Event(backup, corev1.EventTypeWarning, v1.ErrorOnProviderDeleteEventReason, multiErr.Error())
	p.cleanUpDeleteRequest(ctx, backup, requestCR)

	return &genericRequeueableError{
		errMsg: "failed to delete backup",
		err:    multiErr,
	}
}

func (p *provider) cleanUpDeleteRequest(ctx context.Context, backup *v1.Backup, request *velerov1.DeleteBackupRequest) {
	p.recorder.Event(backup, corev1.EventTypeWarning, v1.ProviderDeleteEventReason, "Cleanup velero delete request.")
	err := p.veleroClientSet.VeleroV1().DeleteBackupRequests(request.Namespace).Delete(ctx, request.Name, metav1.DeleteOptions{})
	if err != nil {
		p.recorder.Event(backup, corev1.EventTypeWarning, v1.ProviderDeleteEventReason, "Error cleanup velero delete request.")
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
