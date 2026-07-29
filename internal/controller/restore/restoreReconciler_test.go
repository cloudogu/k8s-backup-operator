package restore

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/config"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1 "github.com/cloudogu/k8s-backup-lib/api/v1"
)

var testCtx = context.TODO()

var testNamespace = "ecosystem-test"
var testRestore = "test-restore"

// expectSuccessfulConditionUpdate expects exactly one status update carrying a Successful condition
// of the given status and reason, and returns the written object so a following Get sees it.
func expectSuccessfulConditionUpdate(restoreClient *mockEcosystemRestoreInterface, status metav1.ConditionStatus, reason string) {
	matchesCondition := mock.MatchedBy(func(restore *v1.Restore) bool {
		condition := findSuccessfulCondition(restore)

		return condition != nil && condition.Status == status && condition.Reason == reason
	})

	restoreClient.EXPECT().
		UpdateStatus(testCtx, matchesCondition, metav1.UpdateOptions{}).
		RunAndReturn(func(_ context.Context, written *v1.Restore, _ metav1.UpdateOptions) (*v1.Restore, error) {
			return written, nil
		}).
		Once()
}

// expectFailingSuccessfulConditionUpdate expects one status update carrying a Successful condition
// of the given status and reason, and fails it.
func expectFailingSuccessfulConditionUpdate(restoreClient *mockEcosystemRestoreInterface, status metav1.ConditionStatus, reason string, updateErr error) {
	matchesCondition := mock.MatchedBy(func(restore *v1.Restore) bool {
		condition := findSuccessfulCondition(restore)

		return condition != nil && condition.Status == status && condition.Reason == reason
	})

	restoreClient.EXPECT().
		UpdateStatus(testCtx, matchesCondition, metav1.UpdateOptions{}).
		Return(nil, updateErr).
		Once()
}

func TestNewRestoreReconciler(t *testing.T) {
	t.Run("should create restore reconciler", func(t *testing.T) {
		// when
		actual := NewRestoreReconciler(nil, nil, "default", nil)

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
		t.Run("should retry on deletion error", func(t *testing.T) {
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

			sut := &restoreReconciler{
				namespace: testNamespace,
				clientSet: clientSetMock,
				manager:   managerMock,
				recorder:  recorderMock,
			}

			// when
			actual, err := sut.Reconcile(testCtx, request)

			// then
			require.Error(t, err)
			assert.ErrorIs(t, err, assert.AnError)
			assert.ErrorContains(t, err, "Delete of restore test-restore failed")
			assert.Equal(t, ctrl.Result{}, actual)
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

			sut := &restoreReconciler{
				namespace: testNamespace,
				clientSet: clientSetMock,
				manager:   managerMock,
				recorder:  recorderMock,
			}

			// when
			actual, err := sut.Reconcile(testCtx, request)

			// then
			require.NoError(t, err)
			assert.Equal(t, ctrl.Result{}, actual)
		})
	})

	t.Run("ignore tests", func(t *testing.T) {
		t.Run("should migrate the legacy status of a failed restore once and then ignore it", func(t *testing.T) {
			// given
			request := ctrl.Request{NamespacedName: types.NamespacedName{Name: testRestore}}
			restore := &v1.Restore{ObjectMeta: metav1.ObjectMeta{
				Name:      testRestore,
				Namespace: testNamespace,
			}, Status: v1.RestoreStatus{Status: v1.RestoreStatusFailed}}

			restoreClientMock := newMockEcosystemRestoreInterface(t)
			restoreClientMock.EXPECT().Get(testCtx, testRestore, metav1.GetOptions{}).Return(restore, nil)
			expectSuccessfulConditionUpdate(restoreClientMock, metav1.ConditionFalse, ReasonMigratedFromLegacyStatus)
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
			assert.Equal(t, v1.RestoreStatusFailed, restore.Status.Status)
		})
		t.Run("should ignore a status phase without writing", func(t *testing.T) {
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
		t.Run("should retry on create error", func(t *testing.T) {
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

			sut := &restoreReconciler{
				namespace: testNamespace,
				clientSet: clientSetMock,
				manager:   managerMock,
				recorder:  recorderMock,
			}

			// when
			actual, err := sut.Reconcile(testCtx, request)

			// then
			require.Error(t, err)
			assert.ErrorIs(t, err, assert.AnError)
			assert.ErrorContains(t, err, "Creation of restore test-restore failed")
			assert.Equal(t, ctrl.Result{}, actual)
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

			sut := &restoreReconciler{
				namespace: testNamespace,
				clientSet: clientSetMock,
				manager:   managerMock,
				recorder:  recorderMock,
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

func Test_requiredOperation(t *testing.T) {
	successful := func(status metav1.ConditionStatus) []metav1.Condition {
		return []metav1.Condition{{Type: v1.ConditionSuccessful, Status: status, Reason: "TestReason", LastTransitionTime: metav1.Now()}}
	}

	for _, testCase := range []struct {
		name      string
		restore   *v1.Restore
		expected  operation
		reasonWhy string
	}{
		{
			name:      "delete a restore that is being deleted",
			restore:   &v1.Restore{ObjectMeta: metav1.ObjectMeta{DeletionTimestamp: &metav1.Time{Time: time.Now()}}},
			expected:  operationDelete,
			reasonWhy: "deletion wins over any outcome, including a terminal one",
		},
		{
			name:      "delete a completed restore that is being deleted",
			restore:   &v1.Restore{ObjectMeta: metav1.ObjectMeta{DeletionTimestamp: &metav1.Time{Time: time.Now()}}, Status: v1.RestoreStatus{Conditions: successful(metav1.ConditionTrue)}},
			expected:  operationDelete,
			reasonWhy: "deleting a finished restore is the normal case",
		},
		{
			name:      "create a fresh restore",
			restore:   &v1.Restore{},
			expected:  operationCreate,
			reasonWhy: "no outcome and no legacy status means the work has not started",
		},
		{
			name:      "ignore a successful restore",
			restore:   &v1.Restore{Status: v1.RestoreStatus{Conditions: successful(metav1.ConditionTrue)}},
			expected:  operationIgnore,
			reasonWhy: "terminal",
		},
		{
			name:      "ignore a failed restore",
			restore:   &v1.Restore{Status: v1.RestoreStatus{Conditions: successful(metav1.ConditionFalse)}},
			expected:  operationIgnore,
			reasonWhy: "terminal",
		},
		{
			name:      "ignore a restore with an unknown outcome",
			restore:   &v1.Restore{Status: v1.RestoreStatus{Conditions: successful(metav1.ConditionUnknown)}},
			expected:  operationIgnore,
			reasonWhy: "an interrupted restore may not repeat the destructive preparation; resuming needs the staged flow",
		},
		{
			name:      "ignore a legacy restore that is still in progress",
			restore:   &v1.Restore{Status: v1.RestoreStatus{Status: v1.RestoreStatusInProgress}},
			expected:  operationIgnore,
			reasonWhy: "same as an unknown outcome, reached through the deprecated scalar status",
		},
		{
			name:      "ignore a legacy restore with an uninterpretable status",
			restore:   &v1.Restore{Status: v1.RestoreStatus{Status: "some-unknown-status"}},
			expected:  operationIgnore,
			reasonWhy: "an unreadable legacy value must not start a destructive restore",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.expected, requiredOperation(testCase.restore), testCase.reasonWhy)
		})
	}
}
