package garbagecollection

import (
	"context"
	"github.com/cloudogu/k8s-backup-operator/pkg/api/ecosystem"

	"k8s.io/client-go/kubernetes"
)

type Manager interface {
	CollectGarbage(ctx context.Context) error
}

type clientSet interface {
	kubernetes.Interface
}

type ecosystemClientSet interface {
	ecosystem.Interface
}
