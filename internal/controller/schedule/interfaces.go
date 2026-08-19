package schedule

import (
	"context"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
)

type OperatorImageGetter interface {
	// ImageForKey returns a container image reference from the operator's additional-images ConfigMap.
	ImageForKey(ctx context.Context, key string) (string, error)
}

type eventRecorder interface {
	record.EventRecorder
}

type cronJobManager interface {
	ensure(ctx context.Context, schedule *backupv1.BackupSchedule) error
	delete(ctx context.Context, schedule *backupv1.BackupSchedule) error
}

type validator interface {
	validate(schedule *backupv1.BackupSchedule) error
}

type conditionManager interface {
	markAccepted(schedule *backupv1.BackupSchedule)
	markInvalid(schedule *backupv1.BackupSchedule, err error)
	markAcceptanceNotEvaluated(schedule *backupv1.BackupSchedule, err error)
	markReady(schedule *backupv1.BackupSchedule)
	markNotReady(schedule *backupv1.BackupSchedule, reason, message string)
}

type metadataManager interface {
	ensure(ctx context.Context, schedule *backupv1.BackupSchedule) error
	remove(ctx context.Context, schedule *backupv1.BackupSchedule) error
}

type reconciler interface {
	reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error)
}
