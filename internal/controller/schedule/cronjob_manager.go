package schedule

import (
	"context"

	v1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var defaultLabels = map[string]string{
	"app":                          "ces",
	"k8s.cloudogu.com/part-of":     "backup",
	"app.kubernetes.io/created-by": "k8s-backup-operator",
	"app.kubernetes.io/part-of":    "k8s-backup-operator",
}

type cronJobManager struct {
	client.Client
	scheme *runtime.Scheme
}

func (c cronJobManager) Ensure(ctx context.Context, schedule *v1.BackupSchedule) error {
	return nil
}

func (c cronJobManager) Delete(ctx context.Context, schedule *v1.BackupSchedule) error {
	return nil
}
