package schedule

import (
	"context"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type CronJobManager interface {
	Ensure(ctx context.Context, schedule *backupv1.BackupSchedule) error
	Delete(ctx context.Context, schedule *backupv1.BackupSchedule) error
}

type Validator interface {
	Validate(schedule *backupv1.BackupSchedule) error
}

type ConditionManager interface {
	MarkAccepted(schedule *backupv1.BackupSchedule)
	MarkInvalid(schedule *backupv1.BackupSchedule, err error)
	MarkCronJobSynced(schedule *backupv1.BackupSchedule)
	MarkCronJobNotSynced(schedule *backupv1.BackupSchedule, err error)
	MarkDeleting(schedule *backupv1.BackupSchedule)
	ComputeReady(schedule *backupv1.BackupSchedule)
}

type MetadataManager interface {
	Ensure(ctx context.Context, schedule *backupv1.BackupSchedule) error
	Remove(ctx context.Context, schedule *backupv1.BackupSchedule) error
}

type reconciler interface {
	Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error)
}
