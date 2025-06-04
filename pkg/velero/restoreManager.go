package velero

import (
	"context"
	"fmt"
	v1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/watch"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type defaultRestoreManager struct {
	k8sClient       k8sWatchClient
	discoveryClient discoveryClient
	recorder        eventRecorder
}

// newDefaultRestoreManager creates a new instance of defaultRestoreManager.
func newDefaultRestoreManager(k8sClient k8sWatchClient, discoveryClient discoveryClient, recorder eventRecorder) *defaultRestoreManager {
	return &defaultRestoreManager{discoveryClient: discoveryClient, k8sClient: k8sClient, recorder: recorder}
}

// CreateRestore creates a restore according to the restore configuration in v1.Restore.
func (rm *defaultRestoreManager) CreateRestore(ctx context.Context, restore *v1.Restore) error {
	rm.recorder.Event(restore, corev1.EventTypeNormal, v1.CreateEventReason, "Using velero as restore provider")

	resources, err := rm.getResourcesForGroup("k8s.cloudogu.com")
	if err != nil {
		return rm.handleFailedRestore(restore, fmt.Errorf("failed to get group resources: %w", err))
	}

	veleroRestore := &velerov1.Restore{
		ObjectMeta: metav1.ObjectMeta{
			Name: restore.Name, Namespace: restore.Namespace,
		},
		Spec: velerov1.RestoreSpec{
			BackupName:             restore.Spec.BackupName,
			ExistingResourcePolicy: velerov1.PolicyTypeUpdate,
			RestoreStatus:          &velerov1.RestoreStatusSpec{IncludedResources: resources},
			LabelSelector: &metav1.LabelSelector{
				// Filter backup-operator from restore.
				MatchExpressions: []metav1.LabelSelectorRequirement{
					{
						Key:      "k8s.cloudogu.com/part-of",
						Operator: metav1.LabelSelectorOpNotIn,
						Values:   []string{"backup"},
					},
				},
			},
		},
	}
	err = rm.k8sClient.Create(ctx, veleroRestore)
	if err != nil {
		return rm.handleFailedRestore(restore, fmt.Errorf("failed to create velero restore [%s]: %w", veleroRestore.Name, err))
	}

	selector, err := fields.ParseSelector(restore.GetFieldSelectorWithName())
	if err != nil {
		return rm.handleFailedRestore(restore, fmt.Errorf("failed to parse selector %q: %w", restore, err))
	}

	watcher, err := rm.k8sClient.Watch(ctx, &velerov1.RestoreList{}, &client.ListOptions{FieldSelector: selector})
	if err != nil {
		return rm.handleFailedRestore(restore, fmt.Errorf("failed to create velero restore watch: %w", err))
	}

	resultChan := watcher.ResultChan()
	defer watcher.Stop()

	err = waitForRestoreCompletionOrFailure(ctx, resultChan)
	if err != nil {
		return rm.handleFailedRestore(restore, err)
	}

	rm.recorder.Eventf(restore, corev1.EventTypeNormal, v1.CreateEventReason, "Successfully completed velero restore [%s]", veleroRestore.Name)
	return nil
}

func (rm *defaultRestoreManager) handleFailedRestore(restore *v1.Restore, err error) error {
	rm.recorder.Event(restore, corev1.EventTypeWarning, v1.ErrorOnCreateEventReason, err.Error())
	return err
}

func (rm *defaultRestoreManager) getResourcesForGroup(group string) ([]string, error) {
	apiResourceLists, err := rm.discoveryClient.ServerPreferredResources()
	if err != nil {
		return nil, fmt.Errorf("error getting api resource List")
	}

	var resources []string
	for _, apiResourceList := range apiResourceLists {
		for _, apiResource := range apiResourceList.APIResources {
			if apiResource.Group == group {
				resources = append(resources, apiResource.Name+"."+apiResource.Group)
			}
		}
	}
	return resources, nil
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
		case watch.Modified:
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

	err := rm.k8sClient.Delete(ctx, restore)
	if errors.IsNotFound(err) {
		logger.Info(fmt.Sprintf("velero restore resource [%s] not found: ignore", restore.Name))
		return nil
	}

	if err != nil {
		return fmt.Errorf("failed to delete velero restore [%s]: %w", restore.Name, err)
	}

	return nil
}
