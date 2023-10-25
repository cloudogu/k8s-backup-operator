package restore

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/config"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
)

var testCtx = context.TODO()

var testNamespace = "ecosystem-test"
var testRestore = "test-restore"

func TestNewRestoreReconciler(t *testing.T) {
	t.Run("should create restore reconciler", func(t *testing.T) {
		// when
		actual := NewRestoreReconciler(nil, nil, "default", nil, nil)

		// then
		assert.NotNil(t, actual)
	})
}

func Test_restoreReconciler_Reconcile(t *testing.T) {
	t.Run("should fail on getting restore", func(t *testing.T) {
		// given
		request := ctrl.Request{NamespacedName: types.NamespacedName{Name: testRestore}}
		restoreClientMock := newMockEcosystemRestoreInterface(t)
		restoreClientMock.EXPECT().Get(testCtx, testRestore, metav1.GetOptions{}).Return(nil, assert.AnError)
		v1alpha1Mock := newMockEcosystemV1Alpha1Interface(t)
		v1alpha1Mock.EXPECT().Restores(testNamespace).Return(restoreClientMock)
		clientSetMock := newMockEcosystemInterface(t)
		clientSetMock.EXPECT().EcosystemV1Alpha1().Return(v1alpha1Mock)
		sut := &restoreReconciler{
			namespace: testNamespace,
			clientSet: clientSetMock,
		}

		// when
		actual, err := sut.Reconcile(testCtx, request)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.Equal(t, ctrl.Result{}, actual)
	})

	t.Run("deletion tests", func(t *testing.T) {
		t.Run("should fail to handle requeue on deletion error", func(t *testing.T) {
			// given
			request := ctrl.Request{NamespacedName: types.NamespacedName{Name: testRestore}}
			restore := &v1.Restore{ObjectMeta: metav1.ObjectMeta{
				Name:              testRestore,
				Namespace:         testNamespace,
				DeletionTimestamp: &metav1.Time{Time: time.Now()},
			}}

			restoreClientMock := newMockEcosystemRestoreInterface(t)
			restoreClientMock.EXPECT().Get(testCtx, testRestore, metav1.GetOptions{}).Return(restore, nil)
			v1alpha1Mock := newMockEcosystemV1Alpha1Interface(t)
			v1alpha1Mock.EXPECT().Restores(testNamespace).Return(restoreClientMock)
			clientSetMock := newMockEcosystemInterface(t)
			clientSetMock.EXPECT().EcosystemV1Alpha1().Return(v1alpha1Mock)

			managerMock := newMockRestoreManager(t)
			managerMock.EXPECT().delete(testCtx, restore).Return(assert.AnError)
			recorderMock := newMockEventRecorder(t)
			recorderMock.EXPECT().Event(restore, corev1.EventTypeWarning, v1.DeleteEventReason, "Delete failed. Reason: assert.AnError general error for testing").Return()
			recorderMock.EXPECT().Eventf(restore, corev1.EventTypeWarning, RequeueEventReason, "Failed to requeue the %s.", "delete")
			requeueHandlerMock := newMockRequeueHandler(t)
			requeueHandlerMock.EXPECT().Handle(testCtx, "Delete of restore test-restore failed", restore, assert.AnError, v1.RestoreStatusNew).Return(reconcile.Result{}, assert.AnError)

			sut := &restoreReconciler{
				namespace:      testNamespace,
				clientSet:      clientSetMock,
				manager:        managerMock,
				recorder:       recorderMock,
				requeueHandler: requeueHandlerMock,
			}

			// when
			actual, err := sut.Reconcile(testCtx, request)

			// then
			require.Error(t, err)
			assert.ErrorIs(t, err, assert.AnError)
			assert.ErrorContains(t, err, "failed to handle requeue")
			assert.Equal(t, ctrl.Result{}, actual)
		})
		t.Run("should succeed to handle requeue on deletion error", func(t *testing.T) {
			// given
			request := ctrl.Request{NamespacedName: types.NamespacedName{Name: testRestore}}
			restore := &v1.Restore{ObjectMeta: metav1.ObjectMeta{
				Name:              testRestore,
				Namespace:         testNamespace,
				DeletionTimestamp: &metav1.Time{Time: time.Now()},
			}}

			restoreClientMock := newMockEcosystemRestoreInterface(t)
			restoreClientMock.EXPECT().Get(testCtx, testRestore, metav1.GetOptions{}).Return(restore, nil)
			v1alpha1Mock := newMockEcosystemV1Alpha1Interface(t)
			v1alpha1Mock.EXPECT().Restores(testNamespace).Return(restoreClientMock)
			clientSetMock := newMockEcosystemInterface(t)
			clientSetMock.EXPECT().EcosystemV1Alpha1().Return(v1alpha1Mock)

			managerMock := newMockRestoreManager(t)
			managerMock.EXPECT().delete(testCtx, restore).Return(assert.AnError)
			recorderMock := newMockEventRecorder(t)
			recorderMock.EXPECT().Event(restore, corev1.EventTypeWarning, v1.DeleteEventReason, "Delete failed. Reason: assert.AnError general error for testing").Return()
			requeueHandlerMock := newMockRequeueHandler(t)
			requeueHandlerMock.EXPECT().Handle(testCtx, "Delete of restore test-restore failed", restore, assert.AnError, v1.RestoreStatusNew).Return(reconcile.Result{Requeue: true}, nil)

			sut := &restoreReconciler{
				namespace:      testNamespace,
				clientSet:      clientSetMock,
				manager:        managerMock,
				recorder:       recorderMock,
				requeueHandler: requeueHandlerMock,
			}

			// when
			actual, err := sut.Reconcile(testCtx, request)

			// then
			require.NoError(t, err)
			assert.Equal(t, ctrl.Result{Requeue: true}, actual)
		})
		t.Run("should succeed with delete", func(t *testing.T) {
			// given
			request := ctrl.Request{NamespacedName: types.NamespacedName{Name: testRestore}}
			restore := &v1.Restore{ObjectMeta: metav1.ObjectMeta{
				Name:              testRestore,
				Namespace:         testNamespace,
				DeletionTimestamp: &metav1.Time{Time: time.Now()},
			}}

			restoreClientMock := newMockEcosystemRestoreInterface(t)
			restoreClientMock.EXPECT().Get(testCtx, testRestore, metav1.GetOptions{}).Return(restore, nil)
			v1alpha1Mock := newMockEcosystemV1Alpha1Interface(t)
			v1alpha1Mock.EXPECT().Restores(testNamespace).Return(restoreClientMock)
			clientSetMock := newMockEcosystemInterface(t)
			clientSetMock.EXPECT().EcosystemV1Alpha1().Return(v1alpha1Mock)

			managerMock := newMockRestoreManager(t)
			managerMock.EXPECT().delete(testCtx, restore).Return(nil)
			recorderMock := newMockEventRecorder(t)
			recorderMock.EXPECT().Event(restore, corev1.EventTypeNormal, v1.DeleteEventReason, "Delete successful").Return()
			requeueHandlerMock := newMockRequeueHandler(t)
			requeueHandlerMock.EXPECT().Handle(testCtx, "Delete of restore test-restore failed", restore, nil, v1.RestoreStatusNew).Return(reconcile.Result{}, nil)

			sut := &restoreReconciler{
				namespace:      testNamespace,
				clientSet:      clientSetMock,
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
	})

	t.Run("ignore tests", func(t *testing.T) {
		t.Run("should ignore when status is failed", func(t *testing.T) {
			// given
			request := ctrl.Request{NamespacedName: types.NamespacedName{Name: testRestore}}
			restore := &v1.Restore{ObjectMeta: metav1.ObjectMeta{
				Name:      testRestore,
				Namespace: testNamespace,
			}, Status: v1.RestoreStatus{Status: v1.RestoreStatusFailed}}

			restoreClientMock := newMockEcosystemRestoreInterface(t)
			restoreClientMock.EXPECT().Get(testCtx, testRestore, metav1.GetOptions{}).Return(restore, nil)
			v1alpha1Mock := newMockEcosystemV1Alpha1Interface(t)
			v1alpha1Mock.EXPECT().Restores(testNamespace).Return(restoreClientMock)
			clientSetMock := newMockEcosystemInterface(t)
			clientSetMock.EXPECT().EcosystemV1Alpha1().Return(v1alpha1Mock)

			sut := &restoreReconciler{
				namespace: testNamespace,
				clientSet: clientSetMock,
			}

			// when
			actual, err := sut.Reconcile(testCtx, request)

			// then
			require.NoError(t, err)
			assert.Equal(t, ctrl.Result{}, actual)
		})
		t.Run("should ignore when status is unknown", func(t *testing.T) {
			// given
			request := ctrl.Request{NamespacedName: types.NamespacedName{Name: testRestore}}
			restore := &v1.Restore{ObjectMeta: metav1.ObjectMeta{
				Name:      testRestore,
				Namespace: testNamespace,
			}, Status: v1.RestoreStatus{Status: "some-unknown-status"}}

			restoreClientMock := newMockEcosystemRestoreInterface(t)
			restoreClientMock.EXPECT().Get(testCtx, testRestore, metav1.GetOptions{}).Return(restore, nil)
			v1alpha1Mock := newMockEcosystemV1Alpha1Interface(t)
			v1alpha1Mock.EXPECT().Restores(testNamespace).Return(restoreClientMock)
			clientSetMock := newMockEcosystemInterface(t)
			clientSetMock.EXPECT().EcosystemV1Alpha1().Return(v1alpha1Mock)

			sut := &restoreReconciler{
				namespace: testNamespace,
				clientSet: clientSetMock,
			}

			// when
			actual, err := sut.Reconcile(testCtx, request)

			// then
			require.NoError(t, err)
			assert.Equal(t, ctrl.Result{}, actual)
		})
	})

	t.Run("creation tests", func(t *testing.T) {
		t.Run("should fail to handle requeue on create error", func(t *testing.T) {
			// given
			request := ctrl.Request{NamespacedName: types.NamespacedName{Name: testRestore}}
			restore := &v1.Restore{ObjectMeta: metav1.ObjectMeta{
				Name:      testRestore,
				Namespace: testNamespace,
			}, Status: v1.RestoreStatus{Status: v1.RestoreStatusNew}}

			restoreClientMock := newMockEcosystemRestoreInterface(t)
			restoreClientMock.EXPECT().Get(testCtx, testRestore, metav1.GetOptions{}).Return(restore, nil)
			v1alpha1Mock := newMockEcosystemV1Alpha1Interface(t)
			v1alpha1Mock.EXPECT().Restores(testNamespace).Return(restoreClientMock)
			clientSetMock := newMockEcosystemInterface(t)
			clientSetMock.EXPECT().EcosystemV1Alpha1().Return(v1alpha1Mock)

			managerMock := newMockRestoreManager(t)
			managerMock.EXPECT().create(testCtx, restore).Return(assert.AnError)
			recorderMock := newMockEventRecorder(t)
			recorderMock.EXPECT().Event(restore, corev1.EventTypeWarning, v1.CreateEventReason, "Creation failed. Reason: assert.AnError general error for testing").Return()
			recorderMock.EXPECT().Eventf(restore, corev1.EventTypeWarning, RequeueEventReason, "Failed to requeue the %s.", "creation")
			requeueHandlerMock := newMockRequeueHandler(t)
			requeueHandlerMock.EXPECT().Handle(testCtx, "Creation of restore test-restore failed", restore, assert.AnError, v1.RestoreStatusNew).Return(reconcile.Result{}, assert.AnError)

			sut := &restoreReconciler{
				namespace:      testNamespace,
				clientSet:      clientSetMock,
				manager:        managerMock,
				recorder:       recorderMock,
				requeueHandler: requeueHandlerMock,
			}

			// when
			actual, err := sut.Reconcile(testCtx, request)

			// then
			require.Error(t, err)
			assert.ErrorIs(t, err, assert.AnError)
			assert.ErrorContains(t, err, "failed to handle requeue")
			assert.Equal(t, ctrl.Result{}, actual)
		})
		t.Run("should succeed to handle requeue on creation error", func(t *testing.T) {
			// given
			request := ctrl.Request{NamespacedName: types.NamespacedName{Name: testRestore}}
			restore := &v1.Restore{ObjectMeta: metav1.ObjectMeta{
				Name:      testRestore,
				Namespace: testNamespace,
			}, Status: v1.RestoreStatus{Status: v1.RestoreStatusNew}}

			restoreClientMock := newMockEcosystemRestoreInterface(t)
			restoreClientMock.EXPECT().Get(testCtx, testRestore, metav1.GetOptions{}).Return(restore, nil)
			v1alpha1Mock := newMockEcosystemV1Alpha1Interface(t)
			v1alpha1Mock.EXPECT().Restores(testNamespace).Return(restoreClientMock)
			clientSetMock := newMockEcosystemInterface(t)
			clientSetMock.EXPECT().EcosystemV1Alpha1().Return(v1alpha1Mock)

			managerMock := newMockRestoreManager(t)
			managerMock.EXPECT().create(testCtx, restore).Return(assert.AnError)
			recorderMock := newMockEventRecorder(t)
			recorderMock.EXPECT().Event(restore, corev1.EventTypeWarning, v1.CreateEventReason, "Creation failed. Reason: assert.AnError general error for testing").Return()
			requeueHandlerMock := newMockRequeueHandler(t)
			requeueHandlerMock.EXPECT().Handle(testCtx, "Creation of restore test-restore failed", restore, assert.AnError, v1.RestoreStatusNew).Return(reconcile.Result{Requeue: true}, nil)

			sut := &restoreReconciler{
				namespace:      testNamespace,
				clientSet:      clientSetMock,
				manager:        managerMock,
				recorder:       recorderMock,
				requeueHandler: requeueHandlerMock,
			}

			// when
			actual, err := sut.Reconcile(testCtx, request)

			// then
			require.NoError(t, err)
			assert.Equal(t, ctrl.Result{Requeue: true}, actual)
		})
		t.Run("should succeed with create", func(t *testing.T) {
			// given
			request := ctrl.Request{NamespacedName: types.NamespacedName{Name: testRestore}}
			restore := &v1.Restore{ObjectMeta: metav1.ObjectMeta{
				Name:      testRestore,
				Namespace: testNamespace,
			}, Status: v1.RestoreStatus{Status: v1.RestoreStatusNew}}

			restoreClientMock := newMockEcosystemRestoreInterface(t)
			restoreClientMock.EXPECT().Get(testCtx, testRestore, metav1.GetOptions{}).Return(restore, nil)
			v1alpha1Mock := newMockEcosystemV1Alpha1Interface(t)
			v1alpha1Mock.EXPECT().Restores(testNamespace).Return(restoreClientMock)
			clientSetMock := newMockEcosystemInterface(t)
			clientSetMock.EXPECT().EcosystemV1Alpha1().Return(v1alpha1Mock)

			managerMock := newMockRestoreManager(t)
			managerMock.EXPECT().create(testCtx, restore).Return(nil)
			recorderMock := newMockEventRecorder(t)
			recorderMock.EXPECT().Event(restore, corev1.EventTypeNormal, v1.CreateEventReason, "Creation successful").Return()
			requeueHandlerMock := newMockRequeueHandler(t)
			requeueHandlerMock.EXPECT().Handle(testCtx, "Creation of restore test-restore failed", restore, nil, v1.RestoreStatusNew).Return(reconcile.Result{}, nil)

			sut := &restoreReconciler{
				namespace:      testNamespace,
				clientSet:      clientSetMock,
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
	})
}

func Test_restoreReconciler_SetupWithManager(t *testing.T) {
	t.Run("should fail", func(t *testing.T) {
		// given
		sut := &restoreReconciler{}

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

		sut := &restoreReconciler{}

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

	scheme.AddKnownTypes(gv, &v1.Restore{})
	return scheme
}
