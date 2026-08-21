package velero

import (
	"time"

	"github.com/cloudogu/k8s-backup-operator/pkg/annotations"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"context"
	"fmt"

	k8sv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// defaultBackupTTL is ten years, basically infinity in backup standards.
const defaultBackupTTL = 87660 * time.Hour

// CreateVeleroBackupResource creates the Velero resource representation of the given backup.
func CreateVeleroBackupResource(backup *k8sv1.Backup, backupStorageName string, labels map[string]string) *velerov1.Backup {
	selectors := []*metav1.LabelSelector{
		{MatchLabels: map[string]string{"k8s.cloudogu.com/type": "global-config"}},
		{MatchExpressions: []metav1.LabelSelectorRequirement{
			{Key: "dogu.name", Operator: metav1.LabelSelectorOpExists},
		}},
		// everything besides dogu-specific config that should be included in the backup, e.g., PVCs of components etc.
		{MatchExpressions: []metav1.LabelSelectorRequirement{
			{Key: "k8s.cloudogu.com/backup-scope", Operator: metav1.LabelSelectorOpExists},
		}},
	}
	volumeFsBackup := false
	return &velerov1.Backup{
		ObjectMeta: metav1.ObjectMeta{
			Name:        backup.Name,
			Namespace:   backup.Namespace,
			Labels:      labels,
			Annotations: annotations.GetBackupAnnotations(backup.ObjectMeta),
		},
		Spec: velerov1.BackupSpec{
			IncludedNamespaces: []string{backup.Namespace},
			IncludedResources: []string{
				"configmaps",
				"secrets",
				"persistentvolumeclaims",
				"persistentvolumes",
				"dogus.k8s.cloudogu.com",
			},
			OrLabelSelectors:         selectors,
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
		log.FromContext(ctx).Info("velero delete backup request not found: create one")
		deleteBackupRequest = &velerov1.DeleteBackupRequest{
			ObjectMeta: metav1.ObjectMeta{Namespace: backup.Namespace, Name: backup.Name},
			Spec:       velerov1.DeleteBackupRequestSpec{BackupName: backup.Name},
		}
		if err = k8sClient.Create(ctx, deleteBackupRequest); err != nil {
			return nil, fmt.Errorf("create velero delete backup request: %w", err)
		}
		return deleteBackupRequest, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get velero delete backup request: %w", err)
	}

	log.FromContext(ctx).Info("velero delete backup request already exists")
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
		return fmt.Errorf("get velero delete backup request while provider backup is running: %w", err)
	}

	log.FromContext(ctx).Info("provider backup is running: delete existing velero delete backup request")
	if err = k8sClient.Delete(ctx, deleteBackupRequest); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("delete velero delete backup request while provider backup is running: %w", err)
	}

	return nil
}
