package additionalimages

import (
	"context"
	"errors"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/log"

	k8sErr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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
func (bsp *updater) Update(ctx context.Context) error {
	logger := log.FromContext(ctx)
	logger.Info("Updating additional images")

	return bsp.updateKubectlImages(ctx)
}

func (bsp *updater) updateKubectlImages(ctx context.Context) error {
	logger := log.FromContext(ctx)
	logger.Info("Updating kubectl images")

	backupScheduleClient := bsp.clientSet.EcosystemV1Alpha1().BackupSchedules(bsp.namespace)
	scheduleList, err := backupScheduleClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list backup schedules whose images are not up to date: %w", err)
	}

	var errs []error
	for _, backupSchedule := range scheduleList.Items {
		if backupSchedule.Status.CurrentKubectlImage == bsp.kubectlImage {
			// image is up-to-date, nothing to do
			continue
		}

		err = bsp.patchCronJob(ctx, &backupSchedule)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to update additional images in cron job %s: %w", backupSchedule.CronJobName(), err))
			continue
		}

		err = bsp.updateCurrentImage(ctx, &backupSchedule, backupScheduleClient)
		errs = append(errs, err)
	}

	logger.Info("Successfully updated kubectl images")
	return errors.Join(errs...)
}

func (bsp *updater) patchCronJob(ctx context.Context, schedule *v1.BackupSchedule) error {
	logger := log.FromContext(ctx)

	cronJobClient := bsp.clientSet.BatchV1().CronJobs(bsp.namespace)
	cronJob, err := cronJobClient.Get(ctx, schedule.CronJobName(), metav1.GetOptions{})
	if k8sErr.IsNotFound(err) {
		message := fmt.Sprintf("Cron job %s for backup schedule %s does not exist. Skipping kubectl image update.", schedule.CronJobName(), schedule.Name)
		logger.Error(err, message)
		bsp.recorder.Event(schedule, corev1.EventTypeWarning, imageUpdateEventReason, message)
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to get cron job %s: %w", schedule.CronJobName(), err)
	}

	cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image = bsp.kubectlImage
	cronJob, err = cronJobClient.Update(ctx, cronJob, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update kubectl image in cron job %s: %w", schedule.CronJobName(), err)
	}

	bsp.recorder.Eventf(schedule, corev1.EventTypeNormal, imageUpdateEventReason, "Updated kubectl image in CronJob to %s.", bsp.kubectlImage)
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
