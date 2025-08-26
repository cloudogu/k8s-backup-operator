package garbagecollection

import (
	"context"

	"github.com/cloudogu/k8s-backup-lib/pkg/api/ecosystem"
	"github.com/cloudogu/k8s-backup-operator/pkg/retention"
)

type Manager interface {
	// CollectGarbage deletes backups according to the configured retention strategy.
	CollectGarbage(ctx context.Context) error
}

type ecosystemClientSet interface {
	ecosystem.Interface
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
type ecosystemV1Alpha1 interface {
	ecosystem.V1Alpha1Interface
}

//nolint:unused
//goland:noinspection GoUnusedType
type backupClient interface {
	ecosystem.BackupInterface
}
