package backupschedule

import (
	"context"
	typedbatchv1 "k8s.io/client-go/kubernetes/typed/batch/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/cloudogu/k8s-backup-operator/pkg/api/ecosystem"
	v1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
)

type ecosystemInterface interface {
	ecosystem.Interface
}

type eventRecorder interface {
	record.EventRecorder
}

type requeueHandler interface {
	Handle(ctx context.Context, contextMessage string, backupSchedule v1.RequeuableObject, originalErr error, requeueStatus string) (ctrl.Result, error)
}

type Manager interface {
	createManager
	updateManager
	deleteManager
}

type createManager interface {
	create(ctx context.Context, backupSchedule *v1.BackupSchedule) error
}

type updateManager interface {
	update(ctx context.Context, backupSchedule *v1.BackupSchedule) error
}

type deleteManager interface {
	delete(ctx context.Context, backupSchedule *v1.BackupSchedule) error
}

type controllerManager interface {
	ctrl.Manager
}

//nolint:unused
//goland:noinspection GoUnusedType
type ecosystemBackupScheduleInterface interface {
	ecosystem.BackupScheduleInterface
}

//nolint:unused
//goland:noinspection GoUnusedType
type ecosystemV1Alpha1Interface interface {
	ecosystem.V1Alpha1Interface
}

//nolint:unused
//goland:noinspection GoUnusedType
type batchV1Interface interface {
	typedbatchv1.BatchV1Interface
}

//nolint:unused
//goland:noinspection GoUnusedType
type cronJobInterface interface {
	typedbatchv1.CronJobInterface
}
