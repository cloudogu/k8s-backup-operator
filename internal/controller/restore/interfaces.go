package restore

import (
	"context"

	"github.com/cloudogu/k8s-registry-lib/repository"
	appsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/events"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type k8sClient interface {
	client.WithWatch
}

type eventRecorder interface {
	events.EventRecorder
}

type controllerManager interface {
	ctrl.Manager
}

type maintenanceModeSwitch interface {
	// Activate activates the maintenance mode.
	Activate(ctx context.Context, description repository.MaintenanceModeDescription, force bool) error
	// Deactivate deactivates the maintenance mode.
	Deactivate(ctx context.Context, force bool) error
	// GetStatus checks if the maintenance mode is active and returns its contents.
	GetStatus(ctx context.Context) (repository.MaintenanceModeDescription, bool, error)
}

type cleanupManager interface {
	// Cleanup deletes all resources that need to be deleted before restoring the backup.
	Cleanup(ctx context.Context) error
}

type scaleManager interface {
	// ScaleDown finds all resources labeled with the scaledown scope label,
	// stores their current replica count, and scales them to zero.
	ScaleDown(ctx context.Context) error
	// ScaleUp finds all resources labeled with the scaledown scope label and restores
	// the stored replica count. It retains the replicas label for recovery observation.
	ScaleUp(ctx context.Context) error
	// AreWorkloadsReady reports whether every workload still marked for recovery reached
	// the replica count stored in its replicas label and is available.
	AreWorkloadsReady(ctx context.Context) (bool, error)
	// FinalizeScaleUp removes the stored replica labels after workload readiness was observed.
	FinalizeScaleUp(ctx context.Context) error
}

// used for mocks

//nolint:unused
//goland:noinspection GoUnusedType
type statefulSetInterface interface {
	appsv1.StatefulSetInterface
}

//nolint:unused
//goland:noinspection GoUnusedType
type serviceInterface interface {
	corev1.ServiceInterface
}

//nolint:unused
//goland:noinspection GoUnusedType
type appsV1Interface interface {
	appsv1.AppsV1Interface
}

//nolint:unused
//goland:noinspection GoUnusedType
type coreV1Interface interface {
	corev1.CoreV1Interface
}

//nolint:unused
//goland:noinspection GoUnusedType
type configMapInterface interface {
	corev1.ConfigMapInterface
}
