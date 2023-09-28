package controller

import (
	"github.com/cloudogu/cesapp-lib/registry"
	"github.com/cloudogu/k8s-backup-operator/pkg/api/ecosystem"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
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

// used for mocks

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
