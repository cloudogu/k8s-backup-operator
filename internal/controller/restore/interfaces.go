package restore

import (
	"context"

	"github.com/cloudogu/k8s-registry-lib/repository"
	appsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/k8s-backup-operator/pkg/provider"
)

type k8sClient interface {
	client.WithWatch
}

type eventRecorder interface {
	record.EventRecorder
}

type controllerManager interface {
	ctrl.Manager
}

// restoreManager is the deletion side of the workflow. Creating a restore is no longer a manager
// operation but a sequence of reconciler stages.
type restoreManager interface {
	deleteManager
}

type deleteManager interface {
	delete(ctx context.Context, restore *v1.Restore) error
}

type maintenanceModeSwitch interface {
	// Activate activates the maintenance mode.
	Activate(ctx context.Context, description repository.MaintenanceModeDescription, force bool) error
	// Deactivate deactivates the maintenance mode.
	Deactivate(ctx context.Context, force bool) error
}

type cleanupManager interface {
	// Cleanup deletes all resources that need to be deleted before restoring the backup.
	Cleanup(ctx context.Context) error
}

type restoreProvider interface {
	provider.Provider
}

type scaleManager interface {
	// ScaleDown finds all resources labeled with the scaledown scope label,
	// stores their current replica count, and scales them to zero.
	ScaleDown(ctx context.Context) error
	// ScaleUp finds all resources labeled with the scaledown scope label,
	// reads the stored replica count, restores it, and removes the replicas label.
	ScaleUp(ctx context.Context) error
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
