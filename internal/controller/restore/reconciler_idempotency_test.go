package restore

import (
	"testing"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/k8s-backup-operator/internal/provider/velero"
	"github.com/cloudogu/k8s-registry-lib/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	corev1 "k8s.io/api/core/v1"
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
			Type:               backupv1.ConditionSucceeded,
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
	return NewRestoreReconciler(fakeClient, nil, testNamespace, nil, nil, requeueAfterTest).Reconcile
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

// advanceChildTo moves the owned provider restore to the given phase, the way the provider would.
func advanceChildTo(t *testing.T, fixture *multiReconcileFixture, phase velerov1.RestorePhase) {
	t.Helper()

	fixture.simulateExternalWrite(t, func(testClient client.WithWatch) error {
		child := &velerov1.Restore{}
		if err := testClient.Get(testCtx, newRestoreRequest(testRestore).NamespacedName, child); err != nil {
			return err
		}
		child.Status.Phase = phase

		return testClient.Update(testCtx, child)
	})
}

// The whole workflow, one stage per reconciliation, from a fresh Restore to a successful one. No
// reconciliation blocks: the provider's progress arrives as a child phase between two of them, and the
// operator restarts while the provider is still running.
func TestTheWorkflowRunsToSuccessOneStagePerReconciliationWithoutBlocking(t *testing.T) {
	restore := newParentRestore()

	providerMock := newMockRestoreProvider(t)
	providerMock.EXPECT().CheckReady(testCtx).Return(nil)
	installProvider(t, providerMock)

	maintenanceMock := newMockMaintenanceModeSwitch(t)
	maintenanceMock.EXPECT().GetStatus(testCtx).Return(repository.MaintenanceModeDescription{}, false, nil).Once()
	maintenanceMock.EXPECT().GetStatus(testCtx).Return(repository.MaintenanceModeDescription{}, true, nil)
	maintenanceMock.EXPECT().Activate(testCtx, mock.Anything, false).Return(nil).Once()
	maintenanceMock.EXPECT().Deactivate(testCtx, false).Return(nil).Twice()
	cleanupMock := newMockCleanupManager(t)
	cleanupMock.EXPECT().Cleanup(testCtx).Return(nil).Once()
	scaleMock := newMockScaleManager(t)
	scaleMock.EXPECT().ScaleDown(testCtx).Return(nil).Once()
	scaleMock.EXPECT().ScaleUp(testCtx).Return(nil).Times(5)
	scaleMock.EXPECT().AreWorkloadsReady(testCtx).Return(true, nil).Times(4)
	scaleMock.EXPECT().FinalizeScaleUp(testCtx).Return(nil).Times(3)

	recorderMock := newMockEventRecorder(t)
	recorderMock.EXPECT().Eventf(matchesRestoreNamed(testRestore), corev1.EventTypeNormal, backupv1.CreateEventReason,
		"Successfully completed the provider restore [%s]", testRestore).Return()
	recorderMock.EXPECT().Event(matchesRestoreNamed(testRestore), corev1.EventTypeNormal, ReasonRestoreCompleted, "Restore successful").Return()
	recorderMock.EXPECT().Event(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	recorderMock.EXPECT().Eventf(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()

	// The maintenance switch has to be replaced after construction: unlike the cleanup and the scale
	// manager it is not a constructor parameter, so the reconciler builds a real adapter for it.
	factory := func(fakeClient client.WithWatch) reconcileFunction {
		reconciler := NewRestoreReconciler(fakeClient, recorderMock, testNamespace, cleanupMock, scaleMock, requeueAfterTest)
		reconciler.maintenanceModeSwitch = maintenanceMock

		return reconciler.Reconcile
	}
	fixture := newMultiReconcileFixture(t, interceptor.Funcs{}, factory, restore)
	request := newRestoreRequest(testRestore)

	// One reconciliation per stage: conditions, metadata, preparation, start, and the first observation
	// of a provider that has accepted the restore but not started it.
	staged, stagedErrs := fixture.reconcileTimes(testCtx, request, 5)
	for index := range stagedErrs {
		require.NoError(t, stagedErrs[index])
	}
	for index, result := range staged[:4] {
		require.Equal(t, ctrl.Result{RequeueAfter: defaultRequeueDelay}, result, "stage %d must end its reconciliation", index+1)
	}
	require.Equal(t, ctrl.Result{RequeueAfter: providerObservationRecoveryDelay}, staged[4],
		"waiting for the provider must never occupy a worker")

	advanceChildTo(t, fixture, velerov1.RestorePhaseInProgress)
	running, runningErrs := fixture.reconcileTimes(testCtx, request, 1)
	require.NoError(t, runningErrs[0])
	require.Equal(t, ctrl.Result{RequeueAfter: providerObservationRecoveryDelay}, running[0])
	assertPersistedCondition(t, fixture.client, backupv1.ConditionProviderSucceeded, metav1.ConditionUnknown, ReasonProviderRestoreRunning)

	// The operator restarts while the provider is still working, and the provider finishes meanwhile.
	fixture.restart(factory)
	advanceChildTo(t, fixture, velerov1.RestorePhaseCompleted)

	finishing, finishingErrs := fixture.reconcileTimes(testCtx, request, 6)
	for index := range finishingErrs {
		require.NoError(t, finishingErrs[index])
	}
	require.Equal(t, ctrl.Result{RequeueAfter: defaultRequeueDelay}, finishing[0], "the resolved provider milestone ends its reconciliation")
	for index := 1; index < 5; index++ {
		require.Equal(t, ctrl.Result{RequeueAfter: defaultRequeueDelay}, finishing[index], "recovery stage %d must end its reconciliation", index)
	}
	require.Equal(t, ctrl.Result{}, finishing[5], "the finished workflow must not be requeued")

	require.Equal(t, []recordedClientAction{
		statusUpdateOf(restore),                // the initialized conditions
		updateOf(restore),                      // the finalizer and the labels
		statusUpdateOf(restore),                // Prepared
		createOf(velero.BuildRestore(restore)), // the provider restore, started and not awaited
		statusUpdateOf(restore),                // ProviderRestoreSuccessful=Unknown, pending
		statusUpdateOf(restore),                // ProviderRestoreSuccessful=Unknown, running
		statusUpdateOf(restore),                // ProviderRestoreSuccessful=True
		statusUpdateOf(restore),                // ScaleUpInitiated
		statusUpdateOf(restore),                // WorkloadsReady
		statusUpdateOf(restore),                // ScaleUpFinalized
		statusUpdateOf(restore),                // MaintenanceModeDeactivated
		statusUpdateOf(restore),                // WorkloadsRecovered=True and Successful=True
	}, fixture.clientActions.snapshot(), "one write per stage, and the child is created but never written")

	stored := &backupv1.Restore{}
	require.NoError(t, fixture.client.Get(testCtx, request.NamespacedName, stored))
	require.Equal(t, backupv1.RestoreStatusCompleted, stored.Status.Status)
	for _, conditionType := range workflowConditionTypes {
		condition := meta.FindStatusCondition(stored.Status.Conditions, conditionType)
		require.NotNil(t, condition, "condition %s", conditionType)
		require.Equal(t, metav1.ConditionTrue, condition.Status, "condition %s must be resolved", conditionType)
	}

	// The first terminal reconciliation releases the shared lease; subsequent ones are read-only.
	before := len(fixture.clientActions.snapshot())
	settled, settledErrs := fixture.reconcileTimes(testCtx, request, 2)
	for index := range settled {
		require.NoError(t, settledErrs[index])
		require.Equal(t, ctrl.Result{}, settled[index])
	}
	require.Len(t, fixture.clientActions.snapshot(), before+1, "a completed restore must release its lease exactly once")
	require.Equal(t, deleteOf(newRestoreLease(restore)), fixture.clientActions.snapshot()[before])
}

// A crash after the cleanup but before the child creation leaves a Restore whose
// preparation is indistinguishable from one that never started. It may therefore repeat the
// preparation and then has to continue instead of stalling.
func TestARestoreInterruptedBeforeItsChildRepeatsThePreparationAndThenStartsTheProviderRestore(t *testing.T) {
	restore := withInitializedConditions(withMetadata(newParentRestore()))

	providerMock := newMockRestoreProvider(t)
	providerMock.EXPECT().CheckReady(testCtx).Return(nil).Once()
	installProvider(t, providerMock)

	maintenanceMock := newMockMaintenanceModeSwitch(t)
	maintenanceMock.EXPECT().GetStatus(testCtx).Return(repository.MaintenanceModeDescription{}, false, nil).Once()
	maintenanceMock.EXPECT().Activate(testCtx, mock.Anything, false).Return(nil).Once()
	maintenanceMock.EXPECT().GetStatus(testCtx).Return(repository.MaintenanceModeDescription{}, true, nil)
	cleanupMock := newMockCleanupManager(t)
	cleanupMock.EXPECT().Cleanup(testCtx).Return(nil).Once()
	// No ScaleUp expectation: recovering the workloads before the provider ran would fail here.
	scaleMock := newMockScaleManager(t)
	scaleMock.EXPECT().ScaleDown(testCtx).Return(nil).Once()
	recorderMock := newMockEventRecorder(t)
	recorderMock.EXPECT().Event(matchesRestoreNamed(testRestore), corev1.EventTypeNormal, ReasonPreparing, "Preparation in progress - scale down and cleanup").Once()
	recorderMock.EXPECT().Event(matchesRestoreNamed(testRestore), corev1.EventTypeNormal, ReasonCleanupCompleted, "Cleanup before restore completed").Once()
	recorderMock.EXPECT().Event(matchesRestoreNamed(testRestore), corev1.EventTypeNormal, ReasonPreparationCompleted, "Preparation completed").Once()
	recorderMock.EXPECT().Event(matchesRestoreNamed(testRestore), corev1.EventTypeNormal, ReasonProviderRestoreRunning, "Start provider restore process").Once()
	recorderMock.EXPECT().Eventf(matchesRestoreNamed(testRestore), corev1.EventTypeNormal, ReasonMaintenanceModeActivated, "Maintenance mode activated").Once()

	factory := func(fakeClient client.WithWatch) reconcileFunction {
		reconciler := NewRestoreReconciler(fakeClient, recorderMock, testNamespace, cleanupMock, scaleMock, requeueAfterTest)
		reconciler.maintenanceModeSwitch = maintenanceMock

		return reconciler.Reconcile
	}
	fixture := newMultiReconcileFixture(t, interceptor.Funcs{}, factory, restore)
	request := newRestoreRequest(testRestore)

	repeated, repeatedErrs := fixture.reconcileTimes(testCtx, request, 1)
	// The operator crashed before it could create the child, so it starts over from the persisted state.
	fixture.restart(factory)
	continued, continuedErrs := fixture.reconcileTimes(testCtx, request, 1)

	require.NoError(t, repeatedErrs[0])
	require.NoError(t, continuedErrs[0])
	require.Equal(t, ctrl.Result{RequeueAfter: defaultRequeueDelay}, repeated[0], "the repeated preparation ends its reconciliation")
	require.Equal(t, ctrl.Result{RequeueAfter: defaultRequeueDelay}, continued[0], "the started provider restore ends its reconciliation")

	require.Equal(t, []recordedClientAction{
		statusUpdateOf(restore),                // Prepared, reached the second time around
		createOf(velero.BuildRestore(restore)), // and only then the child
	}, fixture.clientActions.snapshot(), "the repeated preparation must be persisted before the child is created")

	assertPersistedCondition(t, fixture.client, backupv1.ConditionPrepared, metav1.ConditionTrue, ReasonPreparationCompleted)
}

// The scale-up is the stage under test because it sits behind every destructive one, so a
// regression is observable as a repeated scale-down, cleanup or child creation.
func TestATransientStageFailureIsRetriedWithoutRepeatingAnEarlierDestructiveStage(t *testing.T) {
	restore := recoverableRestore()

	// No cleanup manager at all, and a scale manager that may only scale up: regressing to the
	// preparation panics or fails on an unexpected call.
	scaleMock := newMockScaleManager(t)
	scaleMock.EXPECT().ScaleUp(testCtx).Return(assert.AnError).Once()
	scaleMock.EXPECT().ScaleUp(testCtx).Return(nil).Times(5)
	scaleMock.EXPECT().AreWorkloadsReady(testCtx).Return(true, nil).Times(4)
	scaleMock.EXPECT().FinalizeScaleUp(testCtx).Return(nil).Times(3)
	maintenanceMock := newMockMaintenanceModeSwitch(t)
	maintenanceMock.EXPECT().GetStatus(testCtx).Return(repository.MaintenanceModeDescription{}, true, nil)
	maintenanceMock.EXPECT().Deactivate(testCtx, false).Return(nil).Twice()
	recorderMock := newMockEventRecorder(t)
	recorderMock.EXPECT().Event(matchesRestoreNamed(testRestore), corev1.EventTypeWarning, ReasonWorkloadRecoveryFailed, "failed to initiate workload scale-up after restore").Once()
	recorderMock.EXPECT().Event(matchesRestoreNamed(testRestore), corev1.EventTypeNormal, ReasonRestoreCompleted, "Restore successful").Once()
	recorderMock.EXPECT().Event(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	recorderMock.EXPECT().Eventf(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()

	factory := func(fakeClient client.WithWatch) reconcileFunction {
		reconciler := NewRestoreReconciler(fakeClient, recorderMock, testNamespace, nil, scaleMock, requeueAfterTest)
		reconciler.maintenanceModeSwitch = maintenanceMock

		return reconciler.Reconcile
	}
	fixture := newMultiReconcileFixture(t, interceptor.Funcs{}, factory, restore,
		ownedChildInPhase(restore, velerov1.RestorePhaseCompleted))
	request := newRestoreRequest(testRestore)

	failed, failedErrs := fixture.reconcileTimes(testCtx, request, 1)
	require.ErrorIs(t, failedErrs[0], assert.AnError)
	require.Equal(t, ctrl.Result{}, failed[0], "the error carries the retry, so there must be no explicit requeue")
	assertPersistedCondition(t, fixture.client, backupv1.ConditionWorkloadsRecovered, metav1.ConditionFalse, ReasonWorkloadRecoveryFailed)
	assertSuccessfulCondition(t, fixture.client, restore.Name, metav1.ConditionUnknown, ReasonPending)

	recovered, recoveredErrs := fixture.reconcileTimes(testCtx, request, 5)
	for index := range recovered {
		require.NoError(t, recoveredErrs[index], "reconciliation %d after the failure", index+1)
	}
	for index := 0; index < 4; index++ {
		require.Equal(t, ctrl.Result{RequeueAfter: defaultRequeueDelay}, recovered[index], "stage %d after retry must end its reconciliation", index+1)
	}
	require.Equal(t, ctrl.Result{}, recovered[4], "the finished workflow must not be requeued")

	require.Equal(t, []recordedClientAction{
		statusUpdateOf(restore), // WorkloadsRecovered=False, retried
		statusUpdateOf(restore), // ScaleUpInitiated
		statusUpdateOf(restore), // WorkloadsReady
		statusUpdateOf(restore), // ScaleUpFinalized
		statusUpdateOf(restore), // MaintenanceModeDeactivated
		statusUpdateOf(restore), // WorkloadsRecovered=True and Successful=True
	}, fixture.clientActions.snapshot(), "only the failed stage was repeated, and the child was never written again")

	assertPersistedCondition(t, fixture.client, backupv1.ConditionPrepared, metav1.ConditionTrue, ReasonPreparationCompleted)
	assertSuccessfulCondition(t, fixture.client, restore.Name, metav1.ConditionTrue, ReasonRestoreCompleted)
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
	successfulCondition := findSuccessfulCondition(stored)
	require.NotNil(t, successfulCondition)
	require.Equal(t, metav1.ConditionTrue, successfulCondition.Status)
	require.Equal(t, ReasonMigratedFromLegacyStatus, successfulCondition.Reason)
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
