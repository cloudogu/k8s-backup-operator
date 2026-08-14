package restore

import (
	"context"
	"testing"

	k8sv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/k8s-backup-operator/internal/provider/velero"
	"github.com/cloudogu/k8s-registry-lib/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

// ownedChildInPhase returns the provider restore of the given Restore in the given phase.
func ownedChildInPhase(parent *k8sv1.Restore, phase velerov1.RestorePhase) *velerov1.Restore {
	child := velero.BuildRestore(parent)
	child.Status.Phase = phase

	return child
}

func TestAnUndecidedProviderRestoreEndsTheReconciliationWithoutAnOutcome(t *testing.T) {
	tests := []struct {
		name   string
		phase  velerov1.RestorePhase
		reason string
	}{
		{name: "not started yet", phase: velerov1.RestorePhaseNew, reason: ReasonProviderRestorePending},
		{name: "executing", phase: velerov1.RestorePhaseInProgress, reason: ReasonProviderRestoreRunning},
		{name: "finalizing after a partial failure", phase: velerov1.RestorePhaseFinalizingPartiallyFailed, reason: ReasonProviderRestoreRunning},
		{name: "a phase a later provider version added", phase: velerov1.RestorePhase("SomethingNew"), reason: ReasonProviderRestoreStateUnknown},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			restore := startableRestore()

			factory := func(fakeClient client.WithWatch) reconcileFunction {
				return NewRestoreReconciler(fakeClient, nil, testNamespace, nil, nil, requeueAfterTest).Reconcile
			}
			fixture := newMultiReconcileFixture(t, interceptor.Funcs{}, factory, restore, ownedChildInPhase(restore, test.phase))

			results, errs := fixture.reconcileTimes(testCtx, newRestoreRequest(testRestore), 1)

			require.NoError(t, errs[0])
			assert.Equal(t, ctrl.Result{RequeueAfter: providerObservationRecoveryDelay}, results[0],
				"the provider is observed through its child's events, not by waiting")
			assertPersistedCondition(t, fixture.client, k8sv1.ConditionProviderRestoreSuccessful, metav1.ConditionUnknown, test.reason)

			stored := &k8sv1.Restore{}
			require.NoError(t, fixture.client.Get(testCtx, newRestoreRequest(testRestore).NamespacedName, stored))
			assert.False(t, isTerminal(stored), "an undecided provider restore must not end the restore")
		})
	}
}

func TestACompletedProviderRestoreResolvesItsMilestoneAndThenContinuesTheWorkflow(t *testing.T) {
	restore := startableRestore()

	// The next stage after the resolved milestone is the backup synchronization.
	expectBackupSynchronization(t, nil)

	recorderMock := newMockEventRecorder(t)
	recorderMock.EXPECT().Eventf(matchesRestoreNamed(testRestore), corev1.EventTypeNormal, k8sv1.CreateEventReason,
		"Successfully completed the provider restore [%s]", testRestore).Return()
	maintenanceMock := newMockMaintenanceModeSwitch(t)
	maintenanceMock.EXPECT().GetStatus(testCtx).Return(repository.MaintenanceModeDescription{}, true, nil)

	factory := func(fakeClient client.WithWatch) reconcileFunction {
		reconciler := NewRestoreReconciler(fakeClient, recorderMock, testNamespace, nil, nil, requeueAfterTest)
		reconciler.maintenanceModeSwitch = maintenanceMock
		// no scale manager - the workflow must not reach the recovery in these two reconciliations
		return reconciler.Reconcile
	}
	fixture := newMultiReconcileFixture(t, interceptor.Funcs{}, factory, restore,
		ownedChildInPhase(restore, velerov1.RestorePhaseCompleted))

	results, errs := fixture.reconcileTimes(testCtx, newRestoreRequest(testRestore), 2)

	require.NoError(t, errs[0])
	require.NoError(t, errs[1])
	assert.Equal(t, ctrl.Result{RequeueAfter: defaultRequeueDelay}, results[0], "the resolved milestone must end the reconciliation")
	assert.Equal(t, ctrl.Result{RequeueAfter: defaultRequeueDelay}, results[1], "the workflow continues once the milestone is stored")
	assert.Equal(t, []recordedClientAction{statusUpdateOf(restore), statusUpdateOf(restore)}, fixture.clientActions.snapshot(),
		"one milestone per reconciliation, and the child must never be written")
	assertPersistedCondition(t, fixture.client, k8sv1.ConditionProviderRestoreSuccessful, metav1.ConditionTrue, ReasonProviderRestoreCompleted)
	assertPersistedCondition(t, fixture.client, k8sv1.ConditionBackupsSynchronized, metav1.ConditionTrue, ReasonBackupSynchronizationCompleted)
}

func TestAFailedProviderRestoreIsTerminalWithoutRecoveringTheWorkloads(t *testing.T) {
	for _, phase := range []velerov1.RestorePhase{
		velerov1.RestorePhaseFailed,
		velerov1.RestorePhaseFailedValidation,
		velerov1.RestorePhasePartiallyFailed,
	} {
		t.Run(string(phase), func(t *testing.T) {
			restore := startableRestore()

			recorderMock := newMockEventRecorder(t)
			recorderMock.EXPECT().Event(matchesRestoreNamed(testRestore), corev1.EventTypeWarning, k8sv1.ErrorOnCreateEventReason, mock.Anything).Return()
			maintenanceMock := newMockMaintenanceModeSwitch(t)
			maintenanceMock.EXPECT().GetStatus(testCtx).Return(repository.MaintenanceModeDescription{}, true, nil)
			maintenanceMock.EXPECT().Deactivate(testCtx, false).Return(nil).Once()

			factory := func(fakeClient client.WithWatch) reconcileFunction {
				reconciler := NewRestoreReconciler(fakeClient, recorderMock, testNamespace, nil, newMockScaleManager(t), requeueAfterTest)
				reconciler.maintenanceModeSwitch = maintenanceMock

				return reconciler.Reconcile
			}
			fixture := newMultiReconcileFixture(t, interceptor.Funcs{}, factory, restore, ownedChildInPhase(restore, phase))

			results, errs := fixture.reconcileTimes(testCtx, newRestoreRequest(testRestore), 2)

			require.NoError(t, errs[0])
			require.NoError(t, errs[1])
			assert.Equal(t, ctrl.Result{}, results[0], "a terminal failure must not be requeued")
			assert.Equal(t, ctrl.Result{}, results[1])
			assert.Equal(t, []recordedClientAction{statusUpdateOf(restore), deleteOf(newRestoreLease(restore))}, fixture.clientActions.snapshot(),
				"the failure must be reported exactly once, in one write")

			assertPersistedCondition(t, fixture.client, k8sv1.ConditionProviderRestoreSuccessful, metav1.ConditionFalse, ReasonProviderRestoreFailed)
			assertPersistedCondition(t, fixture.client, k8sv1.ConditionWorkloadsRecovered, metav1.ConditionFalse, ReasonRecoveryNotAttemptedAfterProviderFailure)
			assertPersistedCondition(t, fixture.client, k8sv1.ConditionBackupsSynchronized, metav1.ConditionFalse, ReasonSynchronizationNotAttemptedAfterProviderFailure)
			assertPersistedCondition(t, fixture.client, k8sv1.ConditionSuccessful, metav1.ConditionFalse, ReasonProviderRestoreFailed)
			assertPersistedCondition(t, fixture.client, k8sv1.ConditionPrepared, metav1.ConditionTrue, ReasonPreparationCompleted)
		})
	}
}

func TestAFailedProviderRestoreIsReportedEvenWhenTheMaintenanceModeStaysOn(t *testing.T) {
	restore := startableRestore()

	recorderMock := newMockEventRecorder(t)
	recorderMock.EXPECT().Event(matchesRestoreNamed(testRestore), corev1.EventTypeWarning, k8sv1.ErrorOnCreateEventReason, mock.Anything).Return()
	maintenanceMock := newMockMaintenanceModeSwitch(t)
	maintenanceMock.EXPECT().GetStatus(testCtx).Return(repository.MaintenanceModeDescription{}, true, nil)
	maintenanceMock.EXPECT().Deactivate(testCtx, false).Return(assert.AnError).Once()

	factory := func(fakeClient client.WithWatch) reconcileFunction {
		reconciler := NewRestoreReconciler(fakeClient, recorderMock, testNamespace, nil, nil, requeueAfterTest)
		reconciler.maintenanceModeSwitch = maintenanceMock

		return reconciler.Reconcile
	}
	fixture := newMultiReconcileFixture(t, interceptor.Funcs{}, factory, restore,
		ownedChildInPhase(restore, velerov1.RestorePhaseFailed))

	results, errs := fixture.reconcileTimes(testCtx, newRestoreRequest(testRestore), 1)

	require.NoError(t, errs[0])
	assert.Equal(t, ctrl.Result{}, results[0])
	assertPersistedCondition(t, fixture.client, k8sv1.ConditionSuccessful, metav1.ConditionFalse, ReasonProviderRestoreFailed)
}

func TestAnUnpersistableProviderRestoreStateIsRetried(t *testing.T) {
	restore := startableRestore()

	factory := func(fakeClient client.WithWatch) reconcileFunction {
		return NewRestoreReconciler(fakeClient, nil, testNamespace, nil, nil, requeueAfterTest).Reconcile
	}
	fixture := newMultiReconcileFixture(t, failingStatusUpdate(assert.AnError), factory, restore,
		ownedChildInPhase(restore, velerov1.RestorePhaseInProgress))

	results, errs := fixture.reconcileTimes(testCtx, newRestoreRequest(testRestore), 1)

	require.ErrorIs(t, errs[0], assert.AnError)
	assert.ErrorContains(t, errs[0], "failed to persist the provider restore state of restore test-restore")
	assert.Equal(t, ctrl.Result{}, results[0], "an error carries the retry, so there must be no explicit requeue")
}

func TestAnUnreportableProviderRestoreFailureIsRetried(t *testing.T) {
	restore := startableRestore()

	recorderMock := newMockEventRecorder(t)
	recorderMock.EXPECT().Event(matchesRestoreNamed(testRestore), corev1.EventTypeWarning, k8sv1.ErrorOnCreateEventReason, mock.Anything).Return()
	maintenanceMock := newMockMaintenanceModeSwitch(t)
	maintenanceMock.EXPECT().GetStatus(testCtx).Return(repository.MaintenanceModeDescription{}, true, nil)
	maintenanceMock.EXPECT().Deactivate(testCtx, false).Return(nil)

	factory := func(fakeClient client.WithWatch) reconcileFunction {
		reconciler := NewRestoreReconciler(fakeClient, recorderMock, testNamespace, nil, nil, requeueAfterTest)
		reconciler.maintenanceModeSwitch = maintenanceMock

		return reconciler.Reconcile
	}
	fixture := newMultiReconcileFixture(t, failingStatusUpdate(assert.AnError), factory, restore,
		ownedChildInPhase(restore, velerov1.RestorePhaseFailed))

	_, errs := fixture.reconcileTimes(testCtx, newRestoreRequest(testRestore), 1)

	require.ErrorIs(t, errs[0], assert.AnError)
	assert.ErrorContains(t, errs[0], "failed to report the failed provider restore of restore test-restore")
}

// A resolved milestone is the outcome, so the child is not read again — not even when it is gone.
func TestAResolvedProviderRestoreIsNotObservedAgain(t *testing.T) {
	restore := withProviderRestoreSuccess(startableRestore())

	reconciler := &restoreReconciler{namespace: testNamespace, k8sClient: newTestClient(t, interceptor.Funcs{}, restore)}

	updated, outcome := reconciler.ensureProviderCompletion(testCtx, restore)

	assert.Equal(t, next(), outcome)
	assert.Equal(t, restore, updated)
}

// The barrier stage reads the child first, so an unreadable child stops the reconciliation before this
// stage. It must report the failure rather than treat the unknown state as an outcome.
func TestAnUnreadableChildStopsTheObservation(t *testing.T) {
	restore := startableRestore()

	failingChildGet := interceptor.Funcs{
		Get: func(ctx context.Context, wrapped client.WithWatch, key client.ObjectKey, object client.Object, opts ...client.GetOption) error {
			if _, isChild := object.(*velerov1.Restore); isChild {
				return assert.AnError
			}

			return wrapped.Get(ctx, key, object, opts...)
		},
	}
	reconciler := &restoreReconciler{namespace: testNamespace, k8sClient: newTestClient(t, failingChildGet, restore)}

	updated, outcome := reconciler.ensureProviderCompletion(testCtx, restore)

	require.ErrorIs(t, outcome.err, assert.AnError)
	assert.ErrorContains(t, outcome.err, "failed to read the provider restore of restore test-restore")
	assert.False(t, isTerminal(updated), "an unreadable child is not an outcome")
}

// The stage that starts the restore runs first, so this is only reachable when the child disappears
// between the two stages. The restore must then be startable again instead of being reported failed.
func TestAVanishedChildLeavesTheRestoreResumable(t *testing.T) {
	restore := startableRestore()

	reconciler := &restoreReconciler{namespace: testNamespace, k8sClient: newTestClient(t, interceptor.Funcs{}, restore), requeueDelay: requeueAfterTest}

	updated, outcome := reconciler.ensureProviderCompletion(testCtx, restore)

	assert.Equal(t, retryAfter(defaultRequeueDelay), outcome)
	assert.False(t, isTerminal(updated), "a missing child is not an outcome")
}
