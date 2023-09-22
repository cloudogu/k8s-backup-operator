package velero

import (
	"context"
	"github.com/cloudogu/k8s-backup-operator/pkg/api/ecosystem"
	v1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
	"time"
)

type provider struct{}

func New(client ecosystem.BackupInterface, recorder eventRecorder) *provider {
	return &provider{}
}

func (p *provider) CreateBackup(ctx context.Context, backup *v1.Backup) error {
	println("Hier könnte ihre Backup-Lösung verwendet werden")
	time.Sleep(time.Second * 60)
	println("Ende")
	return nil
}
