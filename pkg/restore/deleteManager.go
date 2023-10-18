package restore

import (
	"context"

	v1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
)

type defaultDeleteManager struct {
	restoreClient ecosystemRestoreInterface
	recorder      eventRecorder
}

func newDeleteManager(restoreClient ecosystemRestoreInterface, recorder eventRecorder) *defaultDeleteManager {
	return &defaultDeleteManager{restoreClient: restoreClient, recorder: recorder}
}

func (dm *defaultDeleteManager) delete(ctx context.Context, backup *v1.Restore) error {
	// TODO implement me
	panic("implement me")
}
