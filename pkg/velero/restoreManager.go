package velero

import (
	"context"
	"fmt"

	v1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/watch"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type defaultRestoreManager struct {
	k8sClient k8sWatchClient
	recorder  eventRecorder
}

// newDefaultRestoreManager creates a new instance of defaultRestoreManager.
func newDefaultRestoreManager(k8sClient k8sWatchClient, recorder eventRecorder) *defaultRestoreManager {
	return &defaultRestoreManager{k8sClient: k8sClient, recorder: recorder}
}

// WaitForRestore blocks until the velero restore of the given v1.Restore completed or failed.
// The velero restore itself is created by the restore controller, which owns the child resource.
func (rm *defaultRestoreManager) WaitForRestore(ctx context.Context, restore *v1.Restore) error {
	rm.recorder.Event(restore, corev1.EventTypeNormal, v1.CreateEventReason, "Using velero as restore provider")

	selector, err := fields.ParseSelector(restore.GetFieldSelectorWithName())
	if err != nil {
		return rm.handleFailedRestore(restore, fmt.Errorf("failed to parse selector %q: %w", restore.GetFieldSelectorWithName(), err))
	}

	watcher, err := rm.k8sClient.Watch(ctx, &velerov1.RestoreList{}, &client.ListOptions{FieldSelector: selector, Namespace: restore.Namespace})
	if err != nil {
		return rm.handleFailedRestore(restore, fmt.Errorf("failed to create velero restore watch: %w", err))
	}

	resultChan := watcher.ResultChan()
	defer watcher.Stop()

	err = waitForRestoreCompletionOrFailure(ctx, resultChan)
	if err != nil {
		return rm.handleFailedRestore(restore, err)
	}

	rm.recorder.Eventf(restore, corev1.EventTypeNormal, v1.CreateEventReason, "Successfully completed velero restore [%s]", restore.Name)
	return nil
}

func (rm *defaultRestoreManager) handleFailedRestore(restore *v1.Restore, err error) error {
	rm.recorder.Event(restore, corev1.EventTypeWarning, v1.ErrorOnCreateEventReason, err.Error())
	return err
}

func waitForRestoreCompletionOrFailure(ctx context.Context, veleroRestoreChan <-chan watch.Event) error {
	logger := log.FromContext(ctx)
	for veleroChange := range veleroRestoreChan {
		changedRestore, ok := veleroChange.Object.(*velerov1.Restore)
		if !ok {
			logger.Error(fmt.Errorf("got event with wrong object type when watching velero restore type: %T object: %#v", veleroChange.Object, veleroChange.Object), "wrong event type")
			continue
		}

		switch veleroChange.Type {
		case watch.Deleted:
			return fmt.Errorf("failed to complete velero restore [%s]: the restore got deleted", changedRestore.Name)
		case watch.Added, watch.Modified:
			switch changedRestore.Status.Phase {
			case velerov1.RestorePhaseFailedValidation:
				fallthrough
			case velerov1.RestorePhaseWaitingForPluginOperationsPartiallyFailed:
				fallthrough
			case velerov1.RestorePhasePartiallyFailed:
				fallthrough
			case velerov1.RestorePhaseFailed:
				return fmt.Errorf("failed to complete velero restore [%s]: has status phase [%s]", changedRestore.Name, changedRestore.Status.Phase)
			case velerov1.RestorePhaseCompleted:
				return nil
			}
		}
	}

	return nil
}

// DeleteRestore deletes a restore.
func (rm *defaultRestoreManager) DeleteRestore(ctx context.Context, restore *v1.Restore) error {
	logger := log.FromContext(ctx)
	rm.recorder.Event(restore, corev1.EventTypeNormal, v1.DeleteEventReason, "Using velero as restore provider")

	veleroRestore := &velerov1.Restore{}
	// Velero resource uses same namespaced name as cloudogu one
	err := rm.k8sClient.Get(ctx, client.ObjectKeyFromObject(restore), veleroRestore)
	if errors.IsNotFound(err) {
		logger.Info(fmt.Sprintf("velero restore resource [%s] not found: ignore", restore.Name))
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to get velero restore [%s]: %w", restore.Name, err)
	}

	err = rm.k8sClient.Delete(ctx, veleroRestore)
	if errors.IsNotFound(err) {
		logger.Info(fmt.Sprintf("velero restore resource [%s] already deleted: ignore", restore.Name))
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to delete velero restore [%s]: %w", restore.Name, err)
	}

	return nil
}
