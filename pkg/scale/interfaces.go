package scale

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Manager handles scaling down and up of workload resources during restore.
type Manager interface {
	// ScaleDown finds all resources labeled with the scaledown scope label,
	// stores their current replica count, and scales them to zero.
	ScaleDown(ctx context.Context) error
	// ScaleUp finds all resources labeled with the scaledown scope label,
	// reads the stored replica count, restores it, and removes the replicas label.
	ScaleUp(ctx context.Context) error
}

type k8sClient interface {
	client.WithWatch
}
