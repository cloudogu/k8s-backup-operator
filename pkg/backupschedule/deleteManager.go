package backupschedule

import (
	"context"
	"fmt"
	"github.com/cloudogu/k8s-backup-operator/pkg/retry"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
)

type defaultDeleteManager struct {
	clientSet ecosystemInterface
	recorder  eventRecorder
	namespace string
}

func newDeleteManager(clientSet ecosystemInterface, recorder eventRecorder, namespace string) *defaultDeleteManager {
	return &defaultDeleteManager{clientSet: clientSet, recorder: recorder, namespace: namespace}
}

func (dm *defaultDeleteManager) delete(ctx context.Context, backupSchedule *v1.BackupSchedule) error {
	schedulesClient := dm.clientSet.EcosystemV1Alpha1().BackupSchedules(dm.namespace)
	_, err := schedulesClient.UpdateStatusDeleting(ctx, backupSchedule)
	if err != nil {
		return fmt.Errorf("failed to set status [%s] in backup schedule resource [%s]: %w", v1.BackupScheduleStatusDeleting, backupSchedule.Name, err)
	}

	err = retry.OnError(5, retry.AlwaysRetryFunc, func() error {
		return dm.clientSet.BatchV1().CronJobs(dm.namespace).Delete(ctx, backupSchedule.CronJobName(), metav1.DeleteOptions{})
	})
	if err != nil {
		return fmt.Errorf("failed to delete CronJob %s: %w", backupSchedule.CronJobName(), err)
	}

	_, err = schedulesClient.RemoveFinalizer(ctx, backupSchedule, v1.BackupScheduleFinalizer)
	if err != nil {
		return fmt.Errorf("failed to remove finalizer [%s] in backup schedule resource [%s]: %w", v1.BackupScheduleFinalizer, backupSchedule.Name, err)
	}

	return nil
}
