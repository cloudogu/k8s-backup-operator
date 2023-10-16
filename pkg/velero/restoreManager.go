package velero

import (
	"context"
	v1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
)

type defaultRestoreManager struct {
}

// NewDefaultRestoreManager creates a new instance of defaultRestoreManager.
func NewDefaultRestoreManager() *defaultRestoreManager {
	return &defaultRestoreManager{}
}

// CreateRestore creates a restore according to the restore configuration in v1.Restore.
func (rm *defaultRestoreManager) CreateRestore(ctx context.Context, restore *v1.Restore) error {
	// TODO
	panic("implement me")
}
