package requeue

import (
	"github.com/cloudogu/k8s-backup-lib/pkg/api/ecosystem"
	"k8s.io/client-go/tools/record"
	"time"
)

type ecosystemInterface interface {
	ecosystem.Interface
}

//nolint:unused
//goland:noinspection GoUnusedType
type ecosystemBackupInterface interface {
	ecosystem.BackupInterface
}

//nolint:unused
//goland:noinspection GoUnusedType
type backupV1Alpha1Interface interface {
	ecosystem.V1Alpha1Interface
}

//nolint:unused
//goland:noinspection GoUnusedType
type ecosystemRestoreInterface interface {
	ecosystem.RestoreInterface
}

//nolint:unused
//goland:noinspection GoUnusedType
type RestoreV1Alpha1Interface interface {
	ecosystem.V1Alpha1Interface
}

// requeuableError indicates that the current error requires the operator to requeue the component.
type requeuableError interface {
	error
	// GetRequeueTime returns the time to wait before the next reconciliation.
	GetRequeueTime(requeueTimeNanos time.Duration) time.Duration
}

type eventRecorder interface {
	record.EventRecorder
}
