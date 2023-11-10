package garbagecollection

import (
	"context"

	"github.com/cloudogu/k8s-backup-operator/pkg/api/ecosystem"
)

type manager struct {
	clientSet ecosystemClientSet
	namespace string
}

func NewManager(clientSet ecosystem.Interface, namespace string) Manager {
	return &manager{clientSet: clientSet, namespace: namespace}
}

func (m *manager) CollectGarbage(ctx context.Context) error {
	//TODO implement me
	panic("implement me")
}
