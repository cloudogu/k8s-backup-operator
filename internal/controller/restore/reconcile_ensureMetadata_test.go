package restore

import (
	"testing"

	"github.com/cloudogu/k8s-backup-operator/internal/provider/velero"
	"github.com/cloudogu/k8s-registry-lib/repository"
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

	// The manager is nil, so the workflow must start the provider restore and stop there.
	recorderMock := newMockEventRecorder(t)
	recorderMock.EXPECT().Event(matchesRestoreNamed(testRestore), corev1.EventTypeNormal, ReasonProviderRestoreRunning, "Start provider restore process").Once()

	maintenanceMock := newMockMaintenanceModeSwitch(t)
	maintenanceMock.EXPECT().GetStatus(testCtx).Return(repository.MaintenanceModeDescription{}, true, nil)

	factory := func(fakeClient client.WithWatch) reconcileFunction {
		reconciler := NewRestoreReconciler(fakeClient, recorderMock, testNamespace, nil, nil, requeueAfterTest, testBackupStorage)
		reconciler.maintenanceModeSwitch = maintenanceMock
		return reconciler.Reconcile
	}
	fixture := newMultiReconcileFixture(t, interceptor.Funcs{}, factory, restore)
	request := newRestoreRequest(testRestore)

	results, errs := fixture.reconcileTimes(testCtx, request, 2)

	require.NoError(t, errs[0])
	require.NoError(t, errs[1])
	assert.Equal(t, ctrl.Result{RequeueAfter: requeueAfterTest}, results[0], "the metadata write must end the reconciliation")
	assert.Equal(t, ctrl.Result{RequeueAfter: requeueAfterTest}, results[1], "the started provider restore must end the reconciliation")
	assert.Equal(t, []recordedClientAction{updateOf(restore), createOf(velero.BuildRestore(restore))}, fixture.clientActions.snapshot(),
		"the finalizer and the labels must be written in one update, and only once, before the restore starts")
	assertPersistedMetadata(t, fixture.client, testRestore)
}

func TestEnsureMetadataDoesNotWriteAgain(t *testing.T) {
	restore := withPreparation(withInitializedConditions(withMetadata(newParentRestore())))

	recorderMock := newMockEventRecorder(t)
	recorderMock.EXPECT().Event(matchesRestoreNamed(testRestore), corev1.EventTypeNormal, ReasonProviderRestoreRunning, "Start provider restore process").Once()

	maintenanceMock := newMockMaintenanceModeSwitch(t)
	maintenanceMock.EXPECT().GetStatus(testCtx).Return(repository.MaintenanceModeDescription{}, true, nil)

	factory := func(fakeClient client.WithWatch) reconcileFunction {
		reconciler := NewRestoreReconciler(fakeClient, recorderMock, testNamespace, nil, nil, requeueAfterTest, testBackupStorage)
		reconciler.maintenanceModeSwitch = maintenanceMock
		return reconciler.Reconcile
	}

	fixture := newMultiReconcileFixture(t, interceptor.Funcs{}, factory, restore)

	results, errs := fixture.reconcileTimes(testCtx, newRestoreRequest(testRestore), 1)

	require.NoError(t, errs[0])
	assert.Equal(t, ctrl.Result{RequeueAfter: requeueAfterTest}, results[0], "a converged restore must start the provider restore right away")
	assert.Equal(t, []recordedClientAction{createOf(velero.BuildRestore(restore))}, fixture.clientActions.snapshot(),
		"a restore with metadata must not be written again")
}

// The manager is nil, so a failing metadata write must not reach the create operation.
func TestEnsureMetadataRetriesAFailedWriteWithoutStartingTheRestore(t *testing.T) {
	restore := withPreparation(withInitializedConditions(newParentRestore()))

	factory := func(fakeClient client.WithWatch) reconcileFunction {
		return NewRestoreReconciler(fakeClient, nil, testNamespace, nil, nil, requeueAfterTest, testBackupStorage).Reconcile
	}
	fixture := newMultiReconcileFixture(t, failingUpdate(assert.AnError), factory, restore)

	results, errs := fixture.reconcileTimes(testCtx, newRestoreRequest(testRestore), 1)

	require.ErrorIs(t, errs[0], assert.AnError)
	assert.ErrorContains(t, errs[0], "failed to converge the metadata of restore test-restore")
	assert.Equal(t, ctrl.Result{}, results[0], "an error carries the retry, so there must be no explicit requeue")
}
