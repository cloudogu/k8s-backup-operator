package backup

import (
	"github.com/cloudogu/k8s-backup-operator/pkg/provider"
	"github.com/stretchr/testify/assert"
	"testing"

	"github.com/stretchr/testify/require"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/cloudogu/k8s-backup-lib/api/v1"
)

func Test_backupStatusSyncManager_syncStatus(t *testing.T) {
	t.Run("should fail to sync backup status", func(t *testing.T) {
		// given
		testBackup := &v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "test-backup"}, Spec: v1.BackupSpec{Provider: "velero", SyncedFromProvider: true}}

		providerMock := newMockBackupProvider(t)
		providerMock.EXPECT().CheckReady(testCtx).Return(nil)
		providerMock.EXPECT().SyncBackupStatus(testCtx, testBackup).Return(assert.AnError)
		oldVeleroProviderFunc := provider.NewVeleroProvider
		defer func() { provider.NewVeleroProvider = oldVeleroProviderFunc }()
		provider.NewVeleroProvider = func(client provider.K8sClient, ecosystemClientSet provider.EcosystemClientSet, recorder provider.EventRecorder, namespace string) provider.Provider {
			return providerMock
		}

		clientMock := newMockK8sClient(t)

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(testBackup, "Normal", "SyncStatus", "Syncing status of backup \"test-backup\" with its corresponding provider backup")

		sut := &backupStatusSyncManager{
			namespace: testNamespace,
			k8sClient: clientMock,
			recorder:  recorderMock,
		}

		// when
		err := sut.syncStatus(testCtx, testBackup)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to sync status of backup \"test-backup\" with its corresponding \"velero\" backup")
	})
	t.Run("should succeed", func(t *testing.T) {
		// given
		testBackup := &v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "test-backup"}, Spec: v1.BackupSpec{Provider: "velero", SyncedFromProvider: true}}

		providerMock := newMockBackupProvider(t)
		providerMock.EXPECT().CheckReady(testCtx).Return(nil)
		providerMock.EXPECT().SyncBackupStatus(testCtx, testBackup).Return(nil)
		oldVeleroProviderFunc := provider.NewVeleroProvider
		defer func() { provider.NewVeleroProvider = oldVeleroProviderFunc }()
		provider.NewVeleroProvider = func(client provider.K8sClient, ecosystemClientSet provider.EcosystemClientSet, recorder provider.EventRecorder, namespace string) provider.Provider {
			return providerMock
		}

		clientMock := newMockK8sClient(t)

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(testBackup, "Normal", "SyncStatus", "Syncing status of backup \"test-backup\" with its corresponding provider backup")
		recorderMock.EXPECT().Event(testBackup, "Normal", "SyncStatus", "Successfully synced status of backup \"test-backup\" with its corresponding provider backup")

		sut := &backupStatusSyncManager{
			namespace: testNamespace,
			k8sClient: clientMock,
			recorder:  recorderMock,
		}

		// when
		err := sut.syncStatus(testCtx, testBackup)

		// then
		require.NoError(t, err)
	})
}
