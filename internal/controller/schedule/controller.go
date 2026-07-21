package schedule

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type reconciler interface {
}

type Controller struct {
	client     client.Client
	reconciler reconciler
}
