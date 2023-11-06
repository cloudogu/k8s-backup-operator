package backupschedule

import (
	"context"
	"errors"
	"fmt"
	"github.com/cloudogu/k8s-backup-operator/pkg/retry"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
)

type defaultDeleteManager struct {
	clientSet ecosystemInterface
	recorder  eventRecorder
	namespace string
}

func newScheduleDeleteManager(clientSet ecosystemInterface, recorder eventRecorder, namespace string) *defaultDeleteManager {
	return &defaultDeleteManager{clientSet: clientSet, recorder: recorder, namespace: namespace}
}

func (dm *defaultDeleteManager) delete(ctx context.Context, backupSchedule *v1.BackupSchedule) error {
	dm.recorder.Event(backupSchedule, corev1.EventTypeNormal, v1.DeleteEventReason, "Deleting backup schedule")

	schedulesClient := dm.clientSet.EcosystemV1Alpha1().BackupSchedules(dm.namespace)
	_, err := schedulesClient.UpdateStatusDeleting(ctx, backupSchedule)
	if err != nil {
		return fmt.Errorf("failed to set status [%s] in backup schedule resource [%s]: %w", v1.BackupScheduleStatusDeleting, backupSchedule.Name, err)
	}

	err = retry.OnError(5, retry.AlwaysRetryFunc, func() error {
		return dm.clientSet.BatchV1().CronJobs(dm.namespace).Delete(ctx, backupSchedule.CronJobName(), metav1.DeleteOptions{})
	})
	if err != nil {
		err = fmt.Errorf("failed to delete cron job for backup schedule [%s]: %w", backupSchedule.Name, err)
		_, updateStatusErr := schedulesClient.UpdateStatusFailed(ctx, backupSchedule)
		if updateStatusErr != nil {
			err = errors.Join(err, fmt.Errorf("failed to update backup schedule status to 'Failed': %w", updateStatusErr))
		}

		return err
	}

	_, err = schedulesClient.RemoveFinalizer(ctx, backupSchedule, v1.BackupScheduleFinalizer)
	if err != nil {
		return fmt.Errorf("failed to remove finalizer [%s] in backup schedule resource [%s]: %w", v1.BackupScheduleFinalizer, backupSchedule.Name, err)
	}

	return nil
}
