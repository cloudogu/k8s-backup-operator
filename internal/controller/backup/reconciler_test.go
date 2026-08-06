package backup

import (
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func newVeleroBackupForReconcilerTest(namespace string, name string, phase velerov1.BackupPhase) *velerov1.Backup {
	return &velerov1.Backup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Status: velerov1.BackupStatus{
			Phase: phase,
		},
	}
}

func newVeleroBackupStorageLocationForReconcilerTest(phase velerov1.BackupStorageLocationPhase) *velerov1.BackupStorageLocation {
	return &velerov1.BackupStorageLocation{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "default",
		},
		Status: velerov1.BackupStorageLocationStatus{
			Phase: phase,
		},
	}
}
