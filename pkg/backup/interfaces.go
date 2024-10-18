package backup

import (
	"context"
	"github.com/cloudogu/k8s-backup-operator/pkg/api/ecosystem"
	v1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
	"github.com/cloudogu/k8s-backup-operator/pkg/provider"
	"github.com/cloudogu/k8s-registry-lib/config"
	"github.com/cloudogu/k8s-registry-lib/repository"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
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
	Activate(ctx context.Context, description repository.MaintenanceModeDescription) error
	Deactivate(ctx context.Context) error
}

type backupControllerManager interface {
	createManager
	deleteManager
	statusSyncManager
}

type createManager interface {
	create(ctx context.Context, backup *v1.Backup) error
}

type deleteManager interface {
	delete(ctx context.Context, backup *v1.Backup) error
}

type statusSyncManager interface {
	syncStatus(ctx context.Context, backup *v1.Backup) error
}

type requeueHandler interface {
	Handle(ctx context.Context, contextMessage string, backup v1.RequeuableObject, originalErr error, requeueStatus string) (ctrl.Result, error)
}

// used for mocks

//nolint:unused
//goland:noinspection GoUnusedType
type backupV1Alpha1Interface interface {
	ecosystem.V1Alpha1Interface
}

//nolint:unused
//goland:noinspection GoUnusedType
type backupCoreV1Interface interface {
	corev1.CoreV1Interface
}

//nolint:unused
//goland:noinspection GoUnusedType
type backupConfigMapInterface interface {
	corev1.ConfigMapInterface
}

//nolint:unused
//goland:noinspection GoUnusedType
type backupProvider interface {
	provider.Provider
}

type globalConfigRepository interface {
	Get(ctx context.Context) (config.GlobalConfig, error)
	Update(ctx context.Context, globalConfig config.GlobalConfig) (config.GlobalConfig, error)
}
