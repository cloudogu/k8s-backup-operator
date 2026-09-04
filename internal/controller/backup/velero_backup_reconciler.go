package backup

import (
	"context"
	"fmt"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/k8s-backup-operator/internal/annotations"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type defaultVeleroBackupReconciler struct {
	client client.Client
}

func NewVeleroBackupReconciler(client client.Client) *defaultVeleroBackupReconciler {
	return &defaultVeleroBackupReconciler{client: client}
}

func (r *defaultVeleroBackupReconciler) ensureBackupExists(ctx context.Context, veleroBackup *velerov1.Backup) error {
	backup := &backupv1.Backup{}
	err := r.client.Get(ctx, client.ObjectKeyFromObject(veleroBackup), backup)
	if err == nil {
		return nil
	}
	if !apierrors.IsNotFound(err) {
		return fmt.Errorf("get cloudogu backup: %w", err)
	}

	backup = &backupv1.Backup{
		ObjectMeta: metav1.ObjectMeta{
			Name:        veleroBackup.Name,
			Namespace:   veleroBackup.Namespace,
			Labels:      copyDefaultLabels(),
			Annotations: annotations.GetBackupAnnotations(veleroBackup.ObjectMeta),
			Finalizers:  []string{backupv1.BackupFinalizer},
		},
		Spec: backupv1.BackupSpec{
			Provider:           backupv1.ProviderVelero,
			SyncedFromProvider: true,
		},
	}

	if err = r.client.Create(ctx, backup); err != nil {
		return fmt.Errorf("create cloudogu backup: %w", err)
	}

	return nil
}

func (r *defaultVeleroBackupReconciler) deleteBackupIfExists(ctx context.Context, key client.ObjectKey) error {
	backup := &backupv1.Backup{}
	if err := r.client.Get(ctx, key, backup); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("get cloudogu backup: %w", err)
	}

	if err := r.ensureDeleteBackupRequestExists(ctx, key); err != nil {
		return err
	}

	// Do not delete canceled backups, they stay for failure history, whereas it's provider backup is always deleted.
	if meta.IsStatusConditionTrue(backup.Status.Conditions, backupv1.ConditionCanceled) {
		return nil
	}

	if err := r.client.Delete(ctx, backup); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("delete cloudogu backup: %w", err)
	}

	return nil
}

func (r *defaultVeleroBackupReconciler) ensureDeleteBackupRequestExists(ctx context.Context, key client.ObjectKey) error {
	deleteRequest := &velerov1.DeleteBackupRequest{}
	if err := r.client.Get(ctx, key, deleteRequest); err == nil {
		return nil
	} else if !apierrors.IsNotFound(err) {
		return fmt.Errorf("get velero delete backup request: %w", err)
	}

	deleteRequest = &velerov1.DeleteBackupRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      key.Name,
			Namespace: key.Namespace,
		},
		Spec: velerov1.DeleteBackupRequestSpec{
			BackupName: key.Name,
		},
	}
	if err := r.client.Create(ctx, deleteRequest); err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("create velero delete backup request: %w", err)
	}

	return nil
}

func copyDefaultLabels() map[string]string {
	labels := make(map[string]string, len(defaultLabels))
	for key, value := range defaultLabels {
		labels[key] = value
	}
	return labels
}
