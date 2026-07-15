package backup

import (
	"testing"

	backupV1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestServiceDeleteBackup(t *testing.T) {
	t.Run("If the backup is running don't delete it", func(t *testing.T) {
		t.Skip("TODO")
	})
}

func TestServiceReconcileBackup(t *testing.T) {
	t.Run("It should activate the maintenance mode", func(t *testing.T) {
		t.Skip("TODO")
	})

	t.Run("It should set start time", func(t *testing.T) {
		t.Skip("TODO")
	})

	t.Run("It should not set start time if it is already set", func(t *testing.T) {
		t.Skip("TODO")
	})

	t.Run("It should set condition for state in progress", func(t *testing.T) {
		t.Skip("TODO")
	})

	t.Run("should create velero backup", func(t *testing.T) {
		t.Skip("TODO")
	})

}

func newTestFixture(t *testing.T) *ServiceImpl {
	fakeClient := fake.NewClientBuilder().Build()
	clockMock := NewMockClock(t)
	serviceImpl := NewService(fakeClient, clockMock)
	return serviceImpl
}

func newBackupForServiceTest() *backupV1.Backup {
	return &backupV1.Backup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ns",
			Namespace: "backup",
		},
		Spec: backupV1.BackupSpec{
			Provider: "velero",
		},
	}
}
