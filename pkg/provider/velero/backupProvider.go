package velero

import (
	"context"
	"fmt"
	"github.com/cloudogu/k8s-backup-operator/pkg/api/ecosystem"
	v1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	veleroclient "github.com/vmware-tanzu/velero/pkg/client"
	"github.com/vmware-tanzu/velero/pkg/generated/clientset/versioned"
	"go.uber.org/multierr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"time"
)

const veleroBackupDeleteRequestKind = "DeleteBackupRequest"

var deleteWaitTimeout = int64(300)

type provider struct {
	recorder        eventRecorder
	veleroClientSet versioned.Interface
}

func New(client ecosystem.BackupInterface, recorder eventRecorder, namespace string) (*provider, error) {
	factory := veleroclient.NewFactory("k8s-backup-operator", map[string]interface{}{"namespace": namespace})
	clientSet, err := factory.Client()
	if err != nil {
		return nil, fmt.Errorf("failed to create velero clientset: %w", err)
	}

	return &provider{recorder: recorder, veleroClientSet: clientSet}, nil
}

func (p *provider) CreateBackup(ctx context.Context, backup *v1.Backup) error {
	p.recorder.Event(backup, corev1.EventTypeNormal, v1.CreateEventReason, "Use velero as backup provider")
	println("Hier könnte ihre Backup-Lösung verwendet werden")
	time.Sleep(time.Second * 5)
	println("Ende")
	return nil
}

// DeleteBackup deletes a velero backup with a delete backup request.
// Potential errors from the request will be returned. Nil if no occur.
// TODO This routine needs an own CRD. In error cases the backup cr will stuck and can't be retriggered.
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
		p.cleanUpDeleteRequest(ctx, backup, requestCR)
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
		p.cleanUpDeleteRequest(ctx, backup, requestCR)
		return fmt.Errorf("failed to get delete backup request %s: %w", backup.Name, err)
	}

	if requestCR.Status.Phase != velerov1.DeleteBackupRequestPhaseProcessed {
		p.cleanUpDeleteRequest(ctx, backup, requestCR)
		return fmt.Errorf("failed to delete backup %s: timout waiting for backup delete request %s", backup.Name, requestCR.Name)
	}

	errors := requestCR.Status.Errors
	if len(errors) == 0 {
		p.recorder.Event(backup, corev1.EventTypeNormal, v1.ProviderDeleteEventReason, "Provider delete request successful.")
		return nil
	}

	var multiErr error = nil
	for _, errStr := range errors {
		multiErr = multierr.Append(multiErr, fmt.Errorf("backup delete request error: %s", errStr))
	}
	p.recorder.Event(backup, corev1.EventTypeWarning, v1.ErrorOnProviderDeleteEventReason, multiErr.Error())
	p.cleanUpDeleteRequest(ctx, backup, requestCR)

	return multiErr
}

func (p *provider) cleanUpDeleteRequest(ctx context.Context, backup *v1.Backup, request *velerov1.DeleteBackupRequest) {
	p.recorder.Event(backup, corev1.EventTypeWarning, v1.ProviderDeleteEventReason, "Cleanup velero delete request.")
	err := p.veleroClientSet.VeleroV1().Backups(request.Namespace).Delete(ctx, request.Name, metav1.DeleteOptions{})
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
