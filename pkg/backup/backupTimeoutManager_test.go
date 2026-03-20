package backup

import (
	"testing"

	v1 "github.com/cloudogu/k8s-backup-lib/api/v1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewBackupTimeoutManager(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		clientSetMock := newMockEcosystemInterface(t)
		clientMock := newMockK8sClient(t)
		recorderMock := newMockEventRecorder(t)

		// when
		manager := newBackupTimeoutManager(clientMock, clientSetMock, testNamespace, recorderMock, 30)

		// then
		require.NotNil(t, manager)
		assert.Equal(t, testNamespace, manager.namespace)
		assert.Equal(t, 30, manager.backupRetryTimeLimit)
	})
}

func Test_backupTimeoutManager_timeout(t *testing.T) {
	t.Run("should set backup status to failed and return error", func(t *testing.T) {
		// given
		backupName := "test-backup"
		backup := &v1.Backup{
			ObjectMeta: metav1.ObjectMeta{Name: backupName, Namespace: testNamespace},
			Spec:       v1.BackupSpec{Provider: "velero"},
		}

		updatedBackup := &v1.Backup{
			ObjectMeta: metav1.ObjectMeta{Name: backupName, Namespace: testNamespace},
			Spec:       v1.BackupSpec{Provider: "velero"},
			Status:     v1.BackupStatus{Status: v1.BackupStatusFailed},
		}

		clientMock := newMockEcosystemBackupInterface(t)
		clientMock.EXPECT().UpdateStatusFailed(testCtx, backup).Return(updatedBackup, nil)

		v1Alpha1Client := newMockBackupV1Alpha1Interface(t)
		v1Alpha1Client.EXPECT().Backups(testNamespace).Return(clientMock)

		clientSetMock := newMockEcosystemInterface(t)
		clientSetMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Client)

		k8sClientMock := newMockK8sClient(t)
		recorderMock := newMockEventRecorder(t)

		sut := &backupTimeoutManager{
			k8sClient:            k8sClientMock,
			clientSet:            clientSetMock,
			namespace:            testNamespace,
			recorder:             recorderMock,
			backupRetryTimeLimit: 30,
		}

		// when
		err := sut.timeout(testCtx, backup)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "backup retry time limit (30 minutes) exceeded")
		assert.Equal(t, v1.BackupStatusFailed, backup.Status.Status)
	})

	t.Run("should return error on UpdateStatusFailed error", func(t *testing.T) {
		// given
		backupName := "test-backup"
		backup := &v1.Backup{
			ObjectMeta: metav1.ObjectMeta{Name: backupName, Namespace: testNamespace},
			Spec:       v1.BackupSpec{Provider: "velero"},
		}

		clientMock := newMockEcosystemBackupInterface(t)
		clientMock.EXPECT().UpdateStatusFailed(testCtx, backup).Return(nil, assert.AnError)

		v1Alpha1Client := newMockBackupV1Alpha1Interface(t)
		v1Alpha1Client.EXPECT().Backups(testNamespace).Return(clientMock)

		clientSetMock := newMockEcosystemInterface(t)
		clientSetMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Client)

		k8sClientMock := newMockK8sClient(t)
		recorderMock := newMockEventRecorder(t)

		sut := &backupTimeoutManager{
			k8sClient:            k8sClientMock,
			clientSet:            clientSetMock,
			namespace:            testNamespace,
			recorder:             recorderMock,
			backupRetryTimeLimit: 30,
		}

		// when
		err := sut.timeout(testCtx, backup)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to update backups status to 'Failed'")
	})

	t.Run("should handle nil updatedBackup gracefully", func(t *testing.T) {
		// given
		backupName := "test-backup"
		backup := &v1.Backup{
			ObjectMeta: metav1.ObjectMeta{Name: backupName, Namespace: testNamespace},
			Spec:       v1.BackupSpec{Provider: "velero"},
		}

		clientMock := newMockEcosystemBackupInterface(t)
		clientMock.EXPECT().UpdateStatusFailed(testCtx, backup).Return(nil, nil)

		v1Alpha1Client := newMockBackupV1Alpha1Interface(t)
		v1Alpha1Client.EXPECT().Backups(testNamespace).Return(clientMock)

		clientSetMock := newMockEcosystemInterface(t)
		clientSetMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Client)

		k8sClientMock := newMockK8sClient(t)
		recorderMock := newMockEventRecorder(t)

		sut := &backupTimeoutManager{
			k8sClient:            k8sClientMock,
			clientSet:            clientSetMock,
			namespace:            testNamespace,
			recorder:             recorderMock,
			backupRetryTimeLimit: 45,
		}

		// when
		err := sut.timeout(testCtx, backup)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "backup retry time limit (45 minutes) exceeded")
	})

	t.Run("should update backup with returned backup object", func(t *testing.T) {
		// given
		backupName := "test-backup"
		backup := &v1.Backup{
			ObjectMeta: metav1.ObjectMeta{Name: backupName, Namespace: testNamespace},
			Spec:       v1.BackupSpec{Provider: "velero"},
			Status:     v1.BackupStatus{Status: v1.BackupStatusNew},
		}

		updatedBackup := &v1.Backup{
			ObjectMeta: metav1.ObjectMeta{
				Name:      backupName,
				Namespace: testNamespace,
				Labels:    map[string]string{"updated": "true"},
			},
			Spec:   v1.BackupSpec{Provider: "velero"},
			Status: v1.BackupStatus{Status: v1.BackupStatusFailed},
		}

		clientMock := newMockEcosystemBackupInterface(t)
		clientMock.EXPECT().UpdateStatusFailed(testCtx, backup).Return(updatedBackup, nil)

		v1Alpha1Client := newMockBackupV1Alpha1Interface(t)
		v1Alpha1Client.EXPECT().Backups(testNamespace).Return(clientMock)

		clientSetMock := newMockEcosystemInterface(t)
		clientSetMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Client)

		k8sClientMock := newMockK8sClient(t)
		recorderMock := newMockEventRecorder(t)

		sut := &backupTimeoutManager{
			k8sClient:            k8sClientMock,
			clientSet:            clientSetMock,
			namespace:            testNamespace,
			recorder:             recorderMock,
			backupRetryTimeLimit: 60,
		}

		// when
		err := sut.timeout(testCtx, backup)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "backup retry time limit (60 minutes) exceeded")
		assert.Equal(t, v1.BackupStatusFailed, backup.Status.Status)
		assert.Equal(t, "true", backup.Labels["updated"])
	})
}
