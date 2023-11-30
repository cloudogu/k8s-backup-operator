package additionalimages

import (
	"context"
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	k8sErr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/cloudogu/k8s-backup-operator/pkg/api/ecosystem"
	v1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
)

type updater struct {
	clientSet ecosystemClientSet
	namespace string
	recorder  eventRecorder
}

const imageUpdateEventReason = "ImageUpdate"

func NewUpdater(clientSet ecosystem.Interface, namespace string, recorder record.EventRecorder) Updater {
	return &updater{clientSet: clientSet, namespace: namespace, recorder: recorder}
}

type ImageConfig struct {
	OperatorImage string
}

// Update sets the newest additional images wherever they are needed.
// E.g., the image used in the CronJob of a BackupSchedule.
func (bsu *updater) Update(ctx context.Context, config ImageConfig) error {
	logger := log.FromContext(ctx)
	logger.Info("Updating additional images")

	err := bsu.updateOperatorImages(ctx, config.OperatorImage)
	if err != nil {
		return fmt.Errorf("failed to update backup schedule cron job images: %w", err)
	}

	logger.Info("Successfully updated additional images")

	return nil
}

func (bsu *updater) updateOperatorImages(ctx context.Context, image string) error {
	logger := log.FromContext(ctx)
	logger.Info("Updating backup schedule cron job images")

	backupScheduleClient := bsu.clientSet.EcosystemV1Alpha1().BackupSchedules(bsu.namespace)
	scheduleList, err := backupScheduleClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list backup schedules whose images are not up to date: %w", err)
	}

	updatedImages := 0
	var errs []error
	for _, backupSchedule := range scheduleList.Items {
		if backupSchedule.Status.CurrentCronJobImage == image {
			// image is up-to-date, nothing to do
			continue
		}

		err = bsu.patchCronJob(ctx, &backupSchedule, image)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to update image in cron job %s: %w", backupSchedule.CronJobName(), err))
			continue
		}

		err = bsu.updateCurrentCronJobImage(ctx, &backupSchedule, backupScheduleClient, image)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		updatedImages += 1
	}

	logger.Info(fmt.Sprintf("Updated %d backup schedule cron job images, encountered %d errors", updatedImages, len(errs)))
	return errors.Join(errs...)
}

func (bsu *updater) patchCronJob(ctx context.Context, schedule *v1.BackupSchedule, image string) error {
	logger := log.FromContext(ctx)

	cronJobClient := bsu.clientSet.BatchV1().CronJobs(bsu.namespace)
	cronJob, err := cronJobClient.Get(ctx, schedule.CronJobName(), metav1.GetOptions{})
	if k8sErr.IsNotFound(err) {
		message := fmt.Sprintf("Cron job %s for backup schedule %s does not exist. Skipping cron job image update.", schedule.CronJobName(), schedule.Name)
		logger.Error(err, message)
		bsu.recorder.Event(schedule, corev1.EventTypeWarning, imageUpdateEventReason, message)
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to get cron job %s: %w", schedule.CronJobName(), err)
	}

	cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image = image
	_, err = cronJobClient.Update(ctx, cronJob, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update image in backup schedule cron job %s: %w", schedule.CronJobName(), err)
	}

	bsu.recorder.Eventf(schedule, corev1.EventTypeNormal, imageUpdateEventReason, "Updated image in backup schedule cron job to %s.", image)
	return nil
}

func (bsu *updater) updateCurrentCronJobImage(ctx context.Context, schedule *v1.BackupSchedule, client backupScheduleClient, image string) error {
	schedule.Status.CurrentCronJobImage = image
	_, err := client.UpdateStatus(ctx, schedule, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update current cron job image in status of backup schedule %s: %w", schedule.Name, err)
	}

	return nil
}
