package velero

import (
	"context"
	"fmt"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"k8s.io/apimachinery/pkg/types"
)

type defaultProvider struct {
	manager
	k8sClient k8sWatchClient
	namespace string
}

// NewDefaultProvider creates a new instance of defaultProvider.
func NewDefaultProvider(k8sClient k8sWatchClient, namespace string, recorder eventRecorder) *defaultProvider {
	return &defaultProvider{manager: NewDefaultManager(k8sClient, recorder, namespace), k8sClient: k8sClient, namespace: namespace}
}

// CheckReady validates that velero is installed and can establish a connection to its backup store.
func (p *defaultProvider) CheckReady(ctx context.Context) error {
	defaultBsl := &velerov1.BackupStorageLocation{}
	err := p.k8sClient.Get(ctx, types.NamespacedName{
		Namespace: p.namespace,
		Name:      defaultStorageLocation,
	}, defaultBsl)
	if err != nil {
		return fmt.Errorf("failed to get backup storage location from cluster: %w", err)
	}

	if defaultBsl.Status.Phase != velerov1.BackupStorageLocationPhaseAvailable {
		return fmt.Errorf("velero is unable to reach the default backup storage location: %s", defaultBsl.Status.Message)
	}

	return nil
}
