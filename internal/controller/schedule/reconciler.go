package schedule

import (
	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
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

/*
	func (c *defaultReconciler) checkCronJobSynced(ctx context.Context, schedule *backupv1.BackupSchedule, namespace string, logger logr.Logger) error {
		schedule.CronJobName()
		return nil
	}

	func (c *defaultReconciler) checkCBackupScheduleCreated(ctx context.Context, schedule *backupv1.BackupSchedule, namespace string, logger logr.Logger) error {
		return nil
	}
*/

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
