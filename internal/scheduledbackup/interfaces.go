package scheduledbackup

import (
	"context"

	backupclient "github.com/cloudogu/k8s-backup-lib/client"
	typedv1 "github.com/cloudogu/k8s-backup-lib/client/typed/api/v1"
	operatortime "github.com/cloudogu/k8s-backup-operator/internal/time"
)

type Manager interface {
	ScheduleBackup(ctx context.Context) error
}

type backupClientSet interface {
	backupclient.Interface
}

type timeProvider interface {
	operatortime.TimeProvider
}

//nolint:unused
//goland:noinspection GoUnusedType
type backupClient interface {
	typedv1.BackupInterface
}

//nolint:unused
//goland:noinspection GoUnusedType
type backupV1Client interface {
	typedv1.BackupV1Interface
}
