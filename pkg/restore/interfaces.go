package restore

import (
	"context"
	"time"

	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/cloudogu/cesapp-lib/registry"

	"github.com/cloudogu/k8s-backup-operator/pkg/api/ecosystem"
	v1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
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

// requeuableError indicates that the current error requires the operator to requeue the component.
type requeuableError interface {
	error
	// GetRequeueTime returns the time to wait before the next reconciliation.
	GetRequeueTime(requeueTimeNanos time.Duration) time.Duration
}

type requeueHandler interface {
	Handle(ctx context.Context, contextMessage string, restore *v1.Restore, originalErr error, requeueStatus string) (ctrl.Result, error)
}

// used for mocks

//nolint:unused
//goland:noinspection GoUnusedType
type ecosystemRestoreInterface interface {
	ecosystem.RestoreInterface
}

//nolint:unused
//goland:noinspection GoUnusedType
type ecosystemV1Alpha1Interface interface {
	ecosystem.V1Alpha1Interface
}

//nolint:unused
//goland:noinspection GoUnusedType
type etcdRegistry interface {
	registry.Registry
}

//nolint:unused
//goland:noinspection GoUnusedType
type etcdContext interface {
	registry.ConfigurationContext
}
