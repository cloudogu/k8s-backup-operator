package restore

import (
	"testing"

	k8sv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

// recoverableRestore is a Restore whose backups are synchronized, so the only stage left is the
// workload recovery.
func recoverableRestore() *k8sv1.Restore {
	return withBackupsSynchronized(synchronizableRestore())
}

// expectNoBackupSynchronization installs a provider that must not be asked to synchronize anything,
// so that a recovery test cannot silently repeat the preceding stage.
func expectNoBackupSynchronization(t *testing.T) {
	installProvider(t, newMockRestoreProvider(t))
}

func TestWorkloadRecoveryScalesUpSwitchesOffTheMaintenanceModeAndCompletesTheRestore(t *testing.T) {
	restore := recoverableRestore()

	expectNoBackupSynchronization(t)

	scaleMock := newMockScaleManager(t)
	scaleMock.EXPECT().ScaleUp(testCtx).Return(nil).Once()
	maintenanceMock := newMockMaintenanceModeSwitch(t)
	maintenanceMock.EXPECT().Deactivate(testCtx, false).Return(nil).Once()
	recorderMock := newMockEventRecorder(t)
	recorderMock.EXPECT().Event(matchesRestoreNamed(testRestore), corev1.EventTypeNormal, k8sv1.CreateEventReason, "Restore successful").Return()

	factory := func(fakeClient client.WithWatch) reconcileFunction {
		reconciler := NewRestoreReconciler(fakeClient, recorderMock, testNamespace, nil, nil, scaleMock)
		reconciler.maintenanceModeSwitch = maintenanceMock

		return reconciler.Reconcile
	}
	fixture := newMultiReconcileFixture(t, interceptor.Funcs{}, factory, restore, ownedChildInPhase(restore, velerov1.RestorePhaseCompleted))

	results, errs := fixture.reconcileTimes(testCtx, newRestoreRequest(testRestore), 2)

	require.NoError(t, errs[0])
	require.NoError(t, errs[1])
	assert.Equal(t, ctrl.Result{}, results[0], "the finished workflow must not be requeued")
	assert.Equal(t, ctrl.Result{}, results[1], "a completed restore must be reconciled without doing anything")
	assert.Equal(t, []recordedClientAction{statusUpdateOf(restore)}, fixture.clientActions.snapshot(),
		"the recovery must write the two milestones it reaches together, once")
	assertPersistedCondition(t, fixture.client, k8sv1.ConditionWorkloadsRecovered, metav1.ConditionTrue, ReasonWorkloadRecoveryCompleted)
	assertSuccessfulCondition(t, fixture.client, restore.Name, metav1.ConditionTrue, ReasonRestoreCompleted)

	stored := &k8sv1.Restore{}
	require.NoError(t, fixture.client.Get(testCtx, newRestoreRequest(testRestore).NamespacedName, stored))
	assert.Equal(t, k8sv1.RestoreStatusCompleted, stored.Status.Status, "the deprecated scalar status must be derived from the conditions")
}

func TestAFailedScaleUpReportsWorkloadsRecoveredFalseAndRetriesWithoutCompletingTheRestore(t *testing.T) {
	restore := recoverableRestore()

	expectNoBackupSynchronization(t)

	scaleMock := newMockScaleManager(t)
	scaleMock.EXPECT().ScaleUp(testCtx).Return(assert.AnError).Once()
	// no maintenance mode deactivation
	maintenanceMock := newMockMaintenanceModeSwitch(t)
	recorderMock := newMockEventRecorder(t)
	recorderMock.EXPECT().Event(matchesRestoreNamed(testRestore), corev1.EventTypeWarning, k8sv1.ErrorOnCreateEventReason,
		"failed to scale up workloads after restore: assert.AnError general error for testing").Return()

	factory := func(fakeClient client.WithWatch) reconcileFunction {
		reconciler := NewRestoreReconciler(fakeClient, recorderMock, testNamespace, nil, nil, scaleMock)
		reconciler.maintenanceModeSwitch = maintenanceMock

		return reconciler.Reconcile
	}
	fixture := newMultiReconcileFixture(t, interceptor.Funcs{}, factory, restore, ownedChildInPhase(restore, velerov1.RestorePhaseCompleted))

	results, errs := fixture.reconcileTimes(testCtx, newRestoreRequest(testRestore), 1)

	require.Error(t, errs[0])
	assert.ErrorIs(t, errs[0], assert.AnError)
	assert.ErrorContains(t, errs[0], "failed to scale up workloads after restore")
	assert.Equal(t, ctrl.Result{}, results[0], "the controller-runtime backoff decides when a transient failure is retried")
	assertPersistedCondition(t, fixture.client, k8sv1.ConditionWorkloadsRecovered, metav1.ConditionFalse, ReasonWorkloadRecoveryFailed)
	assertPersistedCondition(t, fixture.client, k8sv1.ConditionSuccessful, metav1.ConditionUnknown, ReasonPending)
}

func TestAFailedMaintenanceModeDeactivationIsRetriedWithoutCompletingTheRestore(t *testing.T) {
	restore := recoverableRestore()

	expectNoBackupSynchronization(t)

	scaleMock := newMockScaleManager(t)
	scaleMock.EXPECT().ScaleUp(testCtx).Return(nil).Once()
	maintenanceMock := newMockMaintenanceModeSwitch(t)
	maintenanceMock.EXPECT().Deactivate(testCtx, false).Return(assert.AnError).Once()

	factory := func(fakeClient client.WithWatch) reconcileFunction {
		// no recorder: a restore that has not switched off maintenance mode must not emit a success event
		reconciler := NewRestoreReconciler(fakeClient, nil, testNamespace, nil, nil, scaleMock)
		reconciler.maintenanceModeSwitch = maintenanceMock

		return reconciler.Reconcile
	}
	fixture := newMultiReconcileFixture(t, interceptor.Funcs{}, factory, restore, ownedChildInPhase(restore, velerov1.RestorePhaseCompleted))

	results, errs := fixture.reconcileTimes(testCtx, newRestoreRequest(testRestore), 1)

	require.Error(t, errs[0])
	assert.ErrorIs(t, errs[0], assert.AnError)
	assert.ErrorContains(t, errs[0], "maintenance mode could not be deactivated after the restore test-restore")
	assert.Equal(t, ctrl.Result{}, results[0], "the controller-runtime backoff decides when the deactivation is retried")
	assert.Empty(t, fixture.clientActions.snapshot(), "the failed deactivation must not persist a successful outcome")
	assertPersistedCondition(t, fixture.client, k8sv1.ConditionWorkloadsRecovered, metav1.ConditionUnknown, ReasonPending)
	assertSuccessfulCondition(t, fixture.client, restore.Name, metav1.ConditionUnknown, ReasonPending)
}

func TestAnUnpersistableRestoreOutcomeIsRetried(t *testing.T) {
	restore := recoverableRestore()

	expectNoBackupSynchronization(t)

	scaleMock := newMockScaleManager(t)
	scaleMock.EXPECT().ScaleUp(testCtx).Return(nil)
	maintenanceMock := newMockMaintenanceModeSwitch(t)
	maintenanceMock.EXPECT().Deactivate(testCtx, false).Return(nil)

	factory := func(fakeClient client.WithWatch) reconcileFunction {
		// no recorder - the success event must not be emitted for an outcome that was not persisted
		reconciler := NewRestoreReconciler(fakeClient, nil, testNamespace, nil, nil, scaleMock)
		reconciler.maintenanceModeSwitch = maintenanceMock

		return reconciler.Reconcile
	}
	fixture := newMultiReconcileFixture(t, failingStatusUpdate(assert.AnError), factory, restore,
		ownedChildInPhase(restore, velerov1.RestorePhaseCompleted))

	_, errs := fixture.reconcileTimes(testCtx, newRestoreRequest(testRestore), 1)

	require.Error(t, errs[0])
	assert.ErrorIs(t, errs[0], assert.AnError)
	assert.ErrorContains(t, errs[0], "failed to persist the workload recovery of restore test-restore")
}
