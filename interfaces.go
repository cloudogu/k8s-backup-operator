package main

import (
	"github.com/cloudogu/cesapp-lib/registry"
	"github.com/cloudogu/k8s-backup-operator/pkg/additionalimages"
	"github.com/cloudogu/k8s-backup-operator/pkg/cleanup"
	"github.com/cloudogu/k8s-backup-operator/pkg/garbagecollection"
	"github.com/cloudogu/k8s-backup-operator/pkg/provider"
	"github.com/cloudogu/k8s-backup-operator/pkg/scheduledbackup"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type eventRecorder interface {
	record.EventRecorder
}

type controllerManager interface {
	manager.Manager
}

// used for mocks

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
type backupProvider interface {
	provider.Provider
}

//nolint:unused
//goland:noinspection GoUnusedType
type additionalImageGetter interface {
	additionalimages.Getter
}

//nolint:unused
//goland:noinspection GoUnusedType
type additionalImageUpdater interface {
	additionalimages.Updater
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

//nolint:unused
//goland:noinspection GoUnusedType
type k8sClient interface {
	client.Client
}

//nolint:unused
//goland:noinspection GoUnusedType
type cleanupManager interface {
	cleanup.Manager
}
