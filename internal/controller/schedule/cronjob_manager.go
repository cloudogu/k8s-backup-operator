package schedule

import (
	"context"
	"fmt"

	v1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var defaultLabels = map[string]string{
	"app":                          "ces",
	"k8s.cloudogu.com/part-of":     "backup",
	"app.kubernetes.io/created-by": "k8s-backup-operator",
	"app.kubernetes.io/part-of":    "k8s-backup-operator",
}

type defaultCronJobManager struct {
	client.Client
	scheme           *runtime.Scheme
	operatorImage    string
	pullPolicy       corev1.PullPolicy
	imagePullSecrets []corev1.LocalObjectReference
}

func (c defaultCronJobManager) ensure(ctx context.Context, schedule *v1.BackupSchedule) error {
	logger := log.FromContext(ctx)

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
		return fmt.Errorf("failed to create CronJob %s for BackupSchedule %s: %w", cronJob.Name, schedule.Name, err)
	}

	if operation == controllerutil.OperationResultNone {
		logger.V(1).Info("CronJob is up to date", "cronJob", cronJob.Name)
	} else {
		logger.Info("synchronized CronJob", "cronJob", cronJob.Name, "operation", operation)
	}

	return nil
}

func (c defaultCronJobManager) delete(ctx context.Context, schedule *v1.BackupSchedule) error {
	logger := log.FromContext(ctx)

	cronJob := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      schedule.CronJobName(),
			Namespace: schedule.Namespace,
		},
	}

	err := c.Client.Delete(ctx, cronJob)
	if apierrors.IsNotFound(err) {
		logger.V(1).Info("CronJob is already deleted", "cronJob", cronJob.Name)
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to delete CronJob %s for BackupSchedule %s: %w", cronJob.Name, schedule.Name, err)
	}

	logger.Info("requested CronJob deletion", "cronJob", cronJob.Name)
	return nil
}
