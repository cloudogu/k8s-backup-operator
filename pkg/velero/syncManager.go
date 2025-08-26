package velero

import (
	"context"
	"errors"
	"fmt"
	"github.com/cloudogu/retry-lib/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	backupv1 "github.com/cloudogu/k8s-backup-lib/pkg/api/v1"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type defaultSyncManager struct {
	k8sClient k8sWatchClient
	recorder  eventRecorder
	namespace string
}

// SyncBackups syncs backup CRs with velero CRs
func (d *defaultSyncManager) SyncBackups(ctx context.Context) error {
	// Get all Backup CRs from the cluster
	backupsList := &backupv1.BackupList{}
	err := d.k8sClient.List(ctx, backupsList, &client.ListOptions{Namespace: d.namespace})
	if err != nil {
		return fmt.Errorf("could not list ecosystem backups: %w", err)
	}

	// Create backup map, so we don't have to loop through the backups list
	backupMap := make(map[string]*backupv1.Backup)
	for _, backup := range backupsList.Items {
		backupMap[backup.Name] = &backup
	}

	// Get all Velero backups
	veleroBackups := &velerov1.BackupList{}
	err = d.k8sClient.List(ctx, veleroBackups, &client.ListOptions{Namespace: d.namespace})
	if err != nil {
		return fmt.Errorf("could not list velero backups: %w", err)
	}

	// Create Velero backup map, so we don't have to loop through the Velero backups list
	veleroBackupMap := make(map[string]*velerov1.Backup)
	for _, veleroBackup := range veleroBackups.Items {
		veleroBackupMap[veleroBackup.Name] = &veleroBackup
	}

	var errs []error

	// Remove Backup CRs which have no corresponding Velero backup
	removeErrs := removeBackupsWithoutVeleroBackup(ctx, backupsList, veleroBackupMap, d.k8sClient)
	errs = append(errs, removeErrs...)

	// Create Backup CRs for Velero backups that have no counterpart in the cluster yet
	createErrs := createBackupsForVeleroBackups(ctx, veleroBackups, backupMap, d.k8sClient)
	errs = append(errs, createErrs...)

	if len(errs) > 0 {
		return fmt.Errorf("failed to sync backups with velero: %w", errors.Join(errs...))
	}

	return nil
}

func createBackupsForVeleroBackups(ctx context.Context, veleroBackupsList *velerov1.BackupList, backupMap map[string]*backupv1.Backup, k8sClient k8sWatchClient) []error {
	var errs []error
	for _, veleroBackup := range veleroBackupsList.Items {
		if _, exists := backupMap[veleroBackup.Name]; !exists {
			newBackup := &backupv1.Backup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      veleroBackup.Name,
					Namespace: veleroBackup.Namespace,
					Labels:    map[string]string{"app": "ces", "k8s.cloudogu.com/part-of": "backup"},
				},
				Spec: backupv1.BackupSpec{
					Provider:           backupv1.ProviderVelero,
					SyncedFromProvider: true,
				},
			}
			err := k8sClient.Create(ctx, newBackup)
			if err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errs
}

func removeBackupsWithoutVeleroBackup(ctx context.Context, backupsList *backupv1.BackupList, veleroBackupMap map[string]*velerov1.Backup, k8sClient k8sWatchClient) []error {
	var errs []error
	for _, backup := range backupsList.Items {
		if _, exists := veleroBackupMap[backup.Name]; !exists {
			backupPtr := &backup
			err := retry.OnConflict(func() error {
				err := k8sClient.Get(ctx, backup.GetNamespacedName(), backupPtr)
				if err != nil {
					return err
				}

				controllerutil.RemoveFinalizer(backupPtr, backupv1.BackupFinalizer)
				err = k8sClient.Update(ctx, backupPtr)
				if err != nil {
					return err
				}

				return nil
			})
			if err != nil {
				errs = append(errs, err)
				break
			}

			err = k8sClient.Delete(ctx, backupPtr)
			if err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errs
}

// SyncBackupStatus syncs the status of the backup CR with the corresponding velero backup.
// The velero backup must be completed or an error is thrown.
func (d *defaultSyncManager) SyncBackupStatus(ctx context.Context, backup *backupv1.Backup) error {
	veleroBackup := &velerov1.Backup{}
	// we can use backup.GetNamespacedName() here because velero backups are named the same as their corresponding cloudogu backup
	err := d.k8sClient.Get(ctx, backup.GetNamespacedName(), veleroBackup)
	if err != nil {
		return fmt.Errorf("failed to find corresponding velero backup for backup %q: %w", backup.Name, err)
	}

	if veleroBackup.Status.Phase != velerov1.BackupPhaseCompleted {
		return fmt.Errorf("velero backup %q is not completed and therefore cannot be synced", veleroBackup.Name)
	}

	err = retry.OnConflict(func() error {
		updatedBackup := &backupv1.Backup{}
		err = d.k8sClient.Get(ctx, backup.GetNamespacedName(), updatedBackup)
		if err != nil {
			return err
		}

		updatedBackup.Status.Status = backupv1.BackupStatusCompleted
		updatedBackup.Status.StartTimestamp = *veleroBackup.Status.StartTimestamp
		updatedBackup.Status.CompletionTimestamp = *veleroBackup.Status.CompletionTimestamp

		err = d.k8sClient.Status().Update(ctx, updatedBackup)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to update status of backup %q: %w", backup.Name, err)
	}

	return nil
}

// newDefaultSyncManager creates a new instance of defaultSyncManager.
func newDefaultSyncManager(k8sClient k8sWatchClient, recorder eventRecorder, namespace string) *defaultSyncManager {
	return &defaultSyncManager{k8sClient: k8sClient, recorder: recorder, namespace: namespace}
}
