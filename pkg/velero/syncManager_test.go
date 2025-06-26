package velero

import (
	"context"
	backupv1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
	"time"
)

func Test_defaultSyncManager_SyncBackups(t *testing.T) {
	t.Run("should fail to list backups", func(t *testing.T) {
		// given
		k8sClient := newMockK8sWatchClient(t)
		k8sClient.EXPECT().List(testCtx, &backupv1.BackupList{}, &client.ListOptions{Namespace: testNamespace}).Return(assert.AnError)

		sut := &defaultSyncManager{namespace: testNamespace, k8sClient: k8sClient}

		// when
		err := sut.SyncBackups(testCtx)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "could not list ecosystem backups")
	})

	t.Run("should fail to list velero backups", func(t *testing.T) {
		// given
		k8sClient := newMockK8sWatchClient(t)
		k8sClient.EXPECT().List(testCtx, &backupv1.BackupList{}, &client.ListOptions{Namespace: testNamespace}).Return(nil)
		k8sClient.EXPECT().List(testCtx, &velerov1.BackupList{}, &client.ListOptions{Namespace: testNamespace}).Return(assert.AnError)

		sut := &defaultSyncManager{namespace: testNamespace, k8sClient: k8sClient}

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

		mockK8sClient := newMockK8sWatchClient(t)
		backupList := &backupv1.BackupList{}
		mockK8sClient.EXPECT().List(testCtx, backupList, &client.ListOptions{Namespace: testNamespace}).RunAndReturn(func(ctx context.Context, list client.ObjectList, option ...client.ListOption) error {
			backupList = list.(*backupv1.BackupList)
			backupList.Items = []backupv1.Backup{
				backupMock,
			}
			return nil
		})

		mockK8sClient.EXPECT().List(testCtx, &velerov1.BackupList{}, &client.ListOptions{Namespace: testNamespace}).RunAndReturn(func(ctx context.Context, list client.ObjectList, option ...client.ListOption) error {
			return nil
		})

		mockK8sClient.EXPECT().Get(testCtx, mock.Anything, mock.Anything).Return(assert.AnError)

		sut := &defaultSyncManager{namespace: testNamespace, k8sClient: mockK8sClient}

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

		mockK8sClient := newMockK8sWatchClient(t)
		backupList := &backupv1.BackupList{}
		mockK8sClient.EXPECT().List(testCtx, backupList, &client.ListOptions{Namespace: testNamespace}).RunAndReturn(func(ctx context.Context, list client.ObjectList, option ...client.ListOption) error {
			backupList = list.(*backupv1.BackupList)
			backupList.Items = []backupv1.Backup{
				backupMock,
			}
			return nil
		})

		mockK8sClient.EXPECT().List(testCtx, &velerov1.BackupList{}, &client.ListOptions{Namespace: testNamespace}).RunAndReturn(func(ctx context.Context, list client.ObjectList, option ...client.ListOption) error {
			return nil
		})

		mockK8sClient.EXPECT().Get(testCtx, mock.Anything, mock.Anything).Return(nil)
		mockK8sClient.EXPECT().Update(testCtx, mock.Anything).Return(assert.AnError)

		sut := &defaultSyncManager{namespace: testNamespace, k8sClient: mockK8sClient}

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
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      veleroBackupName,
				Namespace: testNamespace,
				Labels:    map[string]string{"app": "ces", "k8s.cloudogu.com/part-of": "backup"},
			},
			Spec: backupv1.BackupSpec{
				Provider:           backupv1.ProviderVelero,
				SyncedFromProvider: true,
			},
		}

		mockK8sClient := newMockK8sWatchClient(t)
		backupList := &backupv1.BackupList{}
		mockK8sClient.EXPECT().List(testCtx, backupList, &client.ListOptions{Namespace: testNamespace}).RunAndReturn(func(ctx context.Context, list client.ObjectList, option ...client.ListOption) error {
			backupList = list.(*backupv1.BackupList)
			backupList.Items = []backupv1.Backup{
				backupMock,
			}
			return nil
		})

		velerobackupList := &velerov1.BackupList{}
		mockK8sClient.EXPECT().List(testCtx, velerobackupList, &client.ListOptions{Namespace: testNamespace}).RunAndReturn(func(ctx context.Context, list client.ObjectList, option ...client.ListOption) error {
			velerobackupList = list.(*velerov1.BackupList)
			velerobackupList.Items = []velerov1.Backup{
				veleroBackupMock,
			}
			return nil
		})

		mockK8sClient.EXPECT().Get(testCtx, mock.Anything, mock.Anything).Return(nil)
		mockK8sClient.EXPECT().Update(testCtx, mock.Anything).Return(nil)
		mockK8sClient.EXPECT().Delete(testCtx, mock.Anything).Return(nil)

		mockK8sClient.EXPECT().Create(testCtx, &createBackupMock).Return(assert.AnError)

		sut := &defaultSyncManager{namespace: testNamespace, k8sClient: mockK8sClient}

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
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      veleroBackupName,
				Namespace: testNamespace,
				Labels:    map[string]string{"app": "ces", "k8s.cloudogu.com/part-of": "backup"},
			},
			Spec: backupv1.BackupSpec{
				Provider:           backupv1.ProviderVelero,
				SyncedFromProvider: true,
			},
		}

		mockK8sClient := newMockK8sWatchClient(t)
		backupList := &backupv1.BackupList{}
		mockK8sClient.EXPECT().List(testCtx, backupList, &client.ListOptions{Namespace: testNamespace}).RunAndReturn(func(ctx context.Context, list client.ObjectList, option ...client.ListOption) error {
			backupList = list.(*backupv1.BackupList)
			backupList.Items = []backupv1.Backup{
				backupMock,
			}
			return nil
		})

		velerobackupList := &velerov1.BackupList{}
		mockK8sClient.EXPECT().List(testCtx, velerobackupList, &client.ListOptions{Namespace: testNamespace}).RunAndReturn(func(ctx context.Context, list client.ObjectList, option ...client.ListOption) error {
			velerobackupList = list.(*velerov1.BackupList)
			velerobackupList.Items = []velerov1.Backup{
				veleroBackupMock,
			}
			return nil
		})

		mockK8sClient.EXPECT().Get(testCtx, mock.Anything, mock.Anything).Return(nil)
		mockK8sClient.EXPECT().Update(testCtx, mock.Anything).Return(nil)
		mockK8sClient.EXPECT().Delete(testCtx, mock.Anything).Return(nil)

		mockK8sClient.EXPECT().Create(testCtx, &createBackupMock).Return(nil)

		sut := &defaultSyncManager{namespace: testNamespace, k8sClient: mockK8sClient}

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

		mockK8sClient := newMockK8sWatchClient(t)

		mockK8sClient.EXPECT().Get(testCtx, testBackup.GetNamespacedName(), &velerov1.Backup{}).Return(assert.AnError)

		recorderMock := newMockEventRecorder(t)

		sut := &defaultSyncManager{
			k8sClient: mockK8sClient,
			recorder:  recorderMock,
			namespace: testNamespace,
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

		mockK8sClient := newMockK8sWatchClient(t)

		mockK8sClient.EXPECT().Get(testCtx, testBackup.GetNamespacedName(), &velerov1.Backup{}).RunAndReturn(func(ctx context.Context, name types.NamespacedName, object client.Object, option ...client.GetOption) error {
			veleroBackupMock := object.(*velerov1.Backup)
			veleroBackupMock.ObjectMeta = metav1.ObjectMeta{Name: backupName}
			veleroBackupMock.Status = velerov1.BackupStatus{Phase: velerov1.BackupPhaseFailed}
			return nil
		},
		)

		recorderMock := newMockEventRecorder(t)

		sut := &defaultSyncManager{
			k8sClient: mockK8sClient,
			recorder:  recorderMock,
			namespace: testNamespace,
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

		mockK8sClient := newMockK8sWatchClient(t)

		veleroBackup := &velerov1.Backup{
			ObjectMeta: metav1.ObjectMeta{Name: backupName},
			Status: velerov1.BackupStatus{
				Phase:               velerov1.BackupPhaseCompleted,
				StartTimestamp:      &metav1.Time{Time: time.Date(2023, time.November, 29, 13, 38, 0, 0, time.Local)},
				CompletionTimestamp: &metav1.Time{Time: time.Date(2023, time.November, 29, 13, 44, 0, 0, time.Local)},
			},
		}

		mockK8sClient.EXPECT().Get(testCtx, testBackup.GetNamespacedName(), &velerov1.Backup{}).RunAndReturn(func(ctx context.Context, name types.NamespacedName, object client.Object, option ...client.GetOption) error {
			veleroBackupMock := object.(*velerov1.Backup)
			veleroBackupMock.ObjectMeta = veleroBackup.ObjectMeta
			veleroBackupMock.Status = veleroBackup.Status
			return nil
		},
		)

		mockK8sClient.EXPECT().Get(testCtx, testBackup.GetNamespacedName(), &backupv1.Backup{}).Return(assert.AnError)

		recorderMock := newMockEventRecorder(t)

		sut := &defaultSyncManager{
			k8sClient: mockK8sClient,
			recorder:  recorderMock,
			namespace: testNamespace,
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

		mockK8sClient := newMockK8sWatchClient(t)

		veleroBackup := &velerov1.Backup{
			ObjectMeta: metav1.ObjectMeta{Name: backupName},
			Status: velerov1.BackupStatus{
				Phase:               velerov1.BackupPhaseCompleted,
				StartTimestamp:      &metav1.Time{Time: time.Date(2023, time.November, 29, 13, 38, 0, 0, time.Local)},
				CompletionTimestamp: &metav1.Time{Time: time.Date(2023, time.November, 29, 13, 44, 0, 0, time.Local)},
			},
		}

		mockK8sClient.EXPECT().Get(testCtx, testBackup.GetNamespacedName(), &velerov1.Backup{}).RunAndReturn(func(ctx context.Context, name types.NamespacedName, object client.Object, option ...client.GetOption) error {
			veleroBackupMock := object.(*velerov1.Backup)
			veleroBackupMock.ObjectMeta = veleroBackup.ObjectMeta
			veleroBackupMock.Status = veleroBackup.Status
			return nil
		},
		)

		mockK8sClient.EXPECT().Get(testCtx, testBackup.GetNamespacedName(), &backupv1.Backup{}).RunAndReturn(func(ctx context.Context, name types.NamespacedName, object client.Object, option ...client.GetOption) error {
			backupMock := object.(*backupv1.Backup)
			backupMock.ObjectMeta = testBackup.ObjectMeta
			backupMock.Spec = testBackup.Spec
			return nil
		})

		mockSubResourceWriter := newMockSubResourceWriter(nil, assert.AnError)
		mockK8sClient.EXPECT().Status().Return(mockSubResourceWriter)

		recorderMock := newMockEventRecorder(t)

		sut := &defaultSyncManager{
			k8sClient: mockK8sClient,
			recorder:  recorderMock,
			namespace: testNamespace,
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

		mockK8sClient := newMockK8sWatchClient(t)

		mockK8sClient.EXPECT().Get(testCtx, testBackup.GetNamespacedName(), &velerov1.Backup{}).RunAndReturn(func(ctx context.Context, name types.NamespacedName, object client.Object, option ...client.GetOption) error {
			veleroBackupMock := object.(*velerov1.Backup)
			veleroBackupMock.ObjectMeta = veleroBackup.ObjectMeta
			veleroBackupMock.Status = veleroBackup.Status
			return nil
		},
		)

		mockK8sClient.EXPECT().Get(testCtx, testBackup.GetNamespacedName(), &backupv1.Backup{}).RunAndReturn(func(ctx context.Context, name types.NamespacedName, object client.Object, option ...client.GetOption) error {
			backupMock := object.(*backupv1.Backup)
			backupMock.ObjectMeta = testBackup.ObjectMeta
			backupMock.Spec = testBackup.Spec
			return nil
		})

		mockSubResourceWriter := newMockSubResourceWriter(testBackup, nil)
		mockK8sClient.EXPECT().Status().Return(mockSubResourceWriter)

		recorderMock := newMockEventRecorder(t)

		sut := &defaultSyncManager{
			k8sClient: mockK8sClient,
			recorder:  recorderMock,
			namespace: testNamespace,
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
	mockK8sClient := newMockK8sWatchClient(t)
	recorderMock := newMockEventRecorder(t)

	// when
	actual := newDefaultSyncManager(mockK8sClient, recorderMock, testNamespace)

	// then
	assert.NotEmpty(t, actual)
}

type mockSubResourceWriter struct {
	reterr error
	backup *backupv1.Backup
}

func newMockSubResourceWriter(b *backupv1.Backup, err error) mockSubResourceWriter {
	return mockSubResourceWriter{backup: b, reterr: err}
}

func (srw mockSubResourceWriter) Create(ctx context.Context, obj client.Object, subResource client.Object, opts ...client.SubResourceCreateOption) error {
	return srw.reterr
}

func (srw mockSubResourceWriter) Update(ctx context.Context, obj client.Object, opts ...client.SubResourceUpdateOption) error {
	if srw.backup != nil {
		backupObj := obj.(*backupv1.Backup)
		srw.backup.Status = backupObj.Status
	}
	return srw.reterr
}

func (srw mockSubResourceWriter) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.SubResourcePatchOption) error {
	return srw.reterr
}
