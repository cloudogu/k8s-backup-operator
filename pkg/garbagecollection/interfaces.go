package garbagecollection

import (
	"context"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/cloudogu/k8s-backup-operator/pkg/api/ecosystem"
	"github.com/cloudogu/k8s-backup-operator/pkg/retention"
)

type Manager interface {
	CollectGarbage(ctx context.Context) error
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

//nolint:unused
//goland:noinspection GoUnusedType
type coreV1 interface {
	corev1.CoreV1Interface
}

//nolint:unused
//goland:noinspection GoUnusedType
type configMapClient interface {
	corev1.ConfigMapInterface
}
