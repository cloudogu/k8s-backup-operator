package garbagecollection

import (
	"context"

	"github.com/cloudogu/k8s-backup-operator/internal/retention"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Manager interface {
	// CollectGarbage deletes backups according to the configured retention strategy.
	CollectGarbage(ctx context.Context) error
}

type k8sClient interface {
	client.Client
}

type strategyGetter interface {
	// Get returns the Strategy identified by the given name.
	Get(name retention.StrategyId) (retention.Strategy, error)
}

type strategy interface {
	retention.Strategy
}
