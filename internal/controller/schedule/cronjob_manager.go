package schedule

import (
	"context"
	"fmt"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/k8s-backup-operator/internal/logging"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var defaultLabels = map[string]string{
	"app":                          "ces",
	"k8s.cloudogu.com/part-of":     "backup",
	"app.kubernetes.io/created-by": "k8s-backup-operator",
	"app.kubernetes.io/part-of":    "k8s-backup-operator",
}

type defaultCronJobManager struct {
	client.Client
	recorder         eventRecorder
	scheme           *runtime.Scheme
	operatorImage    string
	pullPolicy       corev1.PullPolicy
	imagePullSecrets []corev1.LocalObjectReference
}

func (c defaultCronJobManager) ensure(ctx context.Context, schedule *backupv1.BackupSchedule) error {

	cronJob := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      schedule.CronJobName(),
			Namespace: schedule.Namespace,
		},
	}

	operation, err := controllerutil.CreateOrPatch(ctx, c.Client, cronJob, func() error {
		if cronJob.Labels == nil {
			cronJob.Labels = map[string]string{}
		}
		for key, value := range defaultLabels {
			cronJob.Labels[key] = value
		}

		// set as the owner for deletion steps
		if err := controllerutil.SetControllerReference(schedule, cronJob, c.scheme); err != nil {
			return fmt.Errorf("failed to set BackupSchedule %s as owner of CronJob %s: %w", schedule.Name, cronJob.Name, err)
		}

		podTemplate := schedule.CronJobPodTemplate(c.operatorImage, c.pullPolicy)
		if len(c.imagePullSecrets) > 0 {
			podTemplate.Spec.ImagePullSecrets = c.imagePullSecrets
		}

		cronJob.Spec.Schedule = schedule.Spec.Schedule
		cronJob.Spec.JobTemplate.Spec.Template = podTemplate

		return nil
	})
	if err != nil {
		c.recorder.Eventf(
			schedule, nil, corev1.EventTypeWarning, backupv1.CronJobSynchronizationFailedEventReason, actionSynchronizeCronJob,
			"Failed to synchronize CronJob %q: %v", cronJob.Name, err,
		)
		return fmt.Errorf("failed to synchronize CronJob %s for BackupSchedule %s: %w", cronJob.Name, schedule.Name, err)
	}

	switch operation {
	case controllerutil.OperationResultCreated:
		c.recorder.Eventf(
			schedule, cronJob,
			corev1.EventTypeNormal, backupv1.CronJobCreatedEventReason, actionCreateCronJob,
			"Created CronJob %q.", cronJob.Name,
		)

	case controllerutil.OperationResultUpdated:
		c.recorder.Eventf(
			schedule, cronJob,
			corev1.EventTypeNormal, backupv1.CronJobUpdatedEventReason, actionUpdateCronJob,
			"Updated CronJob %q.", cronJob.Name,
		)

	case controllerutil.OperationResultNone:
		logging.Debug(ctx, "CronJob is up to date", "cronJob", cronJob.Name)
	}

	if operation != controllerutil.OperationResultNone {
		logging.Info(ctx, "synchronized CronJob", "cronJob", cronJob.Name, "operation", operation)
	}

	return nil
}

func (c defaultCronJobManager) delete(ctx context.Context, schedule *backupv1.BackupSchedule) error {

	cronJob := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      schedule.CronJobName(),
			Namespace: schedule.Namespace,
		},
	}

	err := c.Delete(ctx, cronJob)
	if apierrors.IsNotFound(err) {
		logging.Debug(ctx, "CronJob is already deleted", "cronJob", cronJob.Name)
		return nil
	}
	if err != nil {
		c.recorder.Eventf(schedule, nil, corev1.EventTypeWarning, backupv1.CronJobDeletionFailedEventReason, actionDeleteCronJob,
			"Failed to delete CronJob %q: %v", cronJob.Name, err,
		)
		return fmt.Errorf("failed to delete CronJob %s for BackupSchedule %s: %w", cronJob.Name, schedule.Name, err)
	}

	c.recorder.Eventf(
		schedule, nil,
		corev1.EventTypeNormal, backupv1.CronJobDeletionRequestedEventReason, actionDeleteCronJob,
		"Requested deletion of CronJob %q.", cronJob.Name,
	)

	logging.Info(ctx, "requested CronJob deletion", "cronJob", cronJob.Name)
	return nil
}
