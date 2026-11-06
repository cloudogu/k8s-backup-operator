package velero

import (
	"time"

	"context"
	"fmt"

	"github.com/cloudogu/k8s-backup-operator/internal/logging"
	"sigs.k8s.io/controller-runtime/pkg/client"

	k8sv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// defaultBackupTTL is ten years, basically infinity in backup standards.
const defaultBackupTTL = 87660 * time.Hour

// CreateVeleroUnitBackupResource creates the Velero resource representation of a single backup unit.
// The selector restricts the backup to the resources of the unit.
func CreateVeleroUnitBackupResource(
	namespace string,
	name string,
	backupStorageName string,
	labels map[string]string,
	annotations map[string]string,
	selector *metav1.LabelSelector,
	includedResources []string,
) *velerov1.Backup {
	volumeFsBackup := false
	return &velerov1.Backup{
		Name:        name,
		Namespace:   namespace,
		Labels:      labels,
		Annotations: annotations,
		Spec: velerov1.BackupSpec{
			IncludedNamespaces:       []string{namespace},
			IncludedResources:        includedResources,
			LabelSelector:            selector,
			TTL:                      metav1.Duration{Duration: defaultBackupTTL},
			StorageLocation:          backupStorageName,
			DefaultVolumesToFsBackup: &volumeFsBackup,
		},
	}
}

// CreateVeleroDeleteBackupRequestIfNotExists returns the Velero DeleteBackupRequest belonging to the given backup.
// If the request does not exist, it is created.
func CreateVeleroDeleteBackupRequestIfNotExists(ctx context.Context, k8sClient client.Client, backup *k8sv1.Backup) (*velerov1.DeleteBackupRequest, error) {
	deleteBackupRequest := &velerov1.DeleteBackupRequest{}
	err := k8sClient.Get(ctx, client.ObjectKeyFromObject(backup), deleteBackupRequest)
	if apierrors.IsNotFound(err) {
		logging.Info(ctx, "velero delete backup request not found: create one")
		deleteBackupRequest = &velerov1.DeleteBackupRequest{
			Namespace: backup.Namespace, Name: backup.Name,
			Spec: velerov1.DeleteBackupRequestSpec{BackupName: backup.Name},
		}
		if err = k8sClient.Create(ctx, deleteBackupRequest); err != nil {
			return nil, fmt.Errorf("create velero delete backup request: %w", err)
		}
		return deleteBackupRequest, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get velero delete backup request: %w", err)
	}

	logging.Info(ctx, "velero delete backup request already exists")
	return deleteBackupRequest, nil
}

// DeleteVeleroDeleteBackupRequestIfExists deletes the Velero DeleteBackupRequest belonging to the given backup.
// An already absent request is not an error.
func DeleteVeleroDeleteBackupRequestIfExists(ctx context.Context, k8sClient client.Client, backup *k8sv1.Backup) error {
	deleteBackupRequest := &velerov1.DeleteBackupRequest{}
	err := k8sClient.Get(ctx, client.ObjectKeyFromObject(backup), deleteBackupRequest)
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("error getting velero delete backup request: %w", err)
	}

	logging.Info(ctx, "delete existing velero delete backup request")
	if err = k8sClient.Delete(ctx, deleteBackupRequest); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("delete velero delete backup request while provider backup is running: %w", err)
	}

	return nil
}
