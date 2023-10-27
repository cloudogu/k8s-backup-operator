package backupschedule

import (
	"context"

	v1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
)

type defaultUpdateManager struct {
	clientSet ecosystemInterface
	recorder  eventRecorder
	namespace string
}

func newUpdateManager(clientSet ecosystemInterface, recorder eventRecorder, namespace string) *defaultUpdateManager {
	return &defaultUpdateManager{clientSet: clientSet, recorder: recorder, namespace: namespace}
}

func (um *defaultUpdateManager) update(ctx context.Context, backupSchedule *v1.BackupSchedule) error {
	//TODO implement me
	panic("implement me")
}
