package restore

import (
	"context"
	"strings"
	"testing"
	"time"

	k8sv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/k8s-backup-operator/internal/provider/velero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestEnsureProviderRestoreDeletedContinuesWhenTheChildIsAlreadyAbsent(t *testing.T) {
	restore := deletedRestore()
	writes := &clientWrites{}
	reconciler := &restoreReconciler{k8sClient: newTestClientWithParent(t, writes.interceptor(), restore)}

	updated, outcome := reconciler.ensureProviderRestoreDeleted(testCtx, restore)

	assert.Equal(t, restore, updated)
	assert.Equal(t, next(), outcome)
	assert.Equal(t, 0, writes.total(), "an absent provider restore needs no write")
}

func TestEnsureProviderRestoreDeletedDeletesAnOwnedChildAndWaitsForItsDisappearance(t *testing.T) {
	restore := deletedRestore()
	restore.UID = testRestoreUID
	child := velero.BuildRestore(restore)
	writes := &clientWrites{}
	testClient := newTestClientWithParent(t, writes.interceptor(), restore, child)
	reconciler := &restoreReconciler{k8sClient: testClient}

	updated, outcome := reconciler.ensureProviderRestoreDeleted(testCtx, restore)

	assert.Equal(t, restore, updated)
	assert.Equal(t, retryAfter(defaultRequeueDelay), outcome,
		"an accepted delete request must be verified by a later reconciliation")
	assert.Equal(t, 1, writes.child.deletes)
	assert.Equal(t, 1, writes.total())

	stored := &velerov1.Restore{}
	err := testClient.Get(testCtx, types.NamespacedName{Namespace: testNamespace, Name: testRestore}, stored)
	assert.True(t, apierrors.IsNotFound(err))
}

func TestEnsureProviderRestoreDeletedDeletesAnUnownedChildOfALegacyRestore(t *testing.T) {
	restore := deletedRestore()
	restore.Status.Status = k8sv1.RestoreStatusCompleted
	child := velero.BuildRestore(restore)
	child.OwnerReferences = nil
	writes := &clientWrites{}
	testClient := newTestClientWithParent(t, writes.interceptor(), restore, child)
	reconciler := &restoreReconciler{k8sClient: testClient}

	_, outcome := reconciler.ensureProviderRestoreDeleted(testCtx, restore)

	assert.Equal(t, retryAfter(defaultRequeueDelay), outcome)
	assert.Equal(t, 1, writes.child.deletes,
		"legacy restores predate owner references, so their namesake child is still deleted")
}

func TestEnsureProviderRestoreDeletedLeavesAForeignChildUntouched(t *testing.T) {
	restore := deletedRestore()
	child := velero.BuildRestore(restore)
	child.OwnerReferences = nil
	writes := &clientWrites{}
	recorderMock := newMockEventRecorder(t)
	recorderMock.EXPECT().Event(matchesRestoreNamed(testRestore), corev1.EventTypeWarning, k8sv1.DeleteEventReason,
		mock.MatchedBy(func(message string) bool {
			return strings.Contains(message, "not owned by this restore")
		})).Return()
	testClient := newTestClientWithParent(t, writes.interceptor(), restore, child)
	reconciler := &restoreReconciler{k8sClient: testClient, recorder: recorderMock}

	updated, outcome := reconciler.ensureProviderRestoreDeleted(testCtx, restore)

	assert.Equal(t, restore, updated)
	assert.Equal(t, next(), outcome, "a foreign child must not wedge deletion of the parent")
	assert.Equal(t, 0, writes.total(), "a foreign child must not be mutated")

	stored := &velerov1.Restore{}
	require.NoError(t, testClient.Get(testCtx, client.ObjectKeyFromObject(child), stored))
}

func TestEnsureProviderRestoreDeletedWaitsForATerminatingOwnedChildWithoutDeletingItAgain(t *testing.T) {
	restore := deletedRestore()
	restore.UID = testRestoreUID
	child := velero.BuildRestore(restore)
	child.Finalizers = []string{"velero.io/test-finalizer"}
	child.DeletionTimestamp = &metav1.Time{Time: time.Now()}
	writes := &clientWrites{}
	reconciler := &restoreReconciler{
		k8sClient: newTestClientWithParent(t, writes.interceptor(), restore, child),
	}

	updated, outcome := reconciler.ensureProviderRestoreDeleted(testCtx, restore)

	assert.Equal(t, restore, updated)
	assert.Equal(t, retryAfter(defaultRequeueDelay), outcome)
	assert.Equal(t, 0, writes.total(), "a child whose deletion is in progress must not receive another delete")
}

func TestEnsureProviderRestoreDeletedRetriesAnUnreadableChild(t *testing.T) {
	restore := deletedRestore()
	failingChildGet := interceptor.Funcs{
		Get: func(ctx context.Context, wrapped client.WithWatch, key client.ObjectKey, object client.Object, opts ...client.GetOption) error {
			if _, isChild := object.(*velerov1.Restore); isChild {
				return assert.AnError
			}

			return wrapped.Get(ctx, key, object, opts...)
		},
	}
	reconciler := &restoreReconciler{k8sClient: newTestClientWithParent(t, failingChildGet, restore)}

	updated, outcome := reconciler.ensureProviderRestoreDeleted(testCtx, restore)

	assert.Equal(t, restore, updated)
	require.Error(t, outcome.err)
	assert.ErrorIs(t, outcome.err, assert.AnError)
	assert.ErrorContains(t, outcome.err, "failed to get provider restore of restore test-restore")
	assert.Equal(t, actionRetry, outcome.action)
}

func TestEnsureProviderRestoreDeletedRetriesAFailedChildDeletion(t *testing.T) {
	restore := deletedRestore()
	restore.UID = testRestoreUID
	child := velero.BuildRestore(restore)
	reconciler := &restoreReconciler{
		k8sClient: newTestClientWithParent(t, failingDelete(assert.AnError), restore, child),
	}

	updated, outcome := reconciler.ensureProviderRestoreDeleted(testCtx, restore)

	assert.Equal(t, restore, updated)
	require.Error(t, outcome.err)
	assert.ErrorIs(t, outcome.err, assert.AnError)
	assert.ErrorContains(t, outcome.err, "failed to delete provider restore of restore test-restore")
	assert.Equal(t, actionRetry, outcome.action)
}
