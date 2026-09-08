package main

import (
	"context"

	"github.com/cloudogu/k8s-backup-operator/internal/config"
	schedulecontroller "github.com/cloudogu/k8s-backup-operator/internal/controller/schedule"
	"github.com/cloudogu/k8s-backup-operator/internal/garbagecollection"
	"github.com/cloudogu/k8s-backup-operator/internal/scheduledbackup"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type eventRecorder interface {
	events.EventRecorder
}

// used for mocks

//nolint:unused
//goland:noinspection GoUnusedType
type controllerManager interface {
	manager.Manager
}

//nolint:unused
//goland:noinspection GoUnusedType
type scheduledBackupManager interface {
	scheduledbackup.Manager
}

//nolint:unused
//goland:noinspection GoUnusedType
type gcManager interface {
	garbagecollection.Manager
}

//nolint:unused
//goland:noinspection GoUnusedType
type additionalImageGetter interface {
	schedulecontroller.OperatorImageGetter
}

//nolint:unused
//goland:noinspection GoUnusedType
type configGetter interface {
	config.Getter
}

//nolint:unused
//goland:noinspection GoUnusedType
type k8sClient interface {
	client.Client
}

//nolint:unused
//goland:noinspection GoUnusedType
type cleanupManager interface {
	// Cleanup deletes all resources that need to be deleted before restoring the backup.
	Cleanup(ctx context.Context) error
}
