package scheduledbackup

import (
	"context"
	"fmt"

	operatortime "github.com/cloudogu/k8s-backup-operator/internal/time"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
)

const timeFormatK8sAcceptableName = "2006-01-02t15.04.05"

type Options struct {
	Name      string
	Namespace string
	Provider  string
}

type DefaultManager struct {
	client  k8sClient
	options Options
	clock   timeProvider
}

func (dm *DefaultManager) ScheduleBackup(ctx context.Context) error {
	backupName := fmt.Sprintf("%s-%s", dm.options.Name, dm.clock.Now().Format(timeFormatK8sAcceptableName))
	scheduledBackup := &backupv1.Backup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      backupName,
			Namespace: dm.options.Namespace,
			Labels: map[string]string{
				"app":                          "ces",
				"k8s.cloudogu.com/part-of":     "backup",
				"app.kubernetes.io/name":       "backup",
				"app.kubernetes.io/part-of":    "k8s-backup-operator",
				"app.kubernetes.io/created-by": "k8s-backup-operator",
			},
		},
		Spec: backupv1.BackupSpec{
			Provider: backupv1.Provider(dm.options.Provider),
		},
	}

	err := dm.client.Create(ctx, scheduledBackup)
	if err != nil {
		return fmt.Errorf("failed to apply backup %q: %w", backupName, err)
	}

	return nil
}

func NewManager(client client.Client, options Options) Manager {
	return &DefaultManager{client: client, options: options, clock: &operatortime.Clock{}}
}
