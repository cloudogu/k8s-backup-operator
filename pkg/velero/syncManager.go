package velero

import (
	"context"
	"errors"
	"fmt"

	backupv1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type defaultSyncManager struct {
	veleroClientSet    veleroClientSet
	ecosystemClientSet ecosystemClientSet
	recorder           eventRecorder
	namespace          string
}

func (d *defaultSyncManager) SyncBackups(ctx context.Context) error {
	// Get all Backup CRs from the cluster
	backupsClient := d.ecosystemClientSet.EcosystemV1Alpha1().Backups(d.namespace)
	backupsList, err := backupsClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("could not list ecosystem backups: %w", err)
	}

	// Create backup map, so we don't have to loop through the backups list
	backupMap := make(map[string]*backupv1.Backup)
	for _, backup := range backupsList.Items {
		backupMap[backup.Name] = &backup
	}

	// Get all Velero backups
	veleroBackups := d.veleroClientSet.VeleroV1().Backups(d.namespace)
	veleroBackupsList, err := veleroBackups.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("could not list velero backups: %w", err)
	}

	// Create Velero backup map, so we don't have to loop through the Velero backups list
	veleroBackupMap := make(map[string]*velerov1.Backup)
	for _, veleroBackup := range veleroBackupsList.Items {
		veleroBackupMap[veleroBackup.Name] = &veleroBackup
	}

	var errs []error

	// Remove Backup CRs which have no corresponding Velero backup
	for _, backup := range backupsList.Items {
		if _, exists := veleroBackupMap[backup.Name]; !exists {
			_, err := backupsClient.RemoveFinalizer(ctx, &backup, backupv1.BackupFinalizer)
			if err != nil {
				errs = append(errs, err)
				break
			}
			err = backupsClient.Delete(ctx, backup.Name, metav1.DeleteOptions{})
			if err != nil {
				errs = append(errs, err)
			}
		}
	}

	// Create Backup CRs for Velero backups that have no counterpart in the cluster yet
	for _, veleroBackup := range veleroBackupsList.Items {
		if _, exists := backupMap[veleroBackup.Name]; !exists {
			newBackup := &backupv1.Backup{
				TypeMeta:   metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{},
				Spec:       backupv1.BackupSpec{Provider: backupv1.ProviderVelero},
				Status: backupv1.BackupStatus{
					Status:              "completed",
					StartTimestamp:      *veleroBackupMap[veleroBackup.Name].Status.StartTimestamp,
					CompletionTimestamp: *veleroBackupMap[veleroBackup.Name].Status.CompletionTimestamp,
				},
			}
			_, err := backupsClient.Create(ctx, newBackup, metav1.CreateOptions{})
			if err != nil {
				errs = append(errs, err)
			}
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("failed to sync backups with velero: %w", errors.Join(errs...))
	}

	return nil
}

// NewDefaultSyncManager creates a new instance of defaultSyncManager.
func NewDefaultSyncManager(veleroClientSet veleroClientSet, ecosystemClientSet ecosystemClientSet, recorder eventRecorder, namespace string) *defaultSyncManager {
	return &defaultSyncManager{veleroClientSet: veleroClientSet, ecosystemClientSet: ecosystemClientSet, recorder: recorder, namespace: namespace}
}