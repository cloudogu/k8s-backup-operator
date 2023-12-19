package velero

import (
	backupv1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
	"time"
)

func Test_defaultSyncManager_SyncBackups(t *testing.T) {
	t.Run("should fail to list backups", func(t *testing.T) {
		// given
		backupClientMock := newMockEcosystemBackupInterface(t)
		backupClientMock.EXPECT().List(testCtx, metav1.ListOptions{}).Return(nil, assert.AnError)
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
		backupClientMock.EXPECT().List(testCtx, metav1.ListOptions{}).Return(&backupv1.BackupList{}, nil)
		v1Alpha1Mock := newMockEcosystemV1Alpha1Interface(t)
		v1Alpha1Mock.EXPECT().Backups(testNamespace).Return(backupClientMock)
		ecosystemClientSetMock := newMockEcosystemClientSet(t)
		ecosystemClientSetMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Mock)

		veleroClientSetMock := newMockVeleroClientSet(t)
		v1VeleroMock := newMockVeleroInterface(t)
		veleroBackupInterfaceMock := newMockVeleroBackupInterface(t)
		veleroBackupInterfaceMock.EXPECT().List(testCtx, metav1.ListOptions{}).Return(nil, assert.AnError)
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
		backupClientMock.EXPECT().List(testCtx, metav1.ListOptions{}).Return(&backupv1.BackupList{
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
		veleroBackupInterfaceMock.EXPECT().List(testCtx, metav1.ListOptions{}).Return(&velerov1.BackupList{}, nil)
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
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
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
		backupClientMock.EXPECT().List(testCtx, metav1.ListOptions{}).Return(&backupv1.BackupList{
			Items: []backupv1.Backup{backupMock},
		}, nil)
		backupClientMock.EXPECT().RemoveFinalizer(testCtx, &backupMock, backupv1.BackupFinalizer).Return(&backupv1.Backup{}, nil)
		backupClientMock.EXPECT().Delete(testCtx, backupName, metav1.DeleteOptions{}).Return(assert.AnError)
		v1Alpha1Mock := newMockEcosystemV1Alpha1Interface(t)
		v1Alpha1Mock.EXPECT().Backups(testNamespace).Return(backupClientMock)
		ecosystemClientSetMock := newMockEcosystemClientSet(t)
		ecosystemClientSetMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Mock)

		veleroClientSetMock := newMockVeleroClientSet(t)
		v1VeleroMock := newMockVeleroInterface(t)
		veleroBackupInterfaceMock := newMockVeleroBackupInterface(t)
		veleroBackupInterfaceMock.EXPECT().List(testCtx, metav1.ListOptions{}).Return(&velerov1.BackupList{}, nil)
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
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
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
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      veleroBackupName,
				Namespace: testNamespace,
			},
			Spec:   velerov1.BackupSpec{},
			Status: velerov1.BackupStatus{StartTimestamp: &metav1.Time{}, CompletionTimestamp: &metav1.Time{}},
		}
		createBackupMock := backupv1.Backup{
			TypeMeta:   metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{Name: veleroBackupName, Labels: map[string]string{"app": "ces", "k8s.cloudogu.com/part-of": "backup"}},
			Spec: backupv1.BackupSpec{
				Provider:           backupv1.ProviderVelero,
				SyncedFromProvider: true,
			},
		}

		backupClientMock := newMockEcosystemBackupInterface(t)
		backupClientMock.EXPECT().List(testCtx, metav1.ListOptions{}).Return(&backupv1.BackupList{
			Items: []backupv1.Backup{backupMock},
		}, nil)
		backupClientMock.EXPECT().RemoveFinalizer(testCtx, &backupMock, backupv1.BackupFinalizer).Return(&backupv1.Backup{}, nil)
		backupClientMock.EXPECT().Delete(testCtx, backupName, metav1.DeleteOptions{}).Return(nil)
		backupClientMock.EXPECT().Create(testCtx, &createBackupMock, metav1.CreateOptions{}).Return(&backupv1.Backup{}, assert.AnError)
		v1Alpha1Mock := newMockEcosystemV1Alpha1Interface(t)
		v1Alpha1Mock.EXPECT().Backups(testNamespace).Return(backupClientMock)
		ecosystemClientSetMock := newMockEcosystemClientSet(t)
		ecosystemClientSetMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Mock)

		veleroClientSetMock := newMockVeleroClientSet(t)
		v1VeleroMock := newMockVeleroInterface(t)
		veleroBackupInterfaceMock := newMockVeleroBackupInterface(t)
		veleroBackupInterfaceMock.EXPECT().List(testCtx, metav1.ListOptions{}).Return(&velerov1.BackupList{Items: []velerov1.Backup{veleroBackupMock}}, nil)
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
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
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
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      veleroBackupName,
				Namespace: testNamespace,
			},
			Spec:   velerov1.BackupSpec{},
			Status: velerov1.BackupStatus{StartTimestamp: &metav1.Time{}, CompletionTimestamp: &metav1.Time{}},
		}
		createBackupMock := backupv1.Backup{
			TypeMeta:   metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{Name: veleroBackupName, Labels: map[string]string{"app": "ces", "k8s.cloudogu.com/part-of": "backup"}},
			Spec: backupv1.BackupSpec{
				Provider:           backupv1.ProviderVelero,
				SyncedFromProvider: true,
			},
		}

		backupClientMock := newMockEcosystemBackupInterface(t)
		backupClientMock.EXPECT().List(testCtx, metav1.ListOptions{}).Return(&backupv1.BackupList{
			Items: []backupv1.Backup{backupMock},
		}, nil)
		backupClientMock.EXPECT().RemoveFinalizer(testCtx, &backupMock, backupv1.BackupFinalizer).Return(&backupv1.Backup{}, nil)
		backupClientMock.EXPECT().Delete(testCtx, backupName, metav1.DeleteOptions{}).Return(nil)
		backupClientMock.EXPECT().Create(testCtx, &createBackupMock, metav1.CreateOptions{}).Return(&backupv1.Backup{}, nil)
		v1Alpha1Mock := newMockEcosystemV1Alpha1Interface(t)
		v1Alpha1Mock.EXPECT().Backups(testNamespace).Return(backupClientMock)
		ecosystemClientSetMock := newMockEcosystemClientSet(t)
		ecosystemClientSetMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Mock)

		veleroClientSetMock := newMockVeleroClientSet(t)
		v1VeleroMock := newMockVeleroInterface(t)
		veleroBackupInterfaceMock := newMockVeleroBackupInterface(t)
		veleroBackupInterfaceMock.EXPECT().List(testCtx, metav1.ListOptions{}).Return(&velerov1.BackupList{Items: []velerov1.Backup{veleroBackupMock}}, nil)
		v1VeleroMock.EXPECT().Backups(testNamespace).Return(veleroBackupInterfaceMock)
		veleroClientSetMock.EXPECT().VeleroV1().Return(v1VeleroMock)

		sut := &defaultSyncManager{namespace: testNamespace, ecosystemClientSet: ecosystemClientSetMock, veleroClientSet: veleroClientSetMock}

		// when
		err := sut.SyncBackups(testCtx)

		// then
		require.NoError(t, err)
	})
}

func Test_defaultSyncManager_SyncBackupStatus(t *testing.T) {
	t.Run("should fail to get velero backup", func(t *testing.T) {
		// given
		backupName := "test-backup"
		testBackup := &backupv1.Backup{ObjectMeta: metav1.ObjectMeta{Name: backupName}, Spec: backupv1.BackupSpec{Provider: "velero", SyncedFromProvider: true}}

		veleroBackupClientMock := newMockVeleroBackupInterface(t)
		veleroBackupClientMock.EXPECT().Get(testCtx, backupName, metav1.GetOptions{}).Return(nil, assert.AnError)
		veleroV1Mock := newMockVeleroInterface(t)
		veleroV1Mock.EXPECT().Backups(testNamespace).Return(veleroBackupClientMock)
		veleroClientMock := newMockVeleroClientSet(t)
		veleroClientMock.EXPECT().VeleroV1().Return(veleroV1Mock)

		ecosystemClientMock := newMockEcosystemClientSet(t)

		recorderMock := newMockEventRecorder(t)

		sut := &defaultSyncManager{
			veleroClientSet:    veleroClientMock,
			ecosystemClientSet: ecosystemClientMock,
			recorder:           recorderMock,
			namespace:          testNamespace,
		}

		// when
		err := sut.SyncBackupStatus(testCtx, testBackup)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to find corresponding velero backup for backup \"test-backup\"")
	})
	t.Run("should fail if phase of velero backup is not completed", func(t *testing.T) {
		// given
		backupName := "test-backup"
		testBackup := &backupv1.Backup{ObjectMeta: metav1.ObjectMeta{Name: backupName}, Spec: backupv1.BackupSpec{Provider: "velero", SyncedFromProvider: true}}

		veleroBackup := &velerov1.Backup{ObjectMeta: metav1.ObjectMeta{Name: backupName},
			Status: velerov1.BackupStatus{Phase: velerov1.BackupPhaseFailed}}

		veleroBackupClientMock := newMockVeleroBackupInterface(t)
		veleroBackupClientMock.EXPECT().Get(testCtx, backupName, metav1.GetOptions{}).Return(veleroBackup, nil)
		veleroV1Mock := newMockVeleroInterface(t)
		veleroV1Mock.EXPECT().Backups(testNamespace).Return(veleroBackupClientMock)
		veleroClientMock := newMockVeleroClientSet(t)
		veleroClientMock.EXPECT().VeleroV1().Return(veleroV1Mock)

		ecosystemClientMock := newMockEcosystemClientSet(t)

		recorderMock := newMockEventRecorder(t)

		sut := &defaultSyncManager{
			veleroClientSet:    veleroClientMock,
			ecosystemClientSet: ecosystemClientMock,
			recorder:           recorderMock,
			namespace:          testNamespace,
		}

		// when
		err := sut.SyncBackupStatus(testCtx, testBackup)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "velero backup \"test-backup\" is not completed and therefore cannot be synced")
	})
	t.Run("should fail to get updated backup", func(t *testing.T) {
		// given
		backupName := "test-backup"
		testBackup := &backupv1.Backup{ObjectMeta: metav1.ObjectMeta{Name: backupName}, Spec: backupv1.BackupSpec{Provider: "velero", SyncedFromProvider: true}}

		veleroBackup := &velerov1.Backup{
			ObjectMeta: metav1.ObjectMeta{Name: backupName},
			Status: velerov1.BackupStatus{
				Phase:               velerov1.BackupPhaseCompleted,
				StartTimestamp:      &metav1.Time{Time: time.Date(2023, time.November, 29, 13, 38, 0, 0, time.Local)},
				CompletionTimestamp: &metav1.Time{Time: time.Date(2023, time.November, 29, 13, 44, 0, 0, time.Local)},
			},
		}

		veleroBackupClientMock := newMockVeleroBackupInterface(t)
		veleroBackupClientMock.EXPECT().Get(testCtx, backupName, metav1.GetOptions{}).Return(veleroBackup, nil)
		veleroV1Mock := newMockVeleroInterface(t)
		veleroV1Mock.EXPECT().Backups(testNamespace).Return(veleroBackupClientMock)
		veleroClientMock := newMockVeleroClientSet(t)
		veleroClientMock.EXPECT().VeleroV1().Return(veleroV1Mock)

		backupClientMock := newMockEcosystemBackupInterface(t)
		backupClientMock.EXPECT().Get(testCtx, backupName, metav1.GetOptions{}).Return(nil, assert.AnError)
		v1Alpha1Mock := newMockEcosystemV1Alpha1Interface(t)
		v1Alpha1Mock.EXPECT().Backups(testNamespace).Return(backupClientMock)
		ecosystemClientMock := newMockEcosystemClientSet(t)
		ecosystemClientMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Mock)

		recorderMock := newMockEventRecorder(t)

		sut := &defaultSyncManager{
			veleroClientSet:    veleroClientMock,
			ecosystemClientSet: ecosystemClientMock,
			recorder:           recorderMock,
			namespace:          testNamespace,
		}

		// when
		err := sut.SyncBackupStatus(testCtx, testBackup)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to update status of backup \"test-backup\"")
	})
	t.Run("should fail to update backup status", func(t *testing.T) {
		// given
		backupName := "test-backup"
		testBackup := &backupv1.Backup{ObjectMeta: metav1.ObjectMeta{Name: backupName}, Spec: backupv1.BackupSpec{Provider: "velero", SyncedFromProvider: true}}

		veleroBackup := &velerov1.Backup{
			ObjectMeta: metav1.ObjectMeta{Name: backupName},
			Status: velerov1.BackupStatus{
				Phase:               velerov1.BackupPhaseCompleted,
				StartTimestamp:      &metav1.Time{Time: time.Date(2023, time.November, 29, 13, 38, 0, 0, time.Local)},
				CompletionTimestamp: &metav1.Time{Time: time.Date(2023, time.November, 29, 13, 44, 0, 0, time.Local)},
			},
		}

		veleroBackupClientMock := newMockVeleroBackupInterface(t)
		veleroBackupClientMock.EXPECT().Get(testCtx, backupName, metav1.GetOptions{}).Return(veleroBackup, nil)
		veleroV1Mock := newMockVeleroInterface(t)
		veleroV1Mock.EXPECT().Backups(testNamespace).Return(veleroBackupClientMock)
		veleroClientMock := newMockVeleroClientSet(t)
		veleroClientMock.EXPECT().VeleroV1().Return(veleroV1Mock)

		backupClientMock := newMockEcosystemBackupInterface(t)
		backupClientMock.EXPECT().Get(testCtx, backupName, metav1.GetOptions{}).Return(testBackup, nil)
		backupClientMock.EXPECT().UpdateStatus(testCtx, testBackup, metav1.UpdateOptions{}).Return(nil, assert.AnError)
		v1Alpha1Mock := newMockEcosystemV1Alpha1Interface(t)
		v1Alpha1Mock.EXPECT().Backups(testNamespace).Return(backupClientMock)
		ecosystemClientMock := newMockEcosystemClientSet(t)
		ecosystemClientMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Mock)

		recorderMock := newMockEventRecorder(t)

		sut := &defaultSyncManager{
			veleroClientSet:    veleroClientMock,
			ecosystemClientSet: ecosystemClientMock,
			recorder:           recorderMock,
			namespace:          testNamespace,
		}

		// when
		err := sut.SyncBackupStatus(testCtx, testBackup)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to update status of backup \"test-backup\"")
	})
	t.Run("should sync backup status", func(t *testing.T) {
		// given
		backupName := "test-backup"
		testBackup := &backupv1.Backup{ObjectMeta: metav1.ObjectMeta{Name: backupName}, Spec: backupv1.BackupSpec{Provider: "velero", SyncedFromProvider: true}}

		startTime := metav1.Time{Time: time.Date(2023, time.November, 29, 13, 38, 0, 0, time.Local)}
		completionTime := metav1.Time{Time: time.Date(2023, time.November, 29, 13, 44, 0, 0, time.Local)}
		veleroBackup := &velerov1.Backup{
			ObjectMeta: metav1.ObjectMeta{Name: backupName},
			Status: velerov1.BackupStatus{
				Phase:               velerov1.BackupPhaseCompleted,
				StartTimestamp:      &startTime,
				CompletionTimestamp: &completionTime,
			},
		}

		veleroBackupClientMock := newMockVeleroBackupInterface(t)
		veleroBackupClientMock.EXPECT().Get(testCtx, backupName, metav1.GetOptions{}).Return(veleroBackup, nil)
		veleroV1Mock := newMockVeleroInterface(t)
		veleroV1Mock.EXPECT().Backups(testNamespace).Return(veleroBackupClientMock)
		veleroClientMock := newMockVeleroClientSet(t)
		veleroClientMock.EXPECT().VeleroV1().Return(veleroV1Mock)

		backupClientMock := newMockEcosystemBackupInterface(t)
		backupClientMock.EXPECT().Get(testCtx, backupName, metav1.GetOptions{}).Return(testBackup, nil)
		backupClientMock.EXPECT().UpdateStatus(testCtx, testBackup, metav1.UpdateOptions{}).Return(testBackup, nil)
		v1Alpha1Mock := newMockEcosystemV1Alpha1Interface(t)
		v1Alpha1Mock.EXPECT().Backups(testNamespace).Return(backupClientMock)
		ecosystemClientMock := newMockEcosystemClientSet(t)
		ecosystemClientMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Mock)

		recorderMock := newMockEventRecorder(t)

		sut := &defaultSyncManager{
			veleroClientSet:    veleroClientMock,
			ecosystemClientSet: ecosystemClientMock,
			recorder:           recorderMock,
			namespace:          testNamespace,
		}

		// when
		err := sut.SyncBackupStatus(testCtx, testBackup)

		// then
		require.NoError(t, err)
		assert.Equal(t, startTime, testBackup.Status.StartTimestamp)
		assert.Equal(t, completionTime, testBackup.Status.CompletionTimestamp)
		assert.Equal(t, "completed", testBackup.Status.Status)
	})
}

func TestNewDefaultSyncManager(t *testing.T) {
	// given
	veleroClientMock := newMockVeleroClientSet(t)
	ecosystemClientMock := newMockEcosystemClientSet(t)
	recorderMock := newMockEventRecorder(t)

	// when
	actual := newDefaultSyncManager(veleroClientMock, ecosystemClientMock, recorderMock, testNamespace)

	// then
	assert.NotEmpty(t, actual)
}
