package velero

import (
	"context"
	"fmt"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	veleroclient "github.com/vmware-tanzu/velero/pkg/client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type defaultProvider struct {
	manager
	veleroClientSet veleroClientSet
	namespace       string
}

// NewDefaultProvider creates a new instance of defaultProvider.
func NewDefaultProvider(namespace string, recorder eventRecorder) (*defaultProvider, error) {
	factory := veleroclient.NewFactory("k8s-backup-operator", map[string]interface{}{"namespace": namespace})
	clientSet, err := factory.Client()
	if err != nil {
		return nil, fmt.Errorf("failed to create velero clientset: %w", err)
	}
	return &defaultProvider{manager: NewDefaultManager(clientSet, recorder), veleroClientSet: clientSet, namespace: namespace}, nil
}

// CheckReady validates that velero is installed and can establish a connection to its backup store.
func (p *defaultProvider) CheckReady(ctx context.Context) error {
	defaultBsl, err := p.veleroClientSet.VeleroV1().BackupStorageLocations(p.namespace).Get(ctx, defaultStorageLocation, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get backup storage location from cluster: %w", err)
	}

	if defaultBsl.Status.Phase != velerov1.BackupStorageLocationPhaseAvailable {
		return fmt.Errorf("velero is unable to reach the default backup storage location: %s", defaultBsl.Status.Message)
	}

	return nil
}
