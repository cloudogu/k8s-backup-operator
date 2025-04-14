package cleanup

import (
	"context"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"k8s.io/client-go/discovery"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Manager provides functionality to clean up the cluster before a restore.
type Manager interface {
	Cleanup(ctx context.Context) error
}

type k8sClient interface {
	client.Client
}

type discoveryInterface interface {
	discovery.DiscoveryInterface
}

type configMapClient interface {
	v1.ConfigMapInterface
}
