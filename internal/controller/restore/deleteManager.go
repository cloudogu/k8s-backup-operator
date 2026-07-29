package restore

import (
	"context"
	"fmt"

	"github.com/cloudogu/k8s-backup-operator/pkg/provider"

	v1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/k8s-backup-operator/internal/provider/velero"
)

type defaultDeleteManager struct {
	k8sClient k8sClient
	clientSet ecosystemInterface
	namespace string
	recorder  eventRecorder
}

func newDeleteManager(k8sClient k8sClient, clientSet ecosystemInterface, namespace string, recorder eventRecorder) *defaultDeleteManager {
	return &defaultDeleteManager{k8sClient: k8sClient, clientSet: clientSet, namespace: namespace, recorder: recorder}
}

func (dm *defaultDeleteManager) delete(ctx context.Context, restore *v1.Restore) error {
	restoreClient := dm.clientSet.EcosystemV1Alpha1().Restores(dm.namespace)

	_, err := restoreClient.UpdateStatusDeleting(ctx, restore)
	if err != nil {
		return fmt.Errorf("failed to update status [%s] on restore [%s]: %w", v1.RestoreStatusDeleting, restore.Name, err)
	}

	// The provider is still resolved for its readiness gate and its provider-selection event, but the
	// provider child is deleted directly: it is owned by this controller, not by the provider gateway.
	_, err = provider.Get(ctx, restore, restore.Spec.Provider, restore.Namespace, dm.recorder, dm.k8sClient)
	if err != nil {
		return fmt.Errorf("failed to get provider [%s]: %w", restore.Spec.Provider, err)
	}

	err = velero.DeleteRestore(ctx, dm.k8sClient, restore)
	if err != nil {
		return fmt.Errorf("failed to delete restore: %w", err)
	}

	_, err = restoreClient.RemoveFinalizer(ctx, restore, v1.RestoreFinalizer)
	if err != nil {
		return fmt.Errorf("failed to delete finalizer [%s]: %w", v1.RestoreFinalizer, err)
	}

	return nil
}
