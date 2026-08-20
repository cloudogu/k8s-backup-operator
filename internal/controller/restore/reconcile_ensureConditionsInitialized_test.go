package restore

import (
	"testing"

	k8sv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/k8s-backup-operator/internal/provider/velero"
	"github.com/cloudogu/k8s-registry-lib/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

// assertUnknownWorkflowConditions asserts that the given condition types are persisted as pending.
func assertUnknownWorkflowConditions(t *testing.T, stored *k8sv1.Restore, conditionTypes ...string) {
	t.Helper()

	for _, conditionType := range conditionTypes {
		condition := meta.FindStatusCondition(stored.Status.Conditions, conditionType)
		require.NotNil(t, condition, "condition %s was not initialized", conditionType)
		assert.Equal(t, metav1.ConditionUnknown, condition.Status, "condition %s", conditionType)
		assert.Equal(t, ReasonPending, condition.Reason, "condition %s", conditionType)
	}
}

func TestTheFirstReconcileOfANewRestoreInitializesAllWorkflowConditions(t *testing.T) {
	restore := newParentRestore()

	factory := func(fakeClient client.WithWatch) reconcileFunction {
		// no manager and no recorder: reaching the create operation would panic
		return NewRestoreReconciler(fakeClient, nil, testNamespace, nil, nil, requeueAfterTest).Reconcile
	}
	fixture := newMultiReconcileFixture(t, interceptor.Funcs{}, factory, restore)
	request := newRestoreRequest(testRestore)

	results, errs := fixture.reconcileTimes(testCtx, request, 2)

	require.NoError(t, errs[0])
	require.NoError(t, errs[1])
	assert.Equal(t, ctrl.Result{RequeueAfter: defaultRequeueDelay}, results[0], "the initializing write must end the reconciliation")
	assert.Equal(t, ctrl.Result{RequeueAfter: defaultRequeueDelay}, results[1], "the second reconciliation belongs to the metadata stage")
	assert.Equal(t, []recordedClientAction{statusUpdateOf(restore), updateOf(restore)}, fixture.clientActions.snapshot(),
		"the conditions must be initialized in one write, before the metadata and only once")

	stored := &k8sv1.Restore{}
	require.NoError(t, fixture.client.Get(testCtx, request.NamespacedName, stored))
	assertUnknownWorkflowConditions(t, stored, workflowConditionTypes...)
	assert.Equal(t, k8sv1.RestoreStatusInProgress, stored.Status.Status,
		"the deprecated status phase is derived from the initialized conditions")
}

func TestConditionInitializationDoesNotResetAResolvedCondition(t *testing.T) {
	restore := withMetadata(newParentRestore())
	restore.Status.Conditions = []metav1.Condition{{
		Type:               k8sv1.ConditionPrepared,
		Status:             metav1.ConditionTrue,
		Reason:             ReasonRestoreCompleted,
		LastTransitionTime: metav1.Now(),
	}}

	factory := func(fakeClient client.WithWatch) reconcileFunction {
		return NewRestoreReconciler(fakeClient, nil, testNamespace, nil, nil, requeueAfterTest).Reconcile
	}
	fixture := newMultiReconcileFixture(t, interceptor.Funcs{}, factory, restore)
	request := newRestoreRequest(testRestore)

	results, errs := fixture.reconcileTimes(testCtx, request, 1)

	require.NoError(t, errs[0])
	assert.Equal(t, ctrl.Result{RequeueAfter: defaultRequeueDelay}, results[0])

	stored := &k8sv1.Restore{}
	require.NoError(t, fixture.client.Get(testCtx, request.NamespacedName, stored))
	prepared := meta.FindStatusCondition(stored.Status.Conditions, k8sv1.ConditionPrepared)
	require.NotNil(t, prepared)
	assert.Equal(t, metav1.ConditionTrue, prepared.Status, "the reached milestone must survive the initialization")
	assertUnknownWorkflowConditions(t, stored, k8sv1.ConditionSuccessful, k8sv1.ConditionProviderRestoreSuccessful,
		k8sv1.ConditionWorkloadsRecovered, k8sv1.ConditionBackupsSynchronized)
}

func TestARestoreInterruptedAtAnUnknownOutcomeContinues(t *testing.T) {
	restore := withPreparation(withInitializedConditions(withMetadata(newParentRestore())))

	recorderMock := newMockEventRecorder(t)
	recorderMock.EXPECT().Event(matchesRestoreNamed(testRestore), corev1.EventTypeNormal, k8sv1.CreateEventReason, "Start restore process").Return()

	maintenanceMock := newMockMaintenanceModeSwitch(t)
	maintenanceMock.EXPECT().GetStatus(testCtx).Return(repository.MaintenanceModeDescription{}, true, nil)

	factory := func(fakeClient client.WithWatch) reconcileFunction {
		reconciler := NewRestoreReconciler(fakeClient, recorderMock, testNamespace, nil, nil, requeueAfterTest)
		reconciler.maintenanceModeSwitch = maintenanceMock
		return reconciler.Reconcile
	}
	fixture := newMultiReconcileFixture(t, interceptor.Funcs{}, factory, restore)

	results, errs := fixture.reconcileTimes(testCtx, newRestoreRequest(testRestore), 1)

	require.NoError(t, errs[0])
	assert.Equal(t, ctrl.Result{RequeueAfter: defaultRequeueDelay}, results[0])
	assert.Equal(t, []recordedClientAction{createOf(velero.BuildRestore(restore))}, fixture.clientActions.snapshot(),
		"the restore must continue with the provider restore instead of being written again")
}

func TestALegacyRestoreWithAnUninterpretableStatusIsInitializedInstead(t *testing.T) {
	restore := newParentRestore()
	restore.Status.Status = "some-unknown-status"

	factory := func(fakeClient client.WithWatch) reconcileFunction {
		return NewRestoreReconciler(fakeClient, nil, testNamespace, nil, nil, requeueAfterTest).Reconcile
	}
	fixture := newMultiReconcileFixture(t, interceptor.Funcs{}, factory, restore)
	request := newRestoreRequest(testRestore)

	results, errs := fixture.reconcileTimes(testCtx, request, 1)

	require.NoError(t, errs[0])
	assert.Equal(t, ctrl.Result{RequeueAfter: defaultRequeueDelay}, results[0])

	stored := &k8sv1.Restore{}
	require.NoError(t, fixture.client.Get(testCtx, request.NamespacedName, stored))
	assertUnknownWorkflowConditions(t, stored, workflowConditionTypes...)
}

func TestAFailedConditionInitializationIsRetriedWithoutStartingTheRestore(t *testing.T) {
	restore := newParentRestore()

	factory := func(fakeClient client.WithWatch) reconcileFunction {
		// The manager is nil, so a failing initialization must not reach any later stage.
		return NewRestoreReconciler(fakeClient, nil, testNamespace, nil, nil, requeueAfterTest).Reconcile
	}
	fixture := newMultiReconcileFixture(t, failingStatusUpdate(assert.AnError), factory, restore)

	results, errs := fixture.reconcileTimes(testCtx, newRestoreRequest(testRestore), 1)

	require.ErrorIs(t, errs[0], assert.AnError)
	assert.ErrorContains(t, errs[0], "failed to initialize the conditions of restore test-restore")
	assert.Equal(t, ctrl.Result{}, results[0], "an error carries the retry, so there must be no explicit requeue")
	assert.Equal(t, []recordedClientAction{statusUpdateOf(restore)}, fixture.clientActions.snapshot(),
		"the metadata stage must not run before the conditions are initialized")
}
