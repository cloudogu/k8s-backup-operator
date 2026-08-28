package scheduledbackup

import (
	"context"

	"github.com/cloudogu/k8s-backup-lib/api/ecosystem"
	operatortime "github.com/cloudogu/k8s-backup-operator/internal/time"
)

type Manager interface {
	ScheduleBackup(ctx context.Context) error
}

type ecosystemClientSet interface {
	ecosystem.Interface
}

type timeProvider interface {
	operatortime.TimeProvider
}

//nolint:unused
//goland:noinspection GoUnusedType
type ecosystemBackupInterface interface {
	ecosystem.BackupInterface
}

//nolint:unused
//goland:noinspection GoUnusedType
type ecosystemV1Alpha1Interface interface {
	ecosystem.V1Alpha1Interface
}
