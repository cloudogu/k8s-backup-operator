package velero

import (
	"context"
	"fmt"

	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// ReasonBackupStorageLocationAvailable reports a provider that can serve backup and restore requests.
	ReasonBackupStorageLocationAvailable = "BackupStorageLocationAvailable"
	// ReasonBackupStorageLocationNotFound reports a missing backup storage location, which usually
	// means that Velero is not installed or not configured yet.
	ReasonBackupStorageLocationNotFound = "BackupStorageLocationNotFound"
	// ReasonBackupStorageLocationNotAvailable reports a backup storage location that Velero cannot reach.
	ReasonBackupStorageLocationNotAvailable = "BackupStorageLocationNotAvailable"
)

// Readiness reports whether the provider can serve backup and restore requests. Reason and Message
// are written to the conditions of the backup or restore that waits for the provider, so they
// describe the provider state rather than the resource that observed it.
type Readiness struct {
	Ready   bool
	Reason  string
	Message string
}

// CheckReady reports whether the provider is ready to serve backup and restore requests. A provider
// that is not ready yet is reported through the Readiness, API server errors are returned as an error.
func CheckReady(ctx context.Context, k8sClient client.Client, namespace string, backupStorageName string) (Readiness, error) {
	backupStorageLocation := &velerov1.BackupStorageLocation{}
	err := k8sClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: backupStorageName}, backupStorageLocation)
	if apierrors.IsNotFound(err) {
		return Readiness{
			Reason:  ReasonBackupStorageLocationNotFound,
			Message: fmt.Sprintf("The velero backup storage location 'name=%s' was not found.", backupStorageName),
		}, nil
	}
	if err != nil {
		return Readiness{}, fmt.Errorf("get velero backup storage location 'name=%s': %w", backupStorageName, err)
	}

	if backupStorageLocation.Status.Phase != velerov1.BackupStorageLocationPhaseAvailable {
		return Readiness{
			Reason: ReasonBackupStorageLocationNotAvailable,
			Message: fmt.Sprintf("The velero backup storage location 'name=%s' is not available (phase: %s).",
				backupStorageName, backupStorageLocation.Status.Phase),
		}, nil
	}

	return Readiness{
		Ready:   true,
		Reason:  ReasonBackupStorageLocationAvailable,
		Message: fmt.Sprintf("The velero backup storage location 'name=%s' is available.", backupStorageName),
	}, nil
}
