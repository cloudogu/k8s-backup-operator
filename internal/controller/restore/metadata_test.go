package restore

import (
	"testing"

	k8sv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

// The metadata stage writes the finalizer and the labels in one update and ends the reconciliation,
// so the following reconciliation is the one that starts the restore.
func TestEnsureMetadataWritesTheParentOnceAndThenLetsTheWorkflowStart(t *testing.T) {
	restore := withPreparation(withInitializedConditions(newParentRestore()))

	managerMock := newMockRestoreManager(t)
	managerMock.EXPECT().create(testCtx, matchesRestoreNamed(testRestore)).Return(nil).Once()
	recorderMock := newMockEventRecorder(t)
	recorderMock.EXPECT().Event(matchesRestoreNamed(testRestore), corev1.EventTypeNormal, k8sv1.CreateEventReason, "Creation successful").Return()

	factory := func(fakeClient client.WithWatch) reconcileFunction {
		return NewRestoreReconciler(fakeClient, recorderMock, testNamespace, managerMock, nil, nil).Reconcile
	}
	fixture := newMultiReconcileFixture(t, interceptor.Funcs{}, factory, restore)
	request := newRestoreRequest(testRestore)

	results, errs := fixture.reconcileTimes(testCtx, request, 2)

	require.NoError(t, errs[0])
	require.NoError(t, errs[1])
	assert.Equal(t, ctrl.Result{RequeueAfter: defaultRequeueDelay}, results[0], "the metadata write must end the reconciliation")
	assert.Equal(t, ctrl.Result{}, results[1])
	assert.Equal(t, []recordedClientAction{updateOf(restore)}, fixture.clientActions.snapshot(),
		"the finalizer and the labels must be written in one update, and only once")
	assertPersistedMetadata(t, fixture.client, testRestore)
}

func TestEnsureMetadataDoesNotWriteAgain(t *testing.T) {
	restore := withPreparation(withInitializedConditions(withMetadata(newParentRestore())))

	managerMock := newMockRestoreManager(t)
	managerMock.EXPECT().create(testCtx, matchesRestoreNamed(testRestore)).Return(nil).Once()
	recorderMock := newMockEventRecorder(t)
	recorderMock.EXPECT().Event(matchesRestoreNamed(testRestore), corev1.EventTypeNormal, k8sv1.CreateEventReason, "Creation successful").Return()

	factory := func(fakeClient client.WithWatch) reconcileFunction {
		return NewRestoreReconciler(fakeClient, recorderMock, testNamespace, managerMock, nil, nil).Reconcile
	}
	fixture := newMultiReconcileFixture(t, interceptor.Funcs{}, factory, restore)

	results, errs := fixture.reconcileTimes(testCtx, newRestoreRequest(testRestore), 1)

	require.NoError(t, errs[0])
	assert.Equal(t, ctrl.Result{}, results[0], "a converged restore must reach the create operation right away")
	assert.Empty(t, fixture.clientActions.snapshot(), "a converged restore must not be written")
}

// The manager is nil, so a failing metadata write must not reach the create operation.
func TestEnsureMetadataRetriesAFailedWriteWithoutStartingTheRestore(t *testing.T) {
	restore := withPreparation(withInitializedConditions(newParentRestore()))

	factory := func(fakeClient client.WithWatch) reconcileFunction {
		return NewRestoreReconciler(fakeClient, nil, testNamespace, nil, nil, nil).Reconcile
	}
	fixture := newMultiReconcileFixture(t, failingUpdate(assert.AnError), factory, restore)

	results, errs := fixture.reconcileTimes(testCtx, newRestoreRequest(testRestore), 1)

	require.ErrorIs(t, errs[0], assert.AnError)
	assert.ErrorContains(t, errs[0], "failed to converge the metadata of restore test-restore")
	assert.Equal(t, ctrl.Result{}, results[0], "an error carries the retry, so there must be no explicit requeue")
}
