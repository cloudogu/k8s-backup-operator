package additionalimages

import (
	"context"
	"errors"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"

	v1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
)

type updater struct {
	clientSet    ecosystemClientSet
	namespace    string
	kubectlImage string
}

func NewUpdater(clientSet ecosystemClientSet, namespace string, kubectlImage string) Updater {
	return &updater{clientSet: clientSet, namespace: namespace, kubectlImage: kubectlImage}
}

// Update sets the newest additional images wherever they are needed.
// E.g., the kubectl image used in the CronJob of a BackupSchedule.
func (bsp *updater) Update(ctx context.Context) error {
	backupScheduleClient := bsp.clientSet.EcosystemV1Alpha1().BackupSchedules(bsp.namespace)
	imageNotUpToDate := fields.OneTermNotEqualSelector("status.currentKubectlImage", bsp.kubectlImage)
	scheduleList, err := backupScheduleClient.List(ctx, metav1.ListOptions{FieldSelector: imageNotUpToDate.String()})
	if err != nil {
		return fmt.Errorf("failed to list backup schedules whose images are not up to date: %w", err)
	}

	var errs []error
	for _, backupSchedule := range scheduleList.Items {
		err = bsp.patchCronJob(ctx, &backupSchedule)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to update additional images in cron job %s: %w", backupSchedule.CronJobName(), err))
			continue
		}

		err = bsp.updateCurrentImage(ctx, &backupSchedule, backupScheduleClient)
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

func (bsp *updater) patchCronJob(ctx context.Context, schedule *v1.BackupSchedule) error {
	cronJobClient := bsp.clientSet.BatchV1().CronJobs(bsp.namespace)
	cronJob, err := cronJobClient.Get(ctx, schedule.CronJobName(), metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get cron job %s: %w", schedule.CronJobName(), err)
	}

	cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image = bsp.kubectlImage
	cronJob, err = cronJobClient.Update(ctx, cronJob, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update kubectl image in cron job %s: %w", schedule.CronJobName(), err)
	}

	return nil
}

func (bsp *updater) updateCurrentImage(ctx context.Context, schedule *v1.BackupSchedule, client backupScheduleClient) error {
	schedule.Status.CurrentKubectlImage = bsp.kubectlImage
	_, err := client.UpdateStatus(ctx, schedule, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update current kubectl image in status of backup schedule %s: %w", schedule.Name, err)
	}

	return nil
}
