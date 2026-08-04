package restore

import (
	"testing"

	k8sv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestEnsureDeletingStatusDoesNotWriteAnAlreadyPersistedDeletingStatusAgain(t *testing.T) {
	restore := deletedRestore()
	restore.Status.Status = k8sv1.RestoreStatusDeleting // NOSONAR -- legacy restore status compatibility
	writes := &clientWrites{}
	reconciler := &restoreReconciler{k8sClient: newTestClientWithParent(t, writes.interceptor(), restore)}

	updated, outcome := reconciler.ensureDeletingStatus(testCtx, restore)

	assert.Equal(t, restore, updated)
	assert.Equal(t, next(), outcome)
	assert.Equal(t, 0, writes.total(), "the converged legacy status must not be written again")
}

func TestEnsureDeletingStatusPersistsTheLegacyStatusAndEndsTheReconciliation(t *testing.T) {
	restore := deletedRestore()
	writes := &clientWrites{}
	testClient := newTestClientWithParent(t, writes.interceptor(), restore)
	reconciler := &restoreReconciler{k8sClient: testClient}

	updated, outcome := reconciler.ensureDeletingStatus(testCtx, restore)

	require.NotNil(t, updated)
	assert.Equal(t, k8sv1.RestoreStatusDeleting, updated.Status.Status) // NOSONAR -- legacy restore status compatibility
	assert.Equal(t, retryAfter(defaultRequeueDelay), outcome,
		"the status write must end the reconciliation so no second mutation runs")
	assert.Equal(t, 1, writes.parent.statusUpdates)
	assert.Equal(t, 1, writes.total())

	stored := &k8sv1.Restore{}
	require.NoError(t, testClient.Get(testCtx, newRestoreRequest(testRestore).NamespacedName, stored))
	assert.Equal(t, k8sv1.RestoreStatusDeleting, stored.Status.Status) // NOSONAR -- legacy restore status compatibility
}

func TestEnsureDeletingStatusRetriesAFailedStatusWrite(t *testing.T) {
	restore := deletedRestore()
	reconciler := &restoreReconciler{
		k8sClient: newTestClientWithParent(t, failingStatusUpdate(assert.AnError), restore),
	}

	updated, outcome := reconciler.ensureDeletingStatus(testCtx, restore)

	assert.Equal(t, restore, updated)
	require.Error(t, outcome.err)
	assert.ErrorIs(t, outcome.err, assert.AnError)
	assert.ErrorContains(t, outcome.err, "failed to persist deleting status of restore test-restore")
	assert.Equal(t, actionRetry, outcome.action)
	assert.Zero(t, outcome.requeueAfter, "controller-runtime backoff must decide when an error is retried")
}

func TestEnsureDeletingStatusUsesANoopStatusUpdateWithoutLosingTheWorkflow(t *testing.T) {
	restore := deletedRestore()
	restore.Status.Status = k8sv1.RestoreStatusDeleting // NOSONAR -- legacy restore status compatibility
	reconciler := &restoreReconciler{k8sClient: newTestClient(t, interceptor.Funcs{}, restore)}

	updated, outcome := reconciler.ensureDeletingStatus(testCtx, restore)

	assert.Equal(t, restore, updated)
	assert.Equal(t, next(), outcome)
}
