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

// synchronizableRestore is a Restore whose provider restore succeeded, so the next stage is the
// backup synchronization.
func synchronizableRestore() *k8sv1.Restore {
	return withProviderRestoreSuccess(startableRestore())
}

func TestBackupSynchronizationPersistsItsMilestoneWithoutRecoveringTheWorkloads(t *testing.T) {
	restore := synchronizableRestore()

	expectBackupSynchronization(t, nil)

	factory := func(fakeClient client.WithWatch) reconcileFunction {
		// no scale manager - recovering the workloads in the same reconciliation would panic
		return NewRestoreReconciler(fakeClient, nil, testNamespace, nil, nil, nil).Reconcile
	}
	fixture := newMultiReconcileFixture(t, interceptor.Funcs{}, factory, restore, ownedChildInPhase(restore, velerov1.RestorePhaseCompleted))

	results, errs := fixture.reconcileTimes(testCtx, newRestoreRequest(testRestore), 1)

	require.NoError(t, errs[0])
	assert.Equal(t, ctrl.Result{RequeueAfter: defaultRequeueDelay}, results[0], "the synchronization must end the reconciliation")
	assert.Equal(t, []recordedClientAction{statusUpdateOf(restore)}, fixture.clientActions.snapshot(),
		"the synchronization must persist its milestone in one status write")
	assertPersistedCondition(t, fixture.client, k8sv1.ConditionBackupsSynchronized, metav1.ConditionTrue, ReasonBackupSynchronizationCompleted)
	assertPersistedCondition(t, fixture.client, k8sv1.ConditionSuccessful, metav1.ConditionUnknown, ReasonPending)
}

func TestARestoreWithSynchronizedBackupsIsNotSynchronizedAgain(t *testing.T) {
	restore := withBackupsSynchronized(synchronizableRestore())

	// The provider mock carries no expectations, so a second synchronization would fail.
	installProvider(t, newMockRestoreProvider(t))

	writes := &clientWrites{}
	reconciler := NewRestoreReconciler(newTestClientWithParent(t, writes.interceptor(), restore), nil, testNamespace, nil, nil, nil)

	updated, outcome := reconciler.ensureBackupsSynchronized(testCtx, restore)

	assert.Equal(t, next(), outcome, "the workflow must continue with the workload recovery")
	assert.Equal(t, restore, updated)
	assert.Equal(t, 0, writes.total(), "a resolved milestone must not be written again")
}

func TestAFailedBackupSynchronizationReportsItsMilestoneFalseAndRetries(t *testing.T) {
	restore := synchronizableRestore()

	expectBackupSynchronization(t, assert.AnError)

	recorderMock := newMockEventRecorder(t)
	recorderMock.EXPECT().Event(matchesRestoreNamed(testRestore), corev1.EventTypeWarning, k8sv1.ErrorOnCreateEventReason,
		"failed to sync backups with provider: assert.AnError general error for testing").Return()

	factory := func(fakeClient client.WithWatch) reconcileFunction {
		// no scale manager - the workloads must not be recovered after a failed synchronization
		return NewRestoreReconciler(fakeClient, recorderMock, testNamespace, nil, nil, nil).Reconcile
	}
	fixture := newMultiReconcileFixture(t, interceptor.Funcs{}, factory, restore, ownedChildInPhase(restore, velerov1.RestorePhaseCompleted))

	results, errs := fixture.reconcileTimes(testCtx, newRestoreRequest(testRestore), 1)

	require.Error(t, errs[0])
	assert.ErrorIs(t, errs[0], assert.AnError)
	assert.ErrorContains(t, errs[0], "failed to sync backups with provider")
	assert.Equal(t, ctrl.Result{}, results[0], "the controller-runtime backoff decides when a transient failure is retried")
	assertPersistedCondition(t, fixture.client, k8sv1.ConditionBackupsSynchronized, metav1.ConditionFalse, ReasonBackupSynchronizationFailed)
	assertPersistedCondition(t, fixture.client, k8sv1.ConditionSuccessful, metav1.ConditionUnknown, ReasonPending)
}

func TestAnUnreachableProviderDefersTheBackupSynchronization(t *testing.T) {
	restore := synchronizableRestore()

	expectReadinessCheck(t, assert.AnError)

	writes := &clientWrites{}
	// no scale manager and no recorder - nothing but the provider lookup may happen
	reconciler := NewRestoreReconciler(newTestClientWithParent(t, writes.interceptor(), restore), nil, testNamespace, nil, nil, nil)

	updated, outcome := reconciler.ensureBackupsSynchronized(testCtx, restore)

	require.Error(t, outcome.err)
	assert.ErrorIs(t, outcome.err, assert.AnError)
	assert.ErrorContains(t, outcome.err, "failed to get restore provider [velero]")
	assert.Equal(t, restore, updated)
	assert.Equal(t, 0, writes.total(), "a provider that cannot be reached is not a milestone the restore failed to reach")
}

func TestAnUnpersistableBackupSynchronizationMilestoneIsRetried(t *testing.T) {
	restore := synchronizableRestore()

	expectBackupSynchronization(t, nil)

	factory := func(fakeClient client.WithWatch) reconcileFunction {
		// no scale manager - the workflow must not continue with an unpersisted milestone
		return NewRestoreReconciler(fakeClient, nil, testNamespace, nil, nil, nil).Reconcile
	}
	fixture := newMultiReconcileFixture(t, failingStatusUpdate(assert.AnError), factory, restore, ownedChildInPhase(restore, velerov1.RestorePhaseCompleted))

	_, errs := fixture.reconcileTimes(testCtx, newRestoreRequest(testRestore), 1)

	require.Error(t, errs[0])
	assert.ErrorIs(t, errs[0], assert.AnError)
	assert.ErrorContains(t, errs[0], "failed to persist the backup synchronization of restore test-restore")
}
