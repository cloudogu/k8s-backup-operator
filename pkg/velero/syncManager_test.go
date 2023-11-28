package velero

import (
	backupv1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func Test_defaultSyncManager_SyncBackups(t *testing.T) {
	t.Run("should fail to list backups", func(t *testing.T) {
		// given
		backupClientMock := newMockEcosystemBackupInterface(t)
		backupClientMock.EXPECT().List(testCtx, v1.ListOptions{}).Return(nil, assert.AnError)
		v1Alpha1Mock := newMockEcosystemV1Alpha1Interface(t)
		v1Alpha1Mock.EXPECT().Backups(testNamespace).Return(backupClientMock)
		ecosystemClientSetMock := newMockEcosystemClientSet(t)
		ecosystemClientSetMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Mock)

		sut := &defaultSyncManager{namespace: testNamespace, ecosystemClientSet: ecosystemClientSetMock}

		// when
		err := sut.SyncBackups(testCtx)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "could not list ecosystem backups")
	})

	t.Run("should fail to list velero backups", func(t *testing.T) {
		// given
		backupClientMock := newMockEcosystemBackupInterface(t)
		backupClientMock.EXPECT().List(testCtx, v1.ListOptions{}).Return(&backupv1.BackupList{}, nil)
		v1Alpha1Mock := newMockEcosystemV1Alpha1Interface(t)
		v1Alpha1Mock.EXPECT().Backups(testNamespace).Return(backupClientMock)
		ecosystemClientSetMock := newMockEcosystemClientSet(t)
		ecosystemClientSetMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Mock)

		veleroClientSetMock := newMockVeleroClientSet(t)
		v1VeleroMock := newMockVeleroInterface(t)
		veleroBackupInterfaceMock := newMockVeleroBackupInterface(t)
		veleroBackupInterfaceMock.EXPECT().List(testCtx, v1.ListOptions{}).Return(nil, assert.AnError)
		v1VeleroMock.EXPECT().Backups(testNamespace).Return(veleroBackupInterfaceMock)
		veleroClientSetMock.EXPECT().VeleroV1().Return(v1VeleroMock)

		sut := &defaultSyncManager{namespace: testNamespace, ecosystemClientSet: ecosystemClientSetMock, veleroClientSet: veleroClientSetMock}

		// when
		err := sut.SyncBackups(testCtx)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "could not list velero backups")
	})

	t.Run("should fail to remove finalizer", func(t *testing.T) {
		// given
		backupMock := backupv1.Backup{
			Spec: backupv1.BackupSpec{
				Provider: "velero",
			},
			Status: backupv1.BackupStatus{
				Status: "completed",
			},
		}
		backupClientMock := newMockEcosystemBackupInterface(t)
		backupClientMock.EXPECT().List(testCtx, v1.ListOptions{}).Return(&backupv1.BackupList{
			Items: []backupv1.Backup{backupMock},
		}, nil)
		backupClientMock.EXPECT().RemoveFinalizer(testCtx, &backupMock, backupv1.BackupFinalizer).Return(nil, assert.AnError)
		v1Alpha1Mock := newMockEcosystemV1Alpha1Interface(t)
		v1Alpha1Mock.EXPECT().Backups(testNamespace).Return(backupClientMock)
		ecosystemClientSetMock := newMockEcosystemClientSet(t)
		ecosystemClientSetMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Mock)

		veleroClientSetMock := newMockVeleroClientSet(t)
		v1VeleroMock := newMockVeleroInterface(t)
		veleroBackupInterfaceMock := newMockVeleroBackupInterface(t)
		veleroBackupInterfaceMock.EXPECT().List(testCtx, v1.ListOptions{}).Return(&velerov1.BackupList{}, nil)
		v1VeleroMock.EXPECT().Backups(testNamespace).Return(veleroBackupInterfaceMock)
		veleroClientSetMock.EXPECT().VeleroV1().Return(v1VeleroMock)

		sut := &defaultSyncManager{namespace: testNamespace, ecosystemClientSet: ecosystemClientSetMock, veleroClientSet: veleroClientSetMock}

		// when
		err := sut.SyncBackups(testCtx)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to sync backups with velero")
	})

	t.Run("should fail to delete backup", func(t *testing.T) {
		// given
		backupName := "testBackup"
		backupMock := backupv1.Backup{
			TypeMeta: v1.TypeMeta{},
			ObjectMeta: v1.ObjectMeta{
				Name:      backupName,
				Namespace: testNamespace,
			},
			Spec: backupv1.BackupSpec{
				Provider: "velero",
			},
			Status: backupv1.BackupStatus{
				Status: "completed",
			},
		}

		backupClientMock := newMockEcosystemBackupInterface(t)
		backupClientMock.EXPECT().List(testCtx, v1.ListOptions{}).Return(&backupv1.BackupList{
			Items: []backupv1.Backup{backupMock},
		}, nil)
		backupClientMock.EXPECT().RemoveFinalizer(testCtx, &backupMock, backupv1.BackupFinalizer).Return(&backupv1.Backup{}, nil)
		backupClientMock.EXPECT().Delete(testCtx, backupName, v1.DeleteOptions{}).Return(assert.AnError)
		v1Alpha1Mock := newMockEcosystemV1Alpha1Interface(t)
		v1Alpha1Mock.EXPECT().Backups(testNamespace).Return(backupClientMock)
		ecosystemClientSetMock := newMockEcosystemClientSet(t)
		ecosystemClientSetMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Mock)

		veleroClientSetMock := newMockVeleroClientSet(t)
		v1VeleroMock := newMockVeleroInterface(t)
		veleroBackupInterfaceMock := newMockVeleroBackupInterface(t)
		veleroBackupInterfaceMock.EXPECT().List(testCtx, v1.ListOptions{}).Return(&velerov1.BackupList{}, nil)
		v1VeleroMock.EXPECT().Backups(testNamespace).Return(veleroBackupInterfaceMock)
		veleroClientSetMock.EXPECT().VeleroV1().Return(v1VeleroMock)

		sut := &defaultSyncManager{namespace: testNamespace, ecosystemClientSet: ecosystemClientSetMock, veleroClientSet: veleroClientSetMock}

		// when
		err := sut.SyncBackups(testCtx)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to sync backups with velero")
	})

	t.Run("should fail to create backup CR", func(t *testing.T) {
		// given
		backupName := "testBackup"
		backupMock := backupv1.Backup{
			TypeMeta: v1.TypeMeta{},
			ObjectMeta: v1.ObjectMeta{
				Name:      backupName,
				Namespace: testNamespace,
			},
			Spec: backupv1.BackupSpec{
				Provider: backupv1.ProviderVelero,
			},
			Status: backupv1.BackupStatus{
				Status: "completed",
			},
		}
		veleroBackupName := "veleroTestBackup"
		veleroBackupMock := velerov1.Backup{
			TypeMeta: v1.TypeMeta{},
			ObjectMeta: v1.ObjectMeta{
				Name:      veleroBackupName,
				Namespace: testNamespace,
			},
			Spec:   velerov1.BackupSpec{},
			Status: velerov1.BackupStatus{StartTimestamp: &v1.Time{}, CompletionTimestamp: &v1.Time{}},
		}
		createBackupMock := backupv1.Backup{
			TypeMeta:   v1.TypeMeta{},
			ObjectMeta: v1.ObjectMeta{},
			Spec: backupv1.BackupSpec{
				Provider: backupv1.ProviderVelero,
			},
			Status: backupv1.BackupStatus{Status: "completed"},
		}

		backupClientMock := newMockEcosystemBackupInterface(t)
		backupClientMock.EXPECT().List(testCtx, v1.ListOptions{}).Return(&backupv1.BackupList{
			Items: []backupv1.Backup{backupMock},
		}, nil)
		backupClientMock.EXPECT().RemoveFinalizer(testCtx, &backupMock, backupv1.BackupFinalizer).Return(&backupv1.Backup{}, nil)
		backupClientMock.EXPECT().Delete(testCtx, backupName, v1.DeleteOptions{}).Return(nil)
		backupClientMock.EXPECT().Create(testCtx, &createBackupMock, v1.CreateOptions{}).Return(&backupv1.Backup{}, assert.AnError)
		v1Alpha1Mock := newMockEcosystemV1Alpha1Interface(t)
		v1Alpha1Mock.EXPECT().Backups(testNamespace).Return(backupClientMock)
		ecosystemClientSetMock := newMockEcosystemClientSet(t)
		ecosystemClientSetMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Mock)

		veleroClientSetMock := newMockVeleroClientSet(t)
		v1VeleroMock := newMockVeleroInterface(t)
		veleroBackupInterfaceMock := newMockVeleroBackupInterface(t)
		veleroBackupInterfaceMock.EXPECT().List(testCtx, v1.ListOptions{}).Return(&velerov1.BackupList{Items: []velerov1.Backup{veleroBackupMock}}, nil)
		v1VeleroMock.EXPECT().Backups(testNamespace).Return(veleroBackupInterfaceMock)
		veleroClientSetMock.EXPECT().VeleroV1().Return(v1VeleroMock)

		sut := &defaultSyncManager{namespace: testNamespace, ecosystemClientSet: ecosystemClientSetMock, veleroClientSet: veleroClientSetMock}

		// when
		err := sut.SyncBackups(testCtx)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to sync backups with velero")
	})

	t.Run("should succeed syncing", func(t *testing.T) {
		// given
		backupName := "testBackup"
		backupMock := backupv1.Backup{
			TypeMeta: v1.TypeMeta{},
			ObjectMeta: v1.ObjectMeta{
				Name:      backupName,
				Namespace: testNamespace,
			},
			Spec: backupv1.BackupSpec{
				Provider: backupv1.ProviderVelero,
			},
			Status: backupv1.BackupStatus{
				Status: "completed",
			},
		}
		veleroBackupName := "veleroTestBackup"
		veleroBackupMock := velerov1.Backup{
			TypeMeta: v1.TypeMeta{},
			ObjectMeta: v1.ObjectMeta{
				Name:      veleroBackupName,
				Namespace: testNamespace,
			},
			Spec:   velerov1.BackupSpec{},
			Status: velerov1.BackupStatus{StartTimestamp: &v1.Time{}, CompletionTimestamp: &v1.Time{}},
		}
		createBackupMock := backupv1.Backup{
			TypeMeta:   v1.TypeMeta{},
			ObjectMeta: v1.ObjectMeta{},
			Spec: backupv1.BackupSpec{
				Provider: backupv1.ProviderVelero,
			},
			Status: backupv1.BackupStatus{Status: "completed"},
		}

		backupClientMock := newMockEcosystemBackupInterface(t)
		backupClientMock.EXPECT().List(testCtx, v1.ListOptions{}).Return(&backupv1.BackupList{
			Items: []backupv1.Backup{backupMock},
		}, nil)
		backupClientMock.EXPECT().RemoveFinalizer(testCtx, &backupMock, backupv1.BackupFinalizer).Return(&backupv1.Backup{}, nil)
		backupClientMock.EXPECT().Delete(testCtx, backupName, v1.DeleteOptions{}).Return(nil)
		backupClientMock.EXPECT().Create(testCtx, &createBackupMock, v1.CreateOptions{}).Return(&backupv1.Backup{}, nil)
		v1Alpha1Mock := newMockEcosystemV1Alpha1Interface(t)
		v1Alpha1Mock.EXPECT().Backups(testNamespace).Return(backupClientMock)
		ecosystemClientSetMock := newMockEcosystemClientSet(t)
		ecosystemClientSetMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Mock)

		veleroClientSetMock := newMockVeleroClientSet(t)
		v1VeleroMock := newMockVeleroInterface(t)
		veleroBackupInterfaceMock := newMockVeleroBackupInterface(t)
		veleroBackupInterfaceMock.EXPECT().List(testCtx, v1.ListOptions{}).Return(&velerov1.BackupList{Items: []velerov1.Backup{veleroBackupMock}}, nil)
		v1VeleroMock.EXPECT().Backups(testNamespace).Return(veleroBackupInterfaceMock)
		veleroClientSetMock.EXPECT().VeleroV1().Return(v1VeleroMock)

		sut := &defaultSyncManager{namespace: testNamespace, ecosystemClientSet: ecosystemClientSetMock, veleroClientSet: veleroClientSetMock}

		// when
		err := sut.SyncBackups(testCtx)

		// then
		require.NoError(t, err)
	})
}
