package main

import (
	"github.com/cloudogu/cesapp-lib/registry"
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
