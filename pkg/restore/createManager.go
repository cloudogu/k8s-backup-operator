package restore

import (
	"context"

	v1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
)

type defaultCreateManager struct {
	clientSet ecosystemInterface
	recorder  eventRecorder
}

func newCreateManager(clientSet ecosystemInterface, recorder eventRecorder) *defaultCreateManager {
	return &defaultCreateManager{clientSet: clientSet, recorder: recorder}
}

func (cm *defaultCreateManager) create(ctx context.Context, backup *v1.Restore) error {
	//TODO implement me
	panic("implement me")
}
