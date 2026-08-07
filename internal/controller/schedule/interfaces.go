package schedule

import (
	"context"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type OperatorImageGetter interface {
	// ImageForKey returns a container image reference from the operator's additional-images ConfigMap.
	ImageForKey(ctx context.Context, key string) (string, error)
}

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
	MarkAcceptanceNotEvaluated(schedule *backupv1.BackupSchedule, err error)
	MarkReady(schedule *backupv1.BackupSchedule)
	MarkNotReady(schedule *backupv1.BackupSchedule, reason, message string)
}

type MetadataManager interface {
	Ensure(ctx context.Context, schedule *backupv1.BackupSchedule) error
	Remove(ctx context.Context, schedule *backupv1.BackupSchedule) error
}

type reconciler interface {
	Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error)
}
