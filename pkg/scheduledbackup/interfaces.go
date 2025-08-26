package scheduledbackup

import (
	"context"
	time2 "github.com/cloudogu/k8s-backup-operator/pkg/time"

	"github.com/cloudogu/k8s-backup-lib/pkg/api/ecosystem"
)

type Manager interface {
	ScheduleBackup(ctx context.Context) error
}

type ecosystemClientSet interface {
	ecosystem.Interface
}

type timeProvider interface {
	time2.TimeProvider
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
