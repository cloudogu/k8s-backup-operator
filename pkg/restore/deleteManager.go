package restore

import (
	"context"
	"fmt"
	"github.com/cloudogu/k8s-backup-operator/pkg/provider"

	v1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
)

type defaultDeleteManager struct {
	restoreClient ecosystemRestoreInterface
	recorder      eventRecorder
}

func newDeleteManager(restoreClient ecosystemRestoreInterface, recorder eventRecorder) *defaultDeleteManager {
	return &defaultDeleteManager{restoreClient: restoreClient, recorder: recorder}
}

func (dm *defaultDeleteManager) delete(ctx context.Context, restore *v1.Restore) error {
	_, err := dm.restoreClient.UpdateStatusDeleting(ctx, restore)
	if err != nil {
		return fmt.Errorf("failed to update status [%s] on restore [%s]: %w", v1.RestoreStatusDeleting, restore.Name, err)
	}

	restoreDeleteProvider, err := provider.GetProvider(ctx, restore.Spec.Provider, restore.Namespace, dm.recorder)
	if err != nil {
		return fmt.Errorf("failed to get provider [%s]: %w", restore.Spec.Provider, err)
	}

	err = restoreDeleteProvider.DeleteRestore(ctx, restore)
	if err != nil {
		return fmt.Errorf("failed to delete restore: %w", err)
	}

	_, err = dm.restoreClient.RemoveFinalizer(ctx, restore, v1.RestoreFinalizer)
	if err != nil {
		return fmt.Errorf("failed to delete finalizer [%s]: %w", v1.RestoreFinalizer, err)
	}

	return nil
}
