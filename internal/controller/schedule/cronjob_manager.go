package schedule

import (
	"context"
	"fmt"

	v1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
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

type cronJobManager struct {
	client.Client
	scheme           *runtime.Scheme
	operatorImage    string
	pullPolicy       corev1.PullPolicy
	imagePullSecrets []corev1.LocalObjectReference
}

func (c cronJobManager) Ensure(ctx context.Context, schedule *v1.BackupSchedule) error {
	cronJob := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      schedule.CronJobName(),
			Namespace: schedule.Namespace,
		},
	}

	_, err := controllerutil.CreateOrPatch(ctx, c.Client, cronJob, func() error {
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

	return nil
}

func (c cronJobManager) Delete(ctx context.Context, schedule *v1.BackupSchedule) error {
	cronJob := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      schedule.CronJobName(),
			Namespace: schedule.Namespace,
		},
	}

    // ignore not found errors because we only want to delete here anyway
	if err := client.IgnoreNotFound(c.Client.Delete(ctx, cronJob)); err != nil {
		return fmt.Errorf("failed to delete CronJob %s for BackupSchedule %s: %w", cronJob.Name, schedule.Name, err)
	}

	return nil
}
