package velero

import (
	"context"

	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/record"

	"github.com/vmware-tanzu/velero/pkg/generated/clientset/versioned"
	velerov1 "github.com/vmware-tanzu/velero/pkg/generated/clientset/versioned/typed/velero/v1"
)

type readinessChecker interface {
	// CheckReady validates if the provider is ready to receive backup requests.
	CheckReady(ctx context.Context) error
}

type eventRecorder interface {
	record.EventRecorder
}

type veleroClientSet interface {
	versioned.Interface
}

// The following interfaces are here to generate mocks.

//nolint:unused
//goland:noinspection GoUnusedType
type veleroInterface interface {
	velerov1.VeleroV1Interface
}

//nolint:unused
//goland:noinspection GoUnusedType
type veleroBackupInterface interface {
	velerov1.BackupInterface
}

//nolint:unused
//goland:noinspection GoUnusedType
type veleroBackupStorageLocationInterface interface {
	velerov1.BackupStorageLocationInterface
}

//nolint:unused
//goland:noinspection GoUnusedType
type veleroDeleteBackupRequest interface {
	velerov1.DeleteBackupRequestInterface
}

//nolint:unused
//goland:noinspection GoUnusedType
type ecosystemWatch interface {
	watch.Interface
}
