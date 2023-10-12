package restore

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	k8sv1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
)

func Test_defaultRequeueHandler_Handle(t *testing.T) {
	t.Run("should exit early if there is no error", func(t *testing.T) {
		// given
		sut := &defaultRequeueHandler{}
		var originalErr error = nil
		restore := &k8sv1.Restore{}

		// when
		actual, err := sut.Handle(testCtx, "", restore, originalErr, "installing")

		// then
		require.NoError(t, err)
		assert.Equal(t, reconcile.Result{Requeue: false, RequeueAfter: 0}, actual)
	})
	t.Run("should exit early if error is not requeuable", func(t *testing.T) {
		// given
		sut := &defaultRequeueHandler{}
		var originalErr = assert.AnError
		restore := &k8sv1.Restore{}

		// when
		actual, err := sut.Handle(testCtx, "", restore, originalErr, "installing")

		// then
		require.NoError(t, err)
		assert.Equal(t, reconcile.Result{Requeue: false, RequeueAfter: 0}, actual)
	})
	t.Run("should fail to update restore status", func(t *testing.T) {
		// given
		restore := createRestore("ecosystem-restore-1", "ecosystem")

		restoreInterfaceMock := newMockEcosystemRestoreInterface(t)
		restoreInterfaceMock.EXPECT().Get(testCtx, restore.Name, mock.Anything).Return(restore, nil)
		restoreInterfaceMock.EXPECT().UpdateStatus(testCtx, restore, metav1.UpdateOptions{}).Return(nil, assert.AnError)
		restoreClientGetterMock := newMockEcosystemV1Alpha1Interface(t)
		restoreClientGetterMock.EXPECT().Restores(testNamespace).Return(restoreInterfaceMock)
		clientSetMock := newMockEcosystemInterface(t)
		clientSetMock.EXPECT().EcosystemV1Alpha1().Return(restoreClientGetterMock)

		sut := &defaultRequeueHandler{namespace: testNamespace, clientSet: clientSetMock}

		requeueErrMock := newMockRequeuableError(t)
		requeueErrMock.EXPECT().GetRequeueTime(mock.Anything).Return(30 * time.Second)

		// when
		actual, err := sut.Handle(testCtx, "", restore, requeueErrMock, "upgrading")

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to update restore status")

		assert.Equal(t, reconcile.Result{Requeue: false, RequeueAfter: 0}, actual)
	})
	t.Run("should succeed", func(t *testing.T) {
		// given
		restore := createRestore("ecosystem-restore-1", "ecosystem")

		restoreInterfaceMock := newMockEcosystemRestoreInterface(t)
		restoreInterfaceMock.EXPECT().Get(testCtx, restore.Name, mock.Anything).Return(restore, nil)
		restoreInterfaceMock.EXPECT().UpdateStatus(testCtx, restore, metav1.UpdateOptions{}).Return(restore, nil)
		restoreClientGetterMock := newMockEcosystemV1Alpha1Interface(t)
		restoreClientGetterMock.EXPECT().Restores(testNamespace).Return(restoreInterfaceMock)
		clientSetMock := newMockEcosystemInterface(t)
		clientSetMock.EXPECT().EcosystemV1Alpha1().Return(restoreClientGetterMock)

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Eventf(restore, "Normal", "Requeue", "Falling back to restore status %s: Trying again in %s.", "upgrading", "1s")

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

func Test_defaultRequeueHandler_noLongerHandleRequeueing(t *testing.T) {
	t.Run("reset requeue time to avoid further requeueing", func(t *testing.T) {
		// given
		finishedRestore := &k8sv1.Restore{Status: k8sv1.RestoreStatus{
			Status:           k8sv1.RestoreStatusCompleted,
			RequeueTimeNanos: 3000}}

		restoreInterfaceMock := newMockEcosystemRestoreInterface(t)
		restoreInterfaceMock.EXPECT().Get(testCtx, finishedRestore.Name, mock.Anything).Return(finishedRestore, nil)
		restoreInterfaceMock.EXPECT().UpdateStatus(testCtx, finishedRestore, metav1.UpdateOptions{}).Return(finishedRestore, nil)
		restoreClientGetterMock := newMockEcosystemV1Alpha1Interface(t)
		restoreClientGetterMock.EXPECT().Restores(testNamespace).Return(restoreInterfaceMock)
		clientSetMock := newMockEcosystemInterface(t)
		clientSetMock.EXPECT().EcosystemV1Alpha1().Return(restoreClientGetterMock)

		sut := &defaultRequeueHandler{namespace: testNamespace, clientSet: clientSetMock}

		// when
		actual, err := sut.noLongerHandleRequeueing(testCtx, finishedRestore)

		// then
		require.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, actual)
		assert.Equal(t, time.Duration(0), finishedRestore.Status.RequeueTimeNanos)
	})
}
