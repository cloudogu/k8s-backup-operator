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
	clientSet    ecosystemClientSet
	namespace    string
	kubectlImage string
	recorder     eventRecorder
}

const imageUpdateEventReason = "ImageUpdate"

func NewUpdater(clientSet ecosystem.Interface, namespace string, kubectlImage string, recorder record.EventRecorder) Updater {
	return &updater{clientSet: clientSet, namespace: namespace, kubectlImage: kubectlImage, recorder: recorder}
}

// Update sets the newest additional images wherever they are needed.
// E.g., the kubectl image used in the CronJob of a BackupSchedule.
func (bsu *updater) Update(ctx context.Context) error {
	logger := log.FromContext(ctx)
	logger.Info("Updating additional images")

	logger.Info("Updating kubectl images")
	err := bsu.updateKubectlImages(ctx)
	if err != nil {
		return fmt.Errorf("failed to update kubectl images: %w", err)
	}

	logger.Info("Successfully updated additional images")

	return nil
}

func (bsu *updater) updateKubectlImages(ctx context.Context) error {
	logger := log.FromContext(ctx)
	logger.Info("Updating kubectl images")

	backupScheduleClient := bsu.clientSet.EcosystemV1Alpha1().BackupSchedules(bsu.namespace)
	scheduleList, err := backupScheduleClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list backup schedules whose images are not up to date: %w", err)
	}

	updatedImages := 0
	var errs []error
	for _, backupSchedule := range scheduleList.Items {
		if backupSchedule.Status.CurrentKubectlImage == bsu.kubectlImage {
			// image is up-to-date, nothing to do
			continue
		}

		err = bsu.patchCronJob(ctx, &backupSchedule)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to update additional images in cron job %s: %w", backupSchedule.CronJobName(), err))
			continue
		}

		err = bsu.updateCurrentImage(ctx, &backupSchedule, backupScheduleClient)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		updatedImages += 1
	}

	logger.Info(fmt.Sprintf("Updated %d kubectl images, encountered %d errors", updatedImages, len(errs)))
	return errors.Join(errs...)
}

func (bsu *updater) patchCronJob(ctx context.Context, schedule *v1.BackupSchedule) error {
	logger := log.FromContext(ctx)

	cronJobClient := bsu.clientSet.BatchV1().CronJobs(bsu.namespace)
	cronJob, err := cronJobClient.Get(ctx, schedule.CronJobName(), metav1.GetOptions{})
	if k8sErr.IsNotFound(err) {
		message := fmt.Sprintf("Cron job %s for backup schedule %s does not exist. Skipping kubectl image update.", schedule.CronJobName(), schedule.Name)
		logger.Error(err, message)
		bsu.recorder.Event(schedule, corev1.EventTypeWarning, imageUpdateEventReason, message)
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to get cron job %s: %w", schedule.CronJobName(), err)
	}

	cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image = bsu.kubectlImage
	_, err = cronJobClient.Update(ctx, cronJob, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update kubectl image in cron job %s: %w", schedule.CronJobName(), err)
	}

	bsu.recorder.Eventf(schedule, corev1.EventTypeNormal, imageUpdateEventReason, "Updated kubectl image in CronJob to %s.", bsu.kubectlImage)
	return nil
}

func (bsu *updater) updateCurrentImage(ctx context.Context, schedule *v1.BackupSchedule, client backupScheduleClient) error {
	schedule.Status.CurrentKubectlImage = bsu.kubectlImage
	_, err := client.UpdateStatus(ctx, schedule, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update current kubectl image in status of backup schedule %s: %w", schedule.Name, err)
	}

	return nil
}
