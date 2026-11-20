package garbagecollection

import (
	"context"

	backupclient "github.com/cloudogu/k8s-backup-lib/client"
	typedv1 "github.com/cloudogu/k8s-backup-lib/client/typed/api/v1"
	"github.com/cloudogu/k8s-backup-operator/internal/retention"
)

type Manager interface {
	// CollectGarbage deletes backups according to the configured retention strategy.
	CollectGarbage(ctx context.Context) error
}

type backupClientSet interface {
	backupclient.Interface
}

type strategyGetter interface {
	// Get returns the Strategy identified by the given name.
	Get(name retention.StrategyId) (retention.Strategy, error)
}

type strategy interface {
	retention.Strategy
}

// The following interfaces are here to generate mocks.

//nolint:unused
//goland:noinspection GoUnusedType
type backupV1Client interface {
	typedv1.BackupV1Interface
}

//nolint:unused
//goland:noinspection GoUnusedType
type backupClient interface {
	typedv1.BackupInterface
}
