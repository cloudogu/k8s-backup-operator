package backup

import (
	"errors"
	v1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	v12 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/config"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"testing"
	"time"
)

var testNamespace = "ecosystem-test"

func TestNewBackupReconciler(t *testing.T) {
	t.Run("should create backup reconciler", func(t *testing.T) {
		// when
		actual := NewBackupReconciler(nil, nil, "default", nil, nil)

		// then
		assert.NotNil(t, actual)
	})
}

func Test_backupReconciler_Reconcile(t *testing.T) {
	t.Run("should succeed with create on empty status", func(t *testing.T) {
		// given
		backupName := "backup"

		request := ctrl.Request{NamespacedName: types.NamespacedName{
			Namespace: testNamespace,
			Name:      backupName,
		}}
		backup := &v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: backupName, Namespace: testNamespace}}

		clientSetMock := newMockEcosystemInterface(t)
		backupInterfaceMock := newMockEcosystemBackupInterface(t)
		backupInterfaceMock.EXPECT().Get(testCtx, backupName, metav1.GetOptions{}).Return(backup, nil)
		backupClientGetterMock := newMockBackupV1Alpha1Interface(t)
		backupClientGetterMock.EXPECT().Backups(testNamespace).Return(backupInterfaceMock)
		clientSetMock.EXPECT().EcosystemV1Alpha1().Return(backupClientGetterMock)

		managerMock := newMockBackupControllerManager(t)
		managerMock.EXPECT().create(testCtx, backup).Return(nil)
		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(backup, v12.EventTypeNormal, "Creation", "Creation successful")

		requeueHandlerMock := newMockRequeueHandler(t)
		requeueHandlerMock.EXPECT().Handle(testCtx, "Creation failed with backup backup", backup, nil, "").Return(ctrl.Result{}, nil)

		sut := &backupReconciler{clientSet: clientSetMock, namespace: testNamespace, manager: managerMock, recorder: recorderMock, requeueHandler: requeueHandlerMock}

		// when
		actual, err := sut.Reconcile(testCtx, request)

		// then
		require.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, actual)
	})

	t.Run("should ignore on completed status", func(t *testing.T) {
		// given
		backupName := "backup"

		request := ctrl.Request{NamespacedName: types.NamespacedName{
			Namespace: testNamespace,
			Name:      backupName,
		}}
		backup := &v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: backupName, Namespace: testNamespace}, Status: v1.BackupStatus{Status: "completed"}}

		clientSetMock := newMockEcosystemInterface(t)
		backupInterfaceMock := newMockEcosystemBackupInterface(t)
		backupInterfaceMock.EXPECT().Get(testCtx, backupName, metav1.GetOptions{}).Return(backup, nil)
		backupClientGetterMock := newMockBackupV1Alpha1Interface(t)
		backupClientGetterMock.EXPECT().Backups(testNamespace).Return(backupInterfaceMock)
		clientSetMock.EXPECT().EcosystemV1Alpha1().Return(backupClientGetterMock)

		sut := &backupReconciler{clientSet: clientSetMock, namespace: testNamespace}

		// when
		actual, err := sut.Reconcile(testCtx, request)

		// then
		require.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, actual)
	})

	t.Run("should sync status on syncFromProvider flag", func(t *testing.T) {
		// given
		backupName := "backup"

		request := ctrl.Request{NamespacedName: types.NamespacedName{
			Namespace: testNamespace,
			Name:      backupName,
		}}
		backup := &v1.Backup{
			ObjectMeta: metav1.ObjectMeta{Name: backupName, Namespace: testNamespace},
			Spec:       v1.BackupSpec{SyncedFromProvider: true},
		}

		managerMock := newMockBackupControllerManager(t)
		managerMock.EXPECT().syncStatus(testCtx, backup).Return(nil)

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(backup, v12.EventTypeNormal, "SyncStatus", "SyncStatus successful")

		requeueHandlerMock := newMockRequeueHandler(t)
		requeueHandlerMock.EXPECT().Handle(testCtx, "SyncStatus failed with backup backup", backup, nil, "").Return(ctrl.Result{}, nil)

		clientSetMock := newMockEcosystemInterface(t)
		backupInterfaceMock := newMockEcosystemBackupInterface(t)
		backupInterfaceMock.EXPECT().Get(testCtx, backupName, metav1.GetOptions{}).Return(backup, nil)
		backupClientGetterMock := newMockBackupV1Alpha1Interface(t)
		backupClientGetterMock.EXPECT().Backups(testNamespace).Return(backupInterfaceMock)
		clientSetMock.EXPECT().EcosystemV1Alpha1().Return(backupClientGetterMock)

		sut := &backupReconciler{
			clientSet:      clientSetMock,
			namespace:      testNamespace,
			manager:        managerMock,
			recorder:       recorderMock,
			requeueHandler: requeueHandlerMock,
		}

		// when
		actual, err := sut.Reconcile(testCtx, request)

		// then
		require.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, actual)
	})

	t.Run("should delete with deletion timestamp", func(t *testing.T) {
		// given
		backupName := "backup"

		request := ctrl.Request{NamespacedName: types.NamespacedName{
			Namespace: testNamespace,
			Name:      backupName,
		}}
		now := metav1.NewTime(time.Now())
		backup := &v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: backupName, Namespace: testNamespace, DeletionTimestamp: &now}, Status: v1.BackupStatus{Status: "completed"}}

		clientSetMock := newMockEcosystemInterface(t)
		backupInterfaceMock := newMockEcosystemBackupInterface(t)
		backupInterfaceMock.EXPECT().Get(testCtx, backupName, metav1.GetOptions{}).Return(backup, nil)
		backupClientGetterMock := newMockBackupV1Alpha1Interface(t)
		backupClientGetterMock.EXPECT().Backups(testNamespace).Return(backupInterfaceMock)
		clientSetMock.EXPECT().EcosystemV1Alpha1().Return(backupClientGetterMock)

		managerMock := newMockBackupControllerManager(t)
		managerMock.EXPECT().delete(testCtx, backup).Return(nil)

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(backup, v12.EventTypeNormal, "Delete", "Delete successful")

		requeueHandlerMock := newMockRequeueHandler(t)
		requeueHandlerMock.EXPECT().Handle(testCtx, "Delete failed with backup backup", backup, nil, v1.BackupStatusCompleted).Return(ctrl.Result{}, nil)

		sut := &backupReconciler{clientSet: clientSetMock, namespace: testNamespace, manager: managerMock, recorder: recorderMock, requeueHandler: requeueHandlerMock}

		// when
		actual, err := sut.Reconcile(testCtx, request)

		// then
		require.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, actual)
	})

	t.Run("should return no error when manager throws requeue error", func(t *testing.T) {
		// given
		backupName := "backup"

		request := ctrl.Request{NamespacedName: types.NamespacedName{
			Namespace: testNamespace,
			Name:      backupName,
		}}
		now := metav1.NewTime(time.Now())
		backup := &v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: backupName, Namespace: testNamespace, DeletionTimestamp: &now}, Status: v1.BackupStatus{Status: "completed"}}

		clientSetMock := newMockEcosystemInterface(t)
		backupInterfaceMock := newMockEcosystemBackupInterface(t)
		backupInterfaceMock.EXPECT().Get(testCtx, backupName, metav1.GetOptions{}).Return(backup, nil)
		backupClientGetterMock := newMockBackupV1Alpha1Interface(t)
		backupClientGetterMock.EXPECT().Backups(testNamespace).Return(backupInterfaceMock)
		clientSetMock.EXPECT().EcosystemV1Alpha1().Return(backupClientGetterMock)

		testErr := requeueError{errors.New("test")}
		managerMock := newMockBackupControllerManager(t)
		managerMock.EXPECT().delete(testCtx, backup).Return(testErr)

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(backup, v12.EventTypeWarning, "Delete", "Delete failed. Reason: test")

		requeueHandlerMock := newMockRequeueHandler(t)
		requeueHandlerMock.EXPECT().Handle(testCtx, "Delete failed with backup backup", backup, testErr, v1.BackupStatusCompleted).Return(ctrl.Result{}, nil)

		sut := &backupReconciler{clientSet: clientSetMock, namespace: testNamespace, manager: managerMock, recorder: recorderMock, requeueHandler: requeueHandlerMock}

		// when
		actual, err := sut.Reconcile(testCtx, request)

		// then
		require.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, actual)
	})

	t.Run("should return error if the requeue handler returns an error", func(t *testing.T) {
		// given
		backupName := "backup"

		request := ctrl.Request{NamespacedName: types.NamespacedName{
			Namespace: testNamespace,
			Name:      backupName,
		}}
		now := metav1.NewTime(time.Now())
		backup := &v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: backupName, Namespace: testNamespace, DeletionTimestamp: &now}, Status: v1.BackupStatus{Status: "completed"}}

		clientSetMock := newMockEcosystemInterface(t)
		backupInterfaceMock := newMockEcosystemBackupInterface(t)
		backupInterfaceMock.EXPECT().Get(testCtx, backupName, metav1.GetOptions{}).Return(backup, nil)
		backupClientGetterMock := newMockBackupV1Alpha1Interface(t)
		backupClientGetterMock.EXPECT().Backups(testNamespace).Return(backupInterfaceMock)
		clientSetMock.EXPECT().EcosystemV1Alpha1().Return(backupClientGetterMock)

		testErr := requeueError{errors.New("test")}
		managerMock := newMockBackupControllerManager(t)
		managerMock.EXPECT().delete(testCtx, backup).Return(testErr)

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(backup, v12.EventTypeWarning, "Delete", "Delete failed. Reason: test")
		recorderMock.EXPECT().Eventf(backup, v12.EventTypeWarning, "Requeue", "Failed to requeue the %s.", "delete")

		requeueHandlerMock := newMockRequeueHandler(t)
		requeueHandlerMock.EXPECT().Handle(testCtx, "Delete failed with backup backup", backup, testErr, v1.BackupStatusCompleted).Return(ctrl.Result{}, assert.AnError)

		sut := &backupReconciler{clientSet: clientSetMock, namespace: testNamespace, manager: managerMock, recorder: recorderMock, requeueHandler: requeueHandlerMock}

		// when
		actual, err := sut.Reconcile(testCtx, request)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to handle requeue")
		assert.Equal(t, ctrl.Result{}, actual)
	})

	t.Run("should error on error getting resource", func(t *testing.T) {
		// given
		backupName := "backup"

		request := ctrl.Request{NamespacedName: types.NamespacedName{
			Namespace: testNamespace,
			Name:      backupName,
		}}

		clientSetMock := newMockEcosystemInterface(t)
		backupInterfaceMock := newMockEcosystemBackupInterface(t)
		backupInterfaceMock.EXPECT().Get(testCtx, backupName, metav1.GetOptions{}).Return(nil, assert.AnError)
		backupClientGetterMock := newMockBackupV1Alpha1Interface(t)
		backupClientGetterMock.EXPECT().Backups(testNamespace).Return(backupInterfaceMock)
		clientSetMock.EXPECT().EcosystemV1Alpha1().Return(backupClientGetterMock)

		sut := &backupReconciler{clientSet: clientSetMock, namespace: testNamespace}

		// when
		actual, err := sut.Reconcile(testCtx, request)

		// then
		require.Error(t, err)
		assert.Equal(t, ctrl.Result{}, actual)
	})
}

type requeueError struct {
	error
}

func (rq *requeueError) GetRequeueTime(requeueTimeNanos time.Duration) time.Duration {
	return requeueTimeNanos
}

func Test_backupReconciler_SetupWithManager(t *testing.T) {
	t.Run("should fail", func(t *testing.T) {
		// given
		sut := &backupReconciler{}

		// when
		err := sut.SetupWithManager(nil)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "must provide a non-nil Manager")
	})
	t.Run("should succeed", func(t *testing.T) {
		// given
		ctrlManMock := newMockControllerManager(t)
		ctrlManMock.EXPECT().GetControllerOptions().Return(config.Controller{})
		ctrlManMock.EXPECT().GetScheme().Return(createScheme(t))
		logger := log.FromContext(testCtx)
		ctrlManMock.EXPECT().GetLogger().Return(logger)
		ctrlManMock.EXPECT().Add(mock.Anything).Return(nil)
		ctrlManMock.EXPECT().GetCache().Return(nil)

		sut := &backupReconciler{}

		// when
		err := sut.SetupWithManager(ctrlManMock)

		// then
		require.NoError(t, err)
	})
}

func createScheme(t *testing.T) *runtime.Scheme {
	t.Helper()

	scheme := runtime.NewScheme()
	gv, err := schema.ParseGroupVersion("k8s.cloudogu.com/v1")
	assert.NoError(t, err)

	scheme.AddKnownTypes(gv, &v1.Backup{})
	return scheme
}

func Test_evaluateRequiredOperation(t *testing.T) {
	now := metav1.NewTime(time.Now())
	type args struct {
		backup *v1.Backup
	}
	tests := []struct {
		name string
		args args
		want operation
	}{
		{name: "should return create operation on empty status", args: args{backup: &v1.Backup{Status: v1.BackupStatus{Status: ""}}}, want: operationCreate},
		{name: "should return ignore on completed status", args: args{backup: &v1.Backup{Status: v1.BackupStatus{Status: v1.BackupStatusCompleted}}}, want: operationIgnore},
		{name: "should return ignore on in progress status", args: args{backup: &v1.Backup{Status: v1.BackupStatus{Status: v1.BackupStatusInProgress}}}, want: operationIgnore},
		{name: "should return ignore on deleting status", args: args{backup: &v1.Backup{Status: v1.BackupStatus{Status: v1.BackupStatusDeleting}}}, want: operationIgnore},
		{name: "should return ignore on failed status", args: args{backup: &v1.Backup{Status: v1.BackupStatus{Status: v1.BackupStatusFailed}}}, want: operationIgnore},
		{name: "should return delete with deletionTimestamp", args: args{backup: &v1.Backup{ObjectMeta: metav1.ObjectMeta{DeletionTimestamp: &now}, Status: v1.BackupStatus{Status: v1.BackupStatusCompleted}}}, want: operationDelete},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, evaluateRequiredOperation(tt.args.backup), "evaluateRequiredOperation(%v)", tt.args.backup)
		})
	}
}
