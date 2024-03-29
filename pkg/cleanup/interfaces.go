package cleanup

import (
	"context"

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
