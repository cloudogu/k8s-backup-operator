package backup

import (
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
)

type defaultProviderBackupStatus struct{}

func (d defaultProviderBackupStatus) isInProgress(backup *velerov1.Backup) bool {
	//TODO implement me
	panic("implement me")
}

func (d defaultProviderBackupStatus) isCompleted(backup *velerov1.Backup) bool {
	//TODO implement me
	panic("implement me")
}

func (d defaultProviderBackupStatus) hasFailed(backup *velerov1.Backup) bool {
	//TODO implement me
	panic("implement me")
}
