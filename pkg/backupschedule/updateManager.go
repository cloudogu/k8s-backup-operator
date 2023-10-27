package backupschedule

import (
	"context"
	"fmt"
	"github.com/cloudogu/k8s-backup-operator/pkg/retry"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
)

type defaultUpdateManager struct {
	clientSet ecosystemInterface
	recorder  eventRecorder
	namespace string
}

func newUpdateManager(clientSet ecosystemInterface, recorder eventRecorder, namespace string) *defaultUpdateManager {
	return &defaultUpdateManager{clientSet: clientSet, recorder: recorder, namespace: namespace}
}

func (um *defaultUpdateManager) update(ctx context.Context, backupSchedule *v1.BackupSchedule) error {
	schedulesClient := um.clientSet.EcosystemV1Alpha1().BackupSchedules(um.namespace)
	_, err := schedulesClient.UpdateStatusUpdating(ctx, backupSchedule)
	if err != nil {
		return fmt.Errorf("failed to set status [%s] in backup schedule resource [%s]: %w", v1.BackupScheduleStatusUpdating, backupSchedule.Name, err)
	}

	cronJobClient := um.clientSet.BatchV1().CronJobs(um.namespace)
	err = retry.OnError(5, retry.AlwaysRetryFunc, func() error {
		cronJob, err := cronJobClient.Get(ctx, backupSchedule.CronJobName(), metav1.GetOptions{})
		if err != nil {
			return err
		}

		cronJob.Spec.Schedule = backupSchedule.Spec.Schedule
		cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Command = backupSchedule.CronJobCommand()

		_, err = cronJobClient.Update(ctx, cronJob, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to update CronJob %s: %w", backupSchedule.CronJobName(), err)
	}

	_, err = schedulesClient.UpdateStatusCreated(ctx, backupSchedule)
	if err != nil {
		return fmt.Errorf("failed to set status [%s] in backup schedule resource [%s]: %w", v1.BackupScheduleStatusCreated, backupSchedule.Name, err)
	}
	return nil
}
