package restore

import (
	"context"
	"github.com/cloudogu/k8s-backup-operator/pkg/cleanup"
	"github.com/cloudogu/k8s-registry-lib/repository"
	appsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/cloudogu/k8s-backup-operator/pkg/api/ecosystem"
	v1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
	"github.com/cloudogu/k8s-backup-operator/pkg/provider"
)

type ecosystemInterface interface {
	ecosystem.Interface
}

type eventRecorder interface {
	record.EventRecorder
}

type controllerManager interface {
	ctrl.Manager
}

type restoreManager interface {
	createManager
	deleteManager
}

type createManager interface {
	create(ctx context.Context, restore *v1.Restore) error
}

type deleteManager interface {
	delete(ctx context.Context, restore *v1.Restore) error
}

type requeueHandler interface {
	Handle(ctx context.Context, contextMessage string, restore v1.RequeuableObject, originalErr error, requeueStatus string) (ctrl.Result, error)
}

type maintenanceModeSwitch interface {
	// Activate activates the maintenance mode.
	Activate(ctx context.Context, description repository.MaintenanceModeDescription) error
	// Deactivate deactivates the maintenance mode.
	Deactivate(ctx context.Context) error
}

type cleanupManager interface {
	cleanup.Manager
}

// used for mocks

//nolint:unused
//goland:noinspection GoUnusedType
type ecosystemRestoreInterface interface {
	ecosystem.RestoreInterface
}

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

//nolint:unused
//goland:noinspection GoUnusedType
type ecosystemV1Alpha1Interface interface {
	ecosystem.V1Alpha1Interface
}

//nolint:unused
//goland:noinspection GoUnusedType
type restoreProvider interface {
	provider.Provider
}
