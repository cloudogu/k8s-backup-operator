package garbagecollection

import (
	"context"
	"github.com/cloudogu/k8s-backup-operator/pkg/api/ecosystem"
	"github.com/cloudogu/k8s-backup-operator/pkg/retention"

	"k8s.io/client-go/kubernetes"
)

type Manager interface {
	CollectGarbage(ctx context.Context) error
}

type clientSet interface {
	kubernetes.Interface
}

type ecosystemClientSet interface {
	ecosystem.Interface
}

type configGetter interface {
	GetConfig(ctx context.Context) (retention.Config, error)
}

type strategyGetter interface {
	Get(name retention.StrategyId) (retention.Strategy, error)
}
