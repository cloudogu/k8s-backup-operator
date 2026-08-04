package velero

import (
	"context"
	"fmt"

	v1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
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
