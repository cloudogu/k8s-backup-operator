package schedule

import (
	"context"
	"fmt"
	"maps"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/k8s-backup-operator/pkg/config"
	"github.com/go-logr/logr"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// TODO: put into backup-lib
const (
	// ConditionTypeReady indicates if the BackupSchedule is fully reconciled and operational.
	ConditionTypeReady = "Ready"

	// ConditionTypeCronJobSynced indicates if the underlying CronJob is successfully synced tp this crd
	ConditionTypeCronJobSynced = "CronJobSynced"
)

// TODO: put into backup-lib
const (
	// Reasons for ConditionTypeReady
	ReasonScheduleActive    = "ScheduleActive"
	ReasonScheduleSuspended = "ScheduleSuspended"
	ReasonInvalidSpec       = "InvalidSpec"

	// Reasons for ConditionTypeCronJobSynced
	ReasonCronJobSyncSuccessful = "CronJobSyncSuccessful"
	ReasonCronJobSyncFailed     = "CronJobSyncFailed"
)

var defaultLabels = map[string]string{
	"app":                          "ces",
	"k8s.cloudogu.com/part-of":     "backup",
	"app.kubernetes.io/created-by": "k8s-backup-operator",
	"app.kubernetes.io/part-of":    "k8s-backup-operator",
}

// Reconciler handles reconciliation logic for BackupSchedule resources.
type Reconciler interface {
	markAsSyncedToCronJob(schedule *backupv1.BackupSchedule) error
}

type defaultReconciler struct {
	client client.Client
}

// NewReconciler creates a new Reconciler instance.
func NewReconciler(client client.Client) Reconciler {
	return &defaultReconciler{
		client: client,
	}
}

func (c *defaultReconciler) checkCronJobSynced(ctx context.Context, schedule *backupv1.BackupSchedule, namespace string, logger logr.Logger) (bool, error) {
	scheduleFromBackupSchedule := schedule.Spec.Schedule
	expectedCronJobName := schedule.CronJobName()

	// List all CronJobs in the namespace
	cronJobList := &batchv1.CronJobList{}
	err := c.client.List(ctx, cronJobList, &client.ListOptions{
		Namespace: namespace,
	})
	if err != nil {
		logger.Error(err, "Failed to list CronJobs")
		return false, fmt.Errorf("failed to list cronjobs: %w", err)
	}

	// Check if any CronJob matches this schedule
	var matchingCronJob *batchv1.CronJob
	for i := range cronJobList.Items {
		cronJob := &cronJobList.Items[i]
		if cronJob.Name == expectedCronJobName && cronJob.Spec.Schedule == scheduleFromBackupSchedule {
			matchingCronJob = cronJob
			break
		}
	}

	if matchingCronJob == nil {
		logger.Info("No matching CronJob found for BackupSchedule", "expectedName", expectedCronJobName, "schedule", scheduleFromBackupSchedule)
		return false, fmt.Errorf("no matching cronjob found for backup schedule")
	}

	logger.Info("CronJob successfully synced with BackupSchedule", "cronJobName", matchingCronJob.Name)
	return true, nil
}

func (c *defaultReconciler) createCronJob(ctx context.Context, schedule *backupv1.BackupSchedule, namespace string, logger logr.Logger) error {
	labels := make(map[string]string)
	maps.Copy(labels, defaultLabels)

	cronJob := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      schedule.CronJobName(),
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: batchv1.CronJobSpec{
			Schedule: schedule.Spec.Schedule,
			JobTemplate: batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					//TODO: operatorImage & imagePullSecrets
					Template: getCronJobTemplate(schedule, "", config.GetStagePullPolicy(), nil),
				},
			},
		},
	}
	createErr := c.client.Create(ctx, cronJob)

	if createErr != nil {
		return fmt.Errorf("failed to create CronJob %s: %w", schedule.CronJobName(), createErr)
	}

	return nil
}

// set conditions

func (c *defaultReconciler) markAsSyncedToCronJob(schedule *backupv1.BackupSchedule) error {
	synced := metav1.Condition{
		Type:    ConditionTypeCronJobSynced,
		Status:  metav1.ConditionTrue,
		Reason:  ReasonCronJobSyncSuccessful,
		Message: "Cron job synced to backup schedule resource",
	}
	scheduleStatus := &schedule.Status
	meta.SetStatusCondition(&scheduleStatus.Conditions, synced)

	return nil
}

func (c *defaultReconciler) markAsNotSyncedToCronJob(schedule *backupv1.BackupSchedule) error {
	synced := metav1.Condition{
		Type:    ConditionTypeCronJobSynced,
		Status:  metav1.ConditionFalse,
		Reason:  ReasonCronJobSyncFailed,
		Message: "Could not sync cronjob to backup schedule",
	}
	scheduleStatus := &schedule.Status
	meta.SetStatusCondition(&scheduleStatus.Conditions, synced)

	return nil
}

func (c *defaultReconciler) markAsReady(schedule *backupv1.BackupSchedule) error {
	synced := metav1.Condition{
		Type:    ConditionTypeReady,
		Status:  metav1.ConditionTrue,
		Reason:  ReasonScheduleActive,
		Message: "Backup schedule ready and active",
	}
	scheduleStatus := &schedule.Status
	meta.SetStatusCondition(&scheduleStatus.Conditions, synced)

	return nil
}

func (c *defaultReconciler) markAsNotReady(schedule *backupv1.BackupSchedule) error {
	synced := metav1.Condition{
		Type:    ConditionTypeReady,
		Status:  metav1.ConditionFalse,
		Reason:  ReasonScheduleSuspended,
		Message: "Backup schedule not ready",
	}
	scheduleStatus := &schedule.Status
	meta.SetStatusCondition(&scheduleStatus.Conditions, synced)

	return nil
}

// helper

func getCronJobTemplate(schedule *backupv1.BackupSchedule, operatorImage string, pullPolicy corev1.PullPolicy, imagePullSecrets []corev1.LocalObjectReference) corev1.PodTemplateSpec {
	podTemplateSpec := schedule.CronJobPodTemplate(operatorImage, pullPolicy)

	if len(imagePullSecrets) > 0 {
		podTemplateSpec.Spec.ImagePullSecrets = imagePullSecrets
	}

	return podTemplateSpec
}
