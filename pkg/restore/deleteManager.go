package restore

import (
	"context"
	v1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
)

type defaultDeleteManager struct {
}

func (dm *defaultDeleteManager) delete(ctx context.Context, backup *v1.Restore) error {
	//TODO implement me
	panic("implement me")
}
