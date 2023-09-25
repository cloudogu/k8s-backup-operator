package velero

import (
	"context"
	"github.com/cloudogu/k8s-backup-operator/pkg/api/ecosystem"
	v1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
	corev1 "k8s.io/api/core/v1"
	"time"
)

type provider struct {
	recorder eventRecorder
}

func New(client ecosystem.BackupInterface, recorder eventRecorder) *provider {
	return &provider{recorder: recorder}
}

func (p *provider) CreateBackup(ctx context.Context, backup *v1.Backup) error {
	p.recorder.Event(backup, corev1.EventTypeNormal, v1.CreateEventReason, "Use velero as backup provider")
	println("Hier könnte ihre Backup-Lösung verwendet werden")
	time.Sleep(time.Second * 60)
	println("Ende")
	return nil
}

func (p *provider) DeleteBackup(ctx context.Context, backup *v1.Backup) error {
	// TODO implement me
	panic("implement me")
}
