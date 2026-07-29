package restore

import (
	"testing"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// The reconciler is built without a manager or recorder, so an external action would panic instead of
// being merely unexpected.
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

	writes := &clientWrites{}
	reconciler := &restoreReconciler{
		namespace: testNamespace,
		k8sClient: newTestClient(t, writes.interceptor(), restore),
	}

	request := ctrl.Request{NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testRestore}}

	for range 2 {
		result, err := reconciler.Reconcile(testCtx, request)
		require.NoError(t, err)
		require.Equal(t, ctrl.Result{}, result)
	}

	require.Equal(t, 0, writes.total(), "a terminal restore must be reconciled without any write")
	stored := &backupv1.Restore{}
	require.NoError(t, reconciler.k8sClient.Get(testCtx, client.ObjectKey{Namespace: testNamespace, Name: testRestore}, stored))
	require.Equal(t, backupv1.RestoreStatusCompleted, stored.Status.Status)
}

func TestRepeatedReconciliationOfACompletedLegacyRestoreWritesConditions(t *testing.T) {
	restore := &backupv1.Restore{
		ObjectMeta: metav1.ObjectMeta{Name: testRestore, Namespace: testNamespace},
		Status:     backupv1.RestoreStatus{Status: backupv1.RestoreStatusCompleted},
	}

	writes := &clientWrites{}
	reconciler := &restoreReconciler{
		namespace: testNamespace,
		k8sClient: newTestClient(t, writes.interceptor(), restore),
	}

	request := ctrl.Request{NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testRestore}}

	for range 2 {
		result, err := reconciler.Reconcile(testCtx, request)
		require.NoError(t, err)
		require.Equal(t, ctrl.Result{}, result)
	}

	require.Equal(t, 1, writes.parent.statusUpdates, "the migration must be persisted exactly once")
	stored := &backupv1.Restore{}
	require.NoError(t, reconciler.k8sClient.Get(testCtx, client.ObjectKey{Namespace: testNamespace, Name: testRestore}, stored))
	require.Equal(t, backupv1.RestoreStatusCompleted, stored.Status.Status)
	require.Equal(t, metav1.ConditionTrue, findSuccessfulCondition(stored).Status)
	require.Equal(t, ReasonMigratedFromLegacyStatus, findSuccessfulCondition(stored).Reason)
}

func TestAParentNotFoundRacePerformsNoWritesOrExternalActions(t *testing.T) {
	writes := &clientWrites{}
	reconciler := &restoreReconciler{
		namespace: testNamespace,
		k8sClient: newTestClient(t, writes.interceptor()),
	}
	request := ctrl.Request{NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testRestore}}

	for range 2 {
		result, err := reconciler.Reconcile(testCtx, request)
		require.NoError(t, err)
		require.Equal(t, ctrl.Result{}, result)
	}

	require.Equal(t, 0, writes.total())
}
