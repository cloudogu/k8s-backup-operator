package restore

import (
	"context"

	v1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
)

type defaultDeleteManager struct {
	clientSet ecosystemInterface
	recorder  eventRecorder
}

func newDeleteManager(clientSet ecosystemInterface, recorder eventRecorder) *defaultDeleteManager {
	return &defaultDeleteManager{clientSet: clientSet, recorder: recorder}
}

func (dm *defaultDeleteManager) delete(ctx context.Context, restore *v1.Restore) error {
	//TODO implement me
	panic("implement me")
}
