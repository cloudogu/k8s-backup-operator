package backup

import (
	"context"
	"github.com/cloudogu/cesapp-lib/registry"
	"github.com/cloudogu/k8s-backup-operator/pkg/api/ecosystem"
	v1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
	"github.com/cloudogu/k8s-backup-operator/pkg/provider"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
)

type ecosystemInterface interface {
	ecosystem.Interface
}

type ecosystemBackupInterface interface {
	ecosystem.BackupInterface
}

type controllerManager interface {
	ctrl.Manager
}

type eventRecorder interface {
	record.EventRecorder
}

type MaintenanceModeSwitch interface {
	ActivateMaintenanceMode(ctx context.Context, title string, text string) error
	DeactivateMaintenanceMode(ctx context.Context) error
}

type backupControllerManager interface {
	createManager
	deleteManager
}

type createManager interface {
	create(ctx context.Context, backup *v1.Backup) error
}

type deleteManager interface {
	delete(ctx context.Context, backup *v1.Backup) error
}

type requeueHandler interface {
	Handle(ctx context.Context, contextMessage string, backup v1.RequeuableObject, originalErr error, requeueStatus string) (ctrl.Result, error)
}

type etcdRegistry interface {
	registry.Registry
}

// used for mocks

//nolint:unused
//goland:noinspection GoUnusedType
type configurationContext interface {
	registry.ConfigurationContext
}

//nolint:unused
//goland:noinspection GoUnusedType
type backupV1Alpha1Interface interface {
	ecosystem.V1Alpha1Interface
}

//nolint:unused
//goland:noinspection GoUnusedType
type backupProvider interface {
	provider.Provider
}
