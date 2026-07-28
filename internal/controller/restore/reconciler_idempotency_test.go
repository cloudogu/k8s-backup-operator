package restore

import (
	"context"
	"testing"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

// The reconciler is built without a manager, recorder, or requeue handler, so an external
// action would panic instead of being merely unexpected.
func TestRepeatedReconciliationOfACompletedRestorePerformsNoWritesOrExternalActions(t *testing.T) {
	restore := &backupv1.Restore{
		ObjectMeta: metav1.ObjectMeta{Name: testRestore, Namespace: testNamespace},
		Status: backupv1.RestoreStatus{
			Status: backupv1.RestoreStatusCompleted,
			Conditions: []metav1.Condition{{
				Type:               backupv1.ConditionSuccessful,
				Status:             metav1.ConditionTrue,
				Reason:             ReasonRestoreCompleted,
				LastTransitionTime: metav1.Now(),
			}},
		},
	}

	restoreClient := newMockEcosystemRestoreInterface(t)
	restoreClient.EXPECT().Get(testCtx, testRestore, metav1.GetOptions{}).Return(restore, nil).Twice()
	v1Client := newMockEcosystemV1Alpha1Interface(t)
	v1Client.EXPECT().Restores(testNamespace).Return(restoreClient)
	clientSet := newMockEcosystemInterface(t)
	clientSet.EXPECT().EcosystemV1Alpha1().Return(v1Client)

	reconciler := &restoreReconciler{
		namespace: testNamespace,
		clientSet: clientSet,
	}

	request := ctrl.Request{NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testRestore}}

	for range 2 {
		result, err := reconciler.Reconcile(testCtx, request)
		require.NoError(t, err)
		require.Equal(t, ctrl.Result{}, result)
	}

	// No UpdateStatus is expected on the mock, so any write would already have failed the test.
	require.Equal(t, backupv1.RestoreStatusCompleted, restore.Status.Status)
}

func TestRepeatedReconciliationOfACompletedLegacyRestoreWritesConditions(t *testing.T) {
	restore := &backupv1.Restore{
		ObjectMeta: metav1.ObjectMeta{Name: testRestore, Namespace: testNamespace},
		Status:     backupv1.RestoreStatus{Status: backupv1.RestoreStatusCompleted},
	}

	// The persisted state has to survive between the two reconciles
	stored := restore
	restoreClient := newMockEcosystemRestoreInterface(t)
	restoreClient.EXPECT().Get(testCtx, testRestore, metav1.GetOptions{}).
		RunAndReturn(func(_ context.Context, _ string, _ metav1.GetOptions) (*backupv1.Restore, error) {
			return stored, nil
		}).Twice()
	restoreClient.EXPECT().UpdateStatus(testCtx, mock.Anything, metav1.UpdateOptions{}).
		RunAndReturn(func(_ context.Context, written *backupv1.Restore, _ metav1.UpdateOptions) (*backupv1.Restore, error) {
			stored = written

			return written, nil
		}).Once()
	v1Client := newMockEcosystemV1Alpha1Interface(t)
	v1Client.EXPECT().Restores(testNamespace).Return(restoreClient)
	clientSet := newMockEcosystemInterface(t)
	clientSet.EXPECT().EcosystemV1Alpha1().Return(v1Client)

	reconciler := &restoreReconciler{
		namespace: testNamespace,
		clientSet: clientSet,
	}

	request := ctrl.Request{NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testRestore}}

	for range 2 {
		result, err := reconciler.Reconcile(testCtx, request)
		require.NoError(t, err)
		require.Equal(t, ctrl.Result{}, result)
	}

	require.Equal(t, backupv1.RestoreStatusCompleted, stored.Status.Status)
	require.Equal(t, metav1.ConditionTrue, findSuccessfulCondition(stored).Status)
	require.Equal(t, ReasonMigratedFromLegacyStatus, findSuccessfulCondition(stored).Reason)
}

func TestAParentNotFoundRacePerformsNoWritesOrExternalActions(t *testing.T) {
	notFound := apierrors.NewNotFound(
		schema.GroupResource{Group: "k8s.cloudogu.com", Resource: "restores"},
		testRestore,
	)

	restoreClient := newMockEcosystemRestoreInterface(t)
	restoreClient.EXPECT().Get(testCtx, testRestore, metav1.GetOptions{}).Return(nil, notFound).Twice()
	v1Client := newMockEcosystemV1Alpha1Interface(t)
	v1Client.EXPECT().Restores(testNamespace).Return(restoreClient).Twice()
	clientSet := newMockEcosystemInterface(t)
	clientSet.EXPECT().EcosystemV1Alpha1().Return(v1Client).Twice()

	reconciler := &restoreReconciler{
		namespace: testNamespace,
		clientSet: clientSet,
	}
	request := ctrl.Request{NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testRestore}}

	for range 2 {
		result, err := reconciler.Reconcile(testCtx, request)
		require.NoError(t, err)
		require.Equal(t, ctrl.Result{}, result)
	}
}
