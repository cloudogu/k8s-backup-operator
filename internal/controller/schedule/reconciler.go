package schedule

import (
	"context"
	"fmt"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/go-logr/logr"
	batchv1 "k8s.io/api/batch/v1"
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
