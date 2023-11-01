package backupschedule

import (
	"context"
	"fmt"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
	"github.com/cloudogu/k8s-backup-operator/pkg/retry"
)

type defaultCreateManager struct {
	clientSet ecosystemInterface
	recorder  eventRecorder
	namespace string
}

func newCreateManager(clientSet ecosystemInterface, recorder eventRecorder, namespace string) *defaultCreateManager {
	return &defaultCreateManager{clientSet: clientSet, recorder: recorder, namespace: namespace}
}

func (cm *defaultCreateManager) create(ctx context.Context, backupSchedule *v1.BackupSchedule) error {
	cm.recorder.Event(backupSchedule, corev1.EventTypeNormal, v1.CreateEventReason, "Creating backup schedule")

	backupScheduleClient := cm.clientSet.EcosystemV1Alpha1().BackupSchedules(cm.namespace)

	backupScheduleName := backupSchedule.Name
	backupSchedule, err := backupScheduleClient.UpdateStatusCreating(ctx, backupSchedule)
	if err != nil {
		return fmt.Errorf("failed to set status [%s] in backup schedule resource [%s]: %w", v1.BackupScheduleStatusCreating, backupScheduleName, err)
	}

	backupSchedule, err = backupScheduleClient.AddFinalizer(ctx, backupSchedule, v1.BackupScheduleFinalizer)
	if err != nil {
		return fmt.Errorf("failed to add finalizer [%s] in backup schedule resource [%s]: %w", v1.BackupScheduleFinalizer, backupScheduleName, err)
	}

	backupSchedule, err = backupScheduleClient.AddLabels(ctx, backupSchedule)
	if err != nil {
		return fmt.Errorf("failed to add labels to backup schedule resource [%s]: %w", backupScheduleName, err)
	}

	err = cm.createCronJob(ctx, backupSchedule)
	if err != nil {
		return fmt.Errorf("failed to create cron job for backup schedule [%s]: %w", backupScheduleName, err)
	}

	_, err = backupScheduleClient.UpdateStatusCreated(ctx, backupSchedule)
	if err != nil {
		return fmt.Errorf("failed to set status [%s] in backup schedule resource [%s]: %w", v1.BackupScheduleStatusCreated, backupScheduleName, err)
	}

	return nil
}

func (cm *defaultCreateManager) createCronJob(ctx context.Context, schedule *v1.BackupSchedule) error {
	cronJob := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      schedule.CronJobName(),
			Namespace: cm.namespace,
			Labels: map[string]string{
				"app":                      "ces",
				"k8s.cloudogu.com/part-of": "backup",
			},
		},
		Spec: batchv1.CronJobSpec{
			Schedule: schedule.Spec.Schedule,
			JobTemplate: batchv1.JobTemplateSpec{Spec: batchv1.JobSpec{
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "scheduled-backup-creator",
						Namespace: cm.namespace,
						Labels: map[string]string{
							"app":                      "ces",
							"k8s.cloudogu.com/part-of": "backup",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{
							Name:            schedule.CronJobName(),
							Image:           "busybox:latest", // TODO use real image
							ImagePullPolicy: corev1.PullIfNotPresent,
							Args:            schedule.CronJobArgs(),
							Env: []corev1.EnvVar{{Name: "NAMESPACE",
								ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.namespace"}}}},
						}},
						RestartPolicy: corev1.RestartPolicyOnFailure,
					},
				},
			}},
		},
	}

	err := retry.OnError(5, retry.AlwaysRetryFunc, func() error {
		_, err := cm.clientSet.BatchV1().CronJobs(cm.namespace).Create(ctx, cronJob, metav1.CreateOptions{})
		return err
	})
	if err != nil {
		return fmt.Errorf("failed to create CronJob %s: %w", schedule.CronJobName(), err)
	}

	return nil
}
