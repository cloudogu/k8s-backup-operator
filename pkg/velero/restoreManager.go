package velero

import (
	"context"
	"fmt"
	v1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

type defaultRestoreManager struct {
	veleroClientSet veleroClientSet
	recorder        eventRecorder
}

// NewDefaultRestoreManager creates a new instance of defaultRestoreManager.
func NewDefaultRestoreManager(veleroClientSet veleroClientSet, recorder eventRecorder) *defaultRestoreManager {
	return &defaultRestoreManager{veleroClientSet: veleroClientSet, recorder: recorder}
}

// CreateRestore creates a restore according to the restore configuration in v1.Restore.
func (rm *defaultRestoreManager) CreateRestore(ctx context.Context, restore *v1.Restore) error {
	rm.recorder.Event(restore, corev1.EventTypeNormal, v1.CreateEventReason, "Using velero as restore provider")

	veleroRestore := &velerov1.Restore{
		ObjectMeta: metav1.ObjectMeta{
			Name: restore.Name, Namespace: restore.Namespace,
		},
		Spec: velerov1.RestoreSpec{
			BackupName:             restore.Spec.BackupName,
			ExistingResourcePolicy: velerov1.PolicyTypeUpdate,
			RestoreStatus:          &velerov1.RestoreStatusSpec{IncludedResources: []string{"*"}},
			LabelSelector: &metav1.LabelSelector{
				// Filter backup-operator from restore.
				MatchExpressions: []metav1.LabelSelectorRequirement{
					{
						Key:      "app.kubernetes.io/name",
						Operator: metav1.LabelSelectorOpNotIn,
						Values:   []string{"k8s-backup-operator"},
					},
					{
						Key:      "app.kubernetes.io/part-of",
						Operator: metav1.LabelSelectorOpNotIn,
						Values:   []string{"k8s-backup-operator"},
					}},
			},
		},
	}

	_, err := rm.veleroClientSet.VeleroV1().Restores(restore.Namespace).Create(ctx, veleroRestore, metav1.CreateOptions{})
	if err != nil {
		return rm.handleFailedRestore(restore, fmt.Errorf("failed to create velero restore [%s]: %w", veleroRestore.Name, err))
	}

	watcher, err := rm.veleroClientSet.VeleroV1().Restores(veleroRestore.Namespace).Watch(ctx, metav1.ListOptions{FieldSelector: restore.GetFieldSelectorWithName()})
	if err != nil {
		return rm.handleFailedRestore(restore, fmt.Errorf("failed to create velero restore watch: %w", err))
	}

	resultChan := watcher.ResultChan()
	defer watcher.Stop()

	err = waitForRestoreCompletionOrFailure(resultChan)
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

func waitForRestoreCompletionOrFailure(veleroRestoreChan <-chan watch.Event) error {
	for veleroChange := range veleroRestoreChan {
		changedRestore, ok := veleroChange.Object.(*velerov1.Restore)
		if !ok {
			return fmt.Errorf("got event with wrong object type when watching velero restore")
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

func (rm *defaultRestoreManager) DeleteRestore(ctx context.Context, restore *v1.Restore) error {
	rm.recorder.Event(restore, corev1.EventTypeNormal, v1.DeleteEventReason, "Using velero as restore provider")

	err := rm.veleroClientSet.VeleroV1().Restores(restore.Namespace).Delete(ctx, restore.Name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete velero restore [%s]: %w", restore.Name, err)
	}

	return nil
}
