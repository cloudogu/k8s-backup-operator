package backup

import (
	"context"
	"time"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var defaultLabels = map[string]string{
	"app":                      "ces",
	"k8s.cloudogu.com/part-of": "backup",
}

const (
	blueprintIdAnnotation    = "backup.cloudogu.com/blueprintId"
	blueprintDogusAnnotation = "backup.cloudogu.com/dogus"
)

type Clock interface {
	Now() time.Time
}

type ServiceImpl struct {
	client client.Client
	clock  Clock
}

func NewService(client client.Client, clock Clock) *ServiceImpl {
	return &ServiceImpl{
		client: client,
		clock:  clock,
	}
}

func (srv *ServiceImpl) reconcileBackup(context context.Context, backup *backupv1.Backup) error {
	//TODO implement me
	panic("implement me")
}

func (srv *ServiceImpl) cancelBackup(context context.Context, backup *backupv1.Backup) error {
	//TODO implement me
	panic("implement me")
}

func (srv *ServiceImpl) deleteBackup(context context.Context, backup *backupv1.Backup) error {
	//TODO implement me
	panic("implement me")
}
