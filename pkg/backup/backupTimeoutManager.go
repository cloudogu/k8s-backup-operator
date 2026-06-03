package backup

import (
	"context"
	"fmt"

	k8sv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/k8s-backup-operator/pkg/config"
	"github.com/cloudogu/k8s-backup-operator/pkg/metrics"
)

type backupTimeoutManager struct {
	k8sClient           k8sClient
	clientSet           ecosystemInterface
	namespace           string
	recorder            eventRecorder
	backupTimeoutGetter config.Getter
}

// newBackupTimeoutManager creates a new instance of backupTimeoutManager.
func newBackupTimeoutManager(k8sClient k8sClient, clientSet ecosystemInterface, namespace string, recorder eventRecorder, backupTimeoutGetter config.Getter) *backupTimeoutManager {
	return &backupTimeoutManager{k8sClient: k8sClient, clientSet: clientSet, namespace: namespace, recorder: recorder, backupTimeoutGetter: backupTimeoutGetter}
}

// when the time since the backup was created exceeds the backupRetryTimeLimit the backup is set to failed
func (btm *backupTimeoutManager) timeout(ctx context.Context, backup *k8sv1.Backup) error {
	backupClient := btm.clientSet.EcosystemV1Alpha1().Backups(btm.namespace)

	updatedBackup, updateStatusErr := backupClient.UpdateStatusFailed(ctx, backup)
	if updateStatusErr != nil {
		return fmt.Errorf("failed to update backups status to 'Failed': %w", updateStatusErr)
	}
	if updatedBackup != nil {
		// use the updated backup to prevent the reconciler from caching the old status
		*backup = *updatedBackup
	}

	metrics.UpdateBackupStatusMetrics(btm.namespace, backup.Name, k8sv1.BackupStatusFailed)

	backupRetryTimeLimit, err := btm.backupTimeoutGetter.GetRetryLimit(ctx)
	if err != nil {
		return fmt.Errorf("failed to get backup timeout: %w", err)
	}

	return fmt.Errorf("backup retry time limit (%d minutes) exceeded", backupRetryTimeLimit)
}
