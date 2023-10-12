package velero

import (
	"context"
	"fmt"

	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type defaultReadinessChecker struct {
	veleroClientSet veleroClientSet
	namespace       string
}

func newReadinessChecker(veleroClientSet veleroClientSet, namespace string) *defaultReadinessChecker {
	return &defaultReadinessChecker{veleroClientSet: veleroClientSet, namespace: namespace}
}

// CheckReady validates that velero is installed and can establish a connection to its backup store.
func (rc *defaultReadinessChecker) CheckReady(ctx context.Context) error {
	defaultBsl, err := rc.veleroClientSet.VeleroV1().BackupStorageLocations(rc.namespace).Get(ctx, defaultStorageLocation, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get backup storage location from cluster: %w", err)
	}

	if defaultBsl.Status.Phase != velerov1.BackupStorageLocationPhaseAvailable {
		return fmt.Errorf("velero is unable to reach the default backup storage location: %s", defaultBsl.Status.Message)
	}

	return nil
}
