package schedule

import "sigs.k8s.io/controller-runtime/pkg/client"

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

type defaultReconciler struct {
	client client.Client
}

func NewReconciler(client client.Client) *defaultReconciler {
	return &defaultReconciler{
		client: client,
	}
}
