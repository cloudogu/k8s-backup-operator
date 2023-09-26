package velero

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"

	backupv1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
)

var testCtx = context.TODO()

const testNamespace = "test-namespace"

func Test_provider_CreateBackup(t *testing.T) {
	t.Run("should fail to create velero backup", func(t *testing.T) {
		// given
		testBackup := &backupv1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "testBackup", Namespace: testNamespace}}

		mockRecorder := newMockEventRecorder(t)
		mockRecorder.EXPECT().Event(testBackup, "Normal", "Creation", "Using velero as backup provider")

		volumeFsBackup := false
		expectedVeleroBackup := &velerov1.Backup{
			ObjectMeta: metav1.ObjectMeta{Name: "testBackup", Namespace: testNamespace},
			Spec: velerov1.BackupSpec{
				ExcludedNamespaces:       []string{testNamespace},
				StorageLocation:          "default",
				DefaultVolumesToFsBackup: &volumeFsBackup,
			},
		}
		mockBackupInterface := newMockVeleroBackupInterface(t)
		mockBackupInterface.EXPECT().Create(testCtx, expectedVeleroBackup, metav1.CreateOptions{}).Return(nil, assert.AnError)
		mockVeleroInterface := newMockVeleroInterface(t)
		mockVeleroInterface.EXPECT().Backups(testNamespace).Return(mockBackupInterface)
		mockVeleroClient := newMockVeleroClientSet(t)
		mockVeleroClient.EXPECT().VeleroV1().Return(mockVeleroInterface)

		sut := &provider{
			recorder:        mockRecorder,
			veleroClientSet: mockVeleroClient,
		}

		// when
		err := sut.CreateBackup(testCtx, testBackup)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to apply velero backup 'test-namespace/testBackup' to cluster")
	})
	t.Run("should succeed to create velero backup", func(t *testing.T) {
		// given
		testBackup := &backupv1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "testBackup", Namespace: testNamespace}}

		mockRecorder := newMockEventRecorder(t)
		mockRecorder.EXPECT().Event(testBackup, "Normal", "Creation", "Using velero as backup provider")

		volumeFsBackup := false
		expectedVeleroBackup := &velerov1.Backup{
			ObjectMeta: metav1.ObjectMeta{Name: "testBackup", Namespace: testNamespace},
			Spec: velerov1.BackupSpec{
				ExcludedNamespaces:       []string{testNamespace},
				StorageLocation:          "default",
				DefaultVolumesToFsBackup: &volumeFsBackup,
			},
		}
		mockBackupInterface := newMockVeleroBackupInterface(t)
		mockBackupInterface.EXPECT().Create(testCtx, expectedVeleroBackup, metav1.CreateOptions{}).Return(expectedVeleroBackup, nil)
		mockVeleroInterface := newMockVeleroInterface(t)
		mockVeleroInterface.EXPECT().Backups(testNamespace).Return(mockBackupInterface)
		mockVeleroClient := newMockVeleroClientSet(t)
		mockVeleroClient.EXPECT().VeleroV1().Return(mockVeleroInterface)

		sut := &provider{
			recorder:        mockRecorder,
			veleroClientSet: mockVeleroClient,
		}

		// when
		err := sut.CreateBackup(testCtx, testBackup)

		// then
		require.NoError(t, err)
	})
}
