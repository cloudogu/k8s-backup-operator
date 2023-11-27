package restore

import (
	"context"
	"github.com/cloudogu/k8s-backup-operator/pkg/cleanup"
	appsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	v12 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/cloudogu/cesapp-lib/registry"
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
	// ActivateMaintenanceMode activates the maintenance mode.
	ActivateMaintenanceMode(ctx context.Context, title string, text string) error
	// DeactivateMaintenanceMode deactivates the maintenance mode.
	DeactivateMaintenanceMode(ctx context.Context) error
}

type cesRegistry interface {
	registry.Registry
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
	v12.ServiceInterface
}

//nolint:unused
//goland:noinspection GoUnusedType
type appsV1Interface interface {
	appsv1.AppsV1Interface
}

type coreV1Interface interface {
	v12.CoreV1Interface
}

//nolint:unused
//goland:noinspection GoUnusedType
type ecosystemV1Alpha1Interface interface {
	ecosystem.V1Alpha1Interface
}

//nolint:unused
//goland:noinspection GoUnusedType
type configurationContext interface {
	registry.ConfigurationContext
}

//nolint:unused
//goland:noinspection GoUnusedType
type restoreProvider interface {
	provider.Provider
}
