package restore

import (
	"testing"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/k8s-backup-operator/internal/provider/velero"
	"github.com/stretchr/testify/require"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

// newReadyRestore returns a Restore that has completed its workflow, with the finalizer and the
// labels the create flow applies, so that a stage which converges metadata is a no-op as well.
func newReadyRestore() *backupv1.Restore {
	restore := newParentRestore()
	restore.Finalizers = []string{backupv1.RestoreFinalizer}
	restore.Labels = restoreLabels()
	restore.Status = backupv1.RestoreStatus{
		Status: backupv1.RestoreStatusCompleted,
		Conditions: []metav1.Condition{{
			Type:               backupv1.ConditionSuccessful,
			Status:             metav1.ConditionTrue,
			Reason:             ReasonRestoreCompleted,
			LastTransitionTime: metav1.Now(),
		}},
	}

	return restore
}

// reconcilerWithoutExternals builds the reconciler under test without a manager or a recorder, so an
// external action panics instead of being merely unexpected.
func reconcilerWithoutExternals(fakeClient client.WithWatch) reconcileFunction {
	return NewRestoreReconciler(fakeClient, nil, testNamespace, nil, nil, nil).Reconcile
}

func TestRepeatedReconciliationOfAReadyRestoreWithACompletedChildPerformsNoWritesOrExternalActions(t *testing.T) {
	restore := newReadyRestore()
	child := velero.BuildRestore(restore)
	child.Status.Phase = velerov1.RestorePhaseCompleted

	fixture := newMultiReconcileFixture(t, interceptor.Funcs{}, reconcilerWithoutExternals, restore, child)
	request := newRestoreRequest(testRestore)

	results, errs := fixture.reconcileTimes(testCtx, request, 2)
	fixture.restart(reconcilerWithoutExternals)
	restartResults, restartErrs := fixture.reconcileTimes(testCtx, request, 1)

	for _, err := range append(errs, restartErrs...) {
		require.NoError(t, err)
	}
	for _, result := range append(results, restartResults...) {
		require.Equal(t, ctrl.Result{}, result)
	}

	require.Empty(t, fixture.clientActions.snapshot(), "a terminal restore must be reconciled without any write")
	stored := &backupv1.Restore{}
	require.NoError(t, fixture.client.Get(testCtx, request.NamespacedName, stored))
	require.Equal(t, backupv1.RestoreStatusCompleted, stored.Status.Status)
	require.Equal(t, metav1.ConditionTrue, findSuccessfulCondition(stored).Status)
	require.Equal(t, ReasonRestoreCompleted, findSuccessfulCondition(stored).Reason)
}

func TestAConflictingConcurrentStatusWriteDuringReconciliationDropsNoCondition(t *testing.T) {
	legacy := newParentRestore()
	legacy.Status = backupv1.RestoreStatus{Status: backupv1.RestoreStatusCompleted}
	concurrent := metav1.Condition{
		Type:    backupv1.ConditionWorkloadsRecovered,
		Status:  metav1.ConditionTrue,
		Reason:  ReasonRecoveringWorkloads,
		Message: "Written by someone else while the migration was in flight.",
	}

	fixture := newMultiReconcileFixture(t, conflictOnFirstStatusUpdate(t, concurrent), reconcilerWithoutExternals, legacy)
	request := newRestoreRequest(testRestore)

	_, errs := fixture.reconcileTimes(testCtx, request, 1)
	require.NoError(t, errs[0])

	require.Equal(t, []recordedClientAction{statusUpdateOf(legacy), statusUpdateOf(legacy)},
		fixture.clientActions.snapshot(), "the conflicting write and the successful retry")

	stored := &backupv1.Restore{}
	require.NoError(t, fixture.client.Get(testCtx, request.NamespacedName, stored))
	require.Equal(t, metav1.ConditionTrue, findSuccessfulCondition(stored).Status)
	require.Equal(t, ReasonMigratedFromLegacyStatus, findSuccessfulCondition(stored).Reason)
	survivor := meta.FindStatusCondition(stored.Status.Conditions, backupv1.ConditionWorkloadsRecovered)
	require.NotNil(t, survivor, "the concurrently written condition must not be dropped")
	require.Equal(t, concurrent.Message, survivor.Message)
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
