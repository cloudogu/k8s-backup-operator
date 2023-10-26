package requeue

import (
	"context"
	k8sv1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var testCtx = context.Background()

const testNamespace = "test"

func Test_backupRequeueHandler_Handle(t *testing.T) {
	t.Run("should exit early if there is no error", func(t *testing.T) {
		// given
		sut := &defaultRequeueHandler{}
		var originalErr error = nil
		backup := &k8sv1.Backup{}

		// when
		actual, err := sut.Handle(testCtx, "", backup, originalErr, "installing")

		// then
		require.NoError(t, err)
		assert.Equal(t, reconcile.Result{Requeue: false, RequeueAfter: 0}, actual)
	})
	t.Run("should exit early if error is not requeuable", func(t *testing.T) {
		// given
		sut := &defaultRequeueHandler{}
		var originalErr = assert.AnError
		backup := &k8sv1.Backup{}

		// when
		actual, err := sut.Handle(testCtx, "", backup, originalErr, "installing")

		// then
		require.NoError(t, err)
		assert.Equal(t, reconcile.Result{Requeue: false, RequeueAfter: 0}, actual)
	})
	t.Run("should fail to update backup status", func(t *testing.T) {
		// given
		backup := createBackup("ecosystem-backup-1", "ecosystem")

		backupInterfaceMock := newMockEcosystemBackupInterface(t)
		backupInterfaceMock.EXPECT().Get(testCtx, backup.Name, mock.Anything).Return(backup, nil)
		backupInterfaceMock.EXPECT().UpdateStatus(testCtx, backup, metav1.UpdateOptions{}).Return(nil, assert.AnError)
		backupClientGetterMock := newMockBackupV1Alpha1Interface(t)
		backupClientGetterMock.EXPECT().Backups(testNamespace).Return(backupInterfaceMock)
		clientSetMock := newMockEcosystemInterface(t)
		clientSetMock.EXPECT().EcosystemV1Alpha1().Return(backupClientGetterMock)

		sut := &defaultRequeueHandler{namespace: testNamespace, clientSet: clientSetMock}

		requeueErrMock := newMockRequeuableError(t)
		requeueErrMock.EXPECT().GetRequeueTime(mock.Anything).Return(30 * time.Second)

		// when
		actual, err := sut.Handle(testCtx, "", backup, requeueErrMock, "upgrading")

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to update status from requeue object ecosystem-backup-1 (type: *v1.Backup) while requeueing")

		assert.Equal(t, reconcile.Result{Requeue: false, RequeueAfter: 0}, actual)
	})
	t.Run("should succeed", func(t *testing.T) {
		// given
		backup := createBackup("ecosystem-backup-1", "ecosystem")

		backupInterfaceMock := newMockEcosystemBackupInterface(t)
		backupInterfaceMock.EXPECT().Get(testCtx, backup.Name, mock.Anything).Return(backup, nil)
		backupInterfaceMock.EXPECT().UpdateStatus(testCtx, backup, metav1.UpdateOptions{}).Return(backup, nil)
		backupClientGetterMock := newMockBackupV1Alpha1Interface(t)
		backupClientGetterMock.EXPECT().Backups(testNamespace).Return(backupInterfaceMock)
		clientSetMock := newMockEcosystemInterface(t)
		clientSetMock.EXPECT().EcosystemV1Alpha1().Return(backupClientGetterMock)

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Eventf(backup, "Normal", "Requeue", "Falling back to backup status %s: Trying again in %s.", "upgrading", "1s")

		sut := &defaultRequeueHandler{namespace: testNamespace, clientSet: clientSetMock, recorder: recorderMock}

		requeueErrMock := newMockRequeuableError(t)
		requeueErrMock.EXPECT().GetRequeueTime(mock.Anything).Return(time.Second)
		requeueErrMock.EXPECT().Error().Return("my error")

		// when
		actual, err := sut.Handle(testCtx, "", backup, requeueErrMock, "upgrading")

		// then
		require.NoError(t, err)

		assert.Equal(t, reconcile.Result{Requeue: true, RequeueAfter: 1000000000}, actual)
	})

	t.Run("should return error on get restore error", func(t *testing.T) {
		// given
		backup := createBackup("ecosystem-restore-1", "ecosystem")

		backupInterfaceMock := newMockEcosystemBackupInterface(t)
		backupInterfaceMock.EXPECT().Get(testCtx, backup.Name, mock.Anything).Return(backup, assert.AnError)
		backupClientGetterMock := newMockBackupV1Alpha1Interface(t)
		backupClientGetterMock.EXPECT().Backups(testNamespace).Return(backupInterfaceMock)
		clientSetMock := newMockEcosystemInterface(t)
		clientSetMock.EXPECT().EcosystemV1Alpha1().Return(backupClientGetterMock)

		sut := &defaultRequeueHandler{namespace: testNamespace, clientSet: clientSetMock}

		requeueErrMock := newMockRequeuableError(t)
		requeueErrMock.EXPECT().GetRequeueTime(mock.Anything).Return(time.Second)

		// when
		actual, err := sut.Handle(testCtx, "", backup, requeueErrMock, "upgrading")

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to update status from requeue object ecosystem-restore-1 (type: *v1.Backup) while requeueing")
		assert.Equal(t, ctrl.Result{}, actual)
	})

	t.Run("should succeed with restore type", func(t *testing.T) {
		// given
		restore := createRestore("ecosystem-restore-1", "ecosystem")

		restoreInterfaceMock := newMockEcosystemRestoreInterface(t)
		restoreInterfaceMock.EXPECT().Get(testCtx, restore.Name, mock.Anything).Return(restore, nil)
		restoreInterfaceMock.EXPECT().UpdateStatus(testCtx, restore, metav1.UpdateOptions{}).Return(restore, nil)
		restoreClientGetterMock := newMockBackupV1Alpha1Interface(t)
		restoreClientGetterMock.EXPECT().Restores(testNamespace).Return(restoreInterfaceMock)
		clientSetMock := newMockEcosystemInterface(t)
		clientSetMock.EXPECT().EcosystemV1Alpha1().Return(restoreClientGetterMock)

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Eventf(restore, "Normal", "Requeue", "Falling back to backup status %s: Trying again in %s.", "upgrading", "1s")

		sut := &defaultRequeueHandler{namespace: testNamespace, clientSet: clientSetMock, recorder: recorderMock}

		requeueErrMock := newMockRequeuableError(t)
		requeueErrMock.EXPECT().GetRequeueTime(mock.Anything).Return(time.Second)
		requeueErrMock.EXPECT().Error().Return("my error")

		// when
		actual, err := sut.Handle(testCtx, "", restore, requeueErrMock, "upgrading")

		// then
		require.NoError(t, err)

		assert.Equal(t, reconcile.Result{Requeue: true, RequeueAfter: 1000000000}, actual)
	})

	t.Run("should return error on get restore error", func(t *testing.T) {
		// given
		restore := createRestore("ecosystem-restore-1", "ecosystem")

		restoreInterfaceMock := newMockEcosystemRestoreInterface(t)
		restoreInterfaceMock.EXPECT().Get(testCtx, restore.Name, mock.Anything).Return(restore, assert.AnError)
		restoreClientGetterMock := newMockBackupV1Alpha1Interface(t)
		restoreClientGetterMock.EXPECT().Restores(testNamespace).Return(restoreInterfaceMock)
		clientSetMock := newMockEcosystemInterface(t)
		clientSetMock.EXPECT().EcosystemV1Alpha1().Return(restoreClientGetterMock)

		sut := &defaultRequeueHandler{namespace: testNamespace, clientSet: clientSetMock}

		requeueErrMock := newMockRequeuableError(t)
		requeueErrMock.EXPECT().GetRequeueTime(mock.Anything).Return(time.Second)

		// when
		actual, err := sut.Handle(testCtx, "", restore, requeueErrMock, "upgrading")

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to update status from requeue object ecosystem-restore-1 (type: *v1.Restore) while requeueing")
		assert.Equal(t, ctrl.Result{}, actual)
	})

	t.Run("should return error on wrong object type", func(t *testing.T) {
		// given
		clientSetMock := newMockEcosystemInterface(t)
		sut := &defaultRequeueHandler{namespace: testNamespace, clientSet: clientSetMock}
		requeueErrMock := newMockRequeuableError(t)
		requeueErrMock.EXPECT().GetRequeueTime(mock.Anything).Return(time.Second)

		// when
		_, err := sut.Handle(testCtx, "", &wrongObject{}, requeueErrMock, "upgrading")

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "wrong requeueable object type *requeue.wrongObject")

	})
}

type wrongObject struct {
	metav1.ObjectMeta
	metav1.TypeMeta
}

func (w wrongObject) DeepCopyObject() runtime.Object {
	return nil
}

func (w wrongObject) GetStatus() k8sv1.RequeueableStatus {
	return wrongRequeueableStatus{}
}

type wrongRequeueableStatus struct {
}

func (w wrongRequeueableStatus) GetRequeueTimeNanos() time.Duration {
	return time.Second * 2
}

func (w wrongRequeueableStatus) GetStatus() string {
	return ""
}

func createRestore(name, namespace string) *k8sv1.Restore {
	return &k8sv1.Restore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: k8sv1.RestoreSpec{},
	}
}

func createBackup(name, namespace string) *k8sv1.Backup {
	return &k8sv1.Backup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: k8sv1.BackupSpec{},
	}
}

func Test_backupRequeueHandler_noLongerHandleRequeueing(t *testing.T) {
	t.Run("reset requeue time to avoid further requeueing", func(t *testing.T) {
		// given
		finishedBackup := &k8sv1.Backup{Status: k8sv1.BackupStatus{
			Status:           k8sv1.BackupStatusCompleted,
			RequeueTimeNanos: 3000}}

		backupInterfaceMock := newMockEcosystemBackupInterface(t)
		backupInterfaceMock.EXPECT().Get(testCtx, finishedBackup.Name, mock.Anything).Return(finishedBackup, nil)
		backupInterfaceMock.EXPECT().UpdateStatus(testCtx, finishedBackup, metav1.UpdateOptions{}).Return(finishedBackup, nil)
		backupClientGetterMock := newMockBackupV1Alpha1Interface(t)
		backupClientGetterMock.EXPECT().Backups(testNamespace).Return(backupInterfaceMock)
		clientSetMock := newMockEcosystemInterface(t)
		clientSetMock.EXPECT().EcosystemV1Alpha1().Return(backupClientGetterMock)

		sut := &defaultRequeueHandler{namespace: testNamespace, clientSet: clientSetMock}

		// when
		actual, err := sut.noLongerHandleRequeueing(testCtx, finishedBackup)

		// then
		require.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, actual)
		assert.Equal(t, time.Duration(0), finishedBackup.Status.RequeueTimeNanos)
	})
}

func TestNewRequeueHandler(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// when
		handler := NewRequeueHandler(newMockEcosystemInterface(t), newMockEventRecorder(t), testNamespace)

		// then
		require.NotNil(t, handler)
	})
}
