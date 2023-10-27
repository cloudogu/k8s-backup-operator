package backupschedule

import (
	"context"

	v1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
)

type defaultDeleteManager struct {
	clientSet ecosystemInterface
	recorder  eventRecorder
	namespace string
}

func newDeleteManager(clientSet ecosystemInterface, recorder eventRecorder, namespace string) *defaultDeleteManager {
	return &defaultDeleteManager{clientSet: clientSet, recorder: recorder, namespace: namespace}
}

func (dm *defaultDeleteManager) delete(ctx context.Context, backupSchedule *v1.BackupSchedule) error {
	//TODO implement me
	panic("implement me")
}
