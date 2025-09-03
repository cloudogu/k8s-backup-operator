package scheduledbackup

import (
	"context"
	"fmt"
	"github.com/cloudogu/k8s-backup-lib/api/ecosystem"
	time2 "github.com/cloudogu/k8s-backup-operator/pkg/time"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
)

const timeFormatK8sAcceptableName = "2006-01-02t15.04.05"

type Options struct {
	Name      string
	Namespace string
	Provider  string
}

type DefaultManager struct {
	clientSet ecosystemClientSet
	options   Options
	clock     timeProvider
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

	_, err := dm.clientSet.EcosystemV1Alpha1().Backups(dm.options.Namespace).Create(ctx, scheduledBackup, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to apply backup %q: %w", backupName, err)
	}

	return nil
}

func NewManager(clientSet ecosystem.Interface, options Options) Manager {
	return &DefaultManager{clientSet: clientSet, options: options, clock: &time2.Clock{}}
}
