package restore

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/config"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1 "github.com/cloudogu/k8s-backup-lib/api/v1"
)

var testCtx = context.TODO()

var testNamespace = "ecosystem-test"
var testRestore = "test-restore"

const requeueAfterTest = time.Duration(5) * time.Second

func TestNewRestoreReconciler(t *testing.T) {
	t.Run("should create restore reconciler", func(t *testing.T) {
		// when
		actual := NewRestoreReconciler(nil, nil, "default", nil, nil, requeueAfterTest)

		// then
		assert.NotNil(t, actual)
	})
}

func Test_restoreReconciler_Reconcile(t *testing.T) {
	t.Run("should fail on getting restore", func(t *testing.T) {
		// given
		request := ctrl.Request{NamespacedName: types.NamespacedName{Name: testRestore}}
		failingGet := interceptor.Funcs{
			Get: func(_ context.Context, _ client.WithWatch, _ client.ObjectKey, _ client.Object, _ ...client.GetOption) error {
				return assert.AnError
			},
		}
		sut := &restoreReconciler{
			namespace: testNamespace,
			k8sClient: newTestClient(t, failingGet),
		}

		// when
		actual, err := sut.Reconcile(testCtx, request)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.Equal(t, ctrl.Result{}, actual)
	})

	t.Run("ignore tests", func(t *testing.T) {
		t.Run("should migrate the legacy status of a failed restore once and then ignore it", func(t *testing.T) {
			// given
			request := ctrl.Request{NamespacedName: types.NamespacedName{Name: testRestore}}
			restore := &v1.Restore{ObjectMeta: metav1.ObjectMeta{
				Name:      testRestore,
				Namespace: testNamespace,
			}, Status: v1.RestoreStatus{Status: v1.RestoreStatusFailed}}

			writes := &clientWrites{}
			testClient := newTestClient(t, writes.interceptor(), restore)
			sut := &restoreReconciler{namespace: testNamespace, k8sClient: testClient}

			// when
			actual, err := sut.Reconcile(testCtx, request)

			// then
			require.NoError(t, err)
			assert.Equal(t, ctrl.Result{}, actual)
			assert.Equal(t, 1, writes.parent.statusUpdates)
			assertSuccessfulCondition(t, testClient, testRestore, metav1.ConditionFalse, ReasonMigratedFromLegacyStatus)

			stored := &v1.Restore{}
			require.NoError(t, sut.k8sClient.Get(testCtx, client.ObjectKey{Namespace: testNamespace, Name: testRestore}, stored))
			assert.Equal(t, v1.RestoreStatusFailed, stored.Status.Status)
		})
		t.Run("should ignore a completed restore without writing", func(t *testing.T) {
			// given
			request := ctrl.Request{NamespacedName: types.NamespacedName{Name: testRestore}}
			restore := &v1.Restore{ObjectMeta: metav1.ObjectMeta{
				Name:      testRestore,
				Namespace: testNamespace,
			}, Status: v1.RestoreStatus{Status: v1.RestoreStatusCompleted, Conditions: []metav1.Condition{{
				Type:               v1.ConditionSucceeded,
				Status:             metav1.ConditionTrue,
				Reason:             ReasonRestoreCompleted,
				LastTransitionTime: metav1.Now(),
			}}}}

			writes := &clientWrites{}
			sut := &restoreReconciler{
				namespace: testNamespace,
				k8sClient: newTestClient(t, writes.interceptor(), restore),
			}

			// when
			actual, err := sut.Reconcile(testCtx, request)

			// then
			require.NoError(t, err)
			assert.Equal(t, ctrl.Result{}, actual)
			assert.Equal(t, 0, writes.total())
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
		// only the owned provider restore watch needs to map an owner reference back to its parent
		ctrlManMock.EXPECT().GetRESTMapper().Return(nil)

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
	// the owned provider restore must be resolvable, otherwise the Owns watch cannot be set up
	require.NoError(t, velerov1.AddToScheme(scheme))

	return scheme
}

func Test_requiredOperation(t *testing.T) {
	successful := func(status metav1.ConditionStatus) []metav1.Condition {
		return []metav1.Condition{{Type: v1.ConditionSucceeded, Status: status, Reason: "TestReason", LastTransitionTime: metav1.Now()}}
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
			name:      "continue a restore with an unknown outcome",
			restore:   &v1.Restore{Status: v1.RestoreStatus{Conditions: successful(metav1.ConditionUnknown)}},
			expected:  operationCreate,
			reasonWhy: "an unknown outcome means in flight, so the staged workflow resumes where it stopped",
		},
		{
			name:      "continue a legacy restore that is still in progress",
			restore:   &v1.Restore{Status: v1.RestoreStatus{Status: v1.RestoreStatusInProgress}},
			expected:  operationCreate,
			reasonWhy: "same as an unknown outcome, reached through the deprecated scalar status",
		},
		{
			name:      "continue a legacy restore with an uninterpretable status",
			restore:   &v1.Restore{Status: v1.RestoreStatus{Status: "some-unknown-status"}},
			expected:  operationCreate,
			reasonWhy: "an unreadable legacy value carries no outcome, and the child barrier guards the destructive stages",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.expected, requiredOperation(testCase.restore), testCase.reasonWhy)
		})
	}
}
