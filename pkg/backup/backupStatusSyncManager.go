package backup

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
	"github.com/cloudogu/k8s-backup-operator/pkg/provider"
)

type backupStatusSyncManager struct {
	clientSet ecosystemInterface
	namespace string
	recorder  eventRecorder
}

func (bsm *backupStatusSyncManager) syncStatus(ctx context.Context, backup *v1.Backup) error {
	logger := log.FromContext(ctx)

	startMessage := fmt.Sprintf("Syncing status of backup %q with its corresponding provider backup", backup.Name)
	logger.Info(startMessage)
	bsm.recorder.Event(backup, corev1.EventTypeNormal, v1.SyncStatusEventReason, startMessage)

	backupProvider, err := provider.Get(ctx, backup, backup.Spec.Provider, backup.Namespace, bsm.recorder, bsm.clientSet)
	if err != nil {
		return fmt.Errorf("failed to get backup provider: %w", err)
	}

	err = backupProvider.SyncBackupStatus(ctx, backup)
	if err != nil {
		return fmt.Errorf("failed to sync status of backup %q with its corresponding %q backup: %w", backup.Name, backup.Spec.Provider, err)
	}

	completionMessage := fmt.Sprintf("Successfully synced status of backup %q with its corresponding provider backup", backup.Name)
	logger.Info(completionMessage)
	bsm.recorder.Event(backup, corev1.EventTypeNormal, v1.SyncStatusEventReason, completionMessage)

	return nil
}

func newBackupStatusSyncManager(clientSet ecosystemInterface, namespace string, recorder eventRecorder) *backupStatusSyncManager {
	return &backupStatusSyncManager{clientSet: clientSet, namespace: namespace, recorder: recorder}
}
