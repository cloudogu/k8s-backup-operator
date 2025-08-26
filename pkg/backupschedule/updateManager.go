package backupschedule

import (
	"context"
	"errors"
	"fmt"
	"github.com/cloudogu/k8s-backup-operator/pkg/additionalimages"
	"github.com/cloudogu/retry-lib/retry"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/cloudogu/k8s-backup-lib/pkg/api/v1"
)

type defaultUpdateManager struct {
	clientSet   ecosystemInterface
	recorder    eventRecorder
	namespace   string
	imageConfig additionalimages.ImageConfig
}

func newUpdateManager(clientSet ecosystemInterface, recorder eventRecorder, namespace string, imageConfig additionalimages.ImageConfig) *defaultUpdateManager {
	return &defaultUpdateManager{clientSet: clientSet, recorder: recorder, namespace: namespace, imageConfig: imageConfig}
}

func (um *defaultUpdateManager) update(ctx context.Context, backupSchedule *v1.BackupSchedule) error {
	um.recorder.Event(backupSchedule, corev1.EventTypeNormal, v1.UpdateEventReason, "Updating backup schedule")
	backupScheduleName := backupSchedule.Name

	schedulesClient := um.clientSet.EcosystemV1Alpha1().BackupSchedules(um.namespace)
	backupSchedule, err := schedulesClient.UpdateStatusUpdating(ctx, backupSchedule)
	if err != nil {
		return fmt.Errorf("failed to set status [%s] in backup schedule resource [%s]: %w", v1.BackupScheduleStatusUpdating, backupScheduleName, err)
	}

	cronJobClient := um.clientSet.BatchV1().CronJobs(um.namespace)
	err = retry.OnError(maxTries, retry.AlwaysRetryFunc, func() error {
		cronJob, err := cronJobClient.Get(ctx, backupSchedule.CronJobName(), metav1.GetOptions{})
		if err != nil {
			return err
		}

		cronJob.Spec.Schedule = backupSchedule.Spec.Schedule
		cronJob.Spec.JobTemplate.Spec.Template = backupSchedule.CronJobPodTemplate(um.imageConfig.OperatorImage)

		_, err = cronJobClient.Update(ctx, cronJob, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		err = fmt.Errorf("failed to update cron job for backup schedule [%s]: %w", backupScheduleName, err)
		_, updateStatusErr := schedulesClient.UpdateStatusFailed(ctx, backupSchedule)
		if updateStatusErr != nil {
			err = errors.Join(err, fmt.Errorf("failed to update backup schedule status to 'Failed': %w", updateStatusErr))
		}

		return err
	}

	_, err = schedulesClient.UpdateStatusCreated(ctx, backupSchedule)
	if err != nil {
		return fmt.Errorf("failed to set status [%s] in backup schedule resource [%s]: %w", v1.BackupScheduleStatusCreated, backupScheduleName, err)
	}
	return nil
}
