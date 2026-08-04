package restore

import (
	"context"
	"testing"

	k8sv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestEnsureDeletionFinalizedAbortsWithoutWritingWhenTheFinalizerIsAbsent(t *testing.T) {
	restore := deletedRestore()
	restore.Finalizers = []string{"example.com/another-finalizer"}
	writes := &clientWrites{}
	reconciler := &restoreReconciler{k8sClient: newTestClientWithParent(t, writes.interceptor(), restore)}

	updated, outcome := reconciler.ensureDeletionFinalized(testCtx, restore)

	assert.Equal(t, restore, updated)
	assert.Equal(t, abort(), outcome)
	assert.Equal(t, 0, writes.total(), "an absent operator finalizer needs no write or event")
}

func TestEnsureDeletionFinalizedRemovesTheFinalizerAndLetsKubernetesDeleteTheRestore(t *testing.T) {
	restore := deletedRestore()
	writes := &clientWrites{}
	recorderMock := newMockEventRecorder(t)
	recorderMock.EXPECT().Event(matchesRestoreNamed(testRestore), corev1.EventTypeNormal,
		k8sv1.DeleteEventReason, "Delete successful").Return()
	testClient := newTestClientWithParent(t, writes.interceptor(), restore)
	reconciler := &restoreReconciler{k8sClient: testClient, recorder: recorderMock}

	updated, outcome := reconciler.ensureDeletionFinalized(testCtx, restore)

	assert.Equal(t, restore, updated)
	assert.Equal(t, abort(), outcome)
	assert.Equal(t, 1, writes.parent.updates)
	assert.Equal(t, 1, writes.total())

	stored := &k8sv1.Restore{}
	err := testClient.Get(testCtx, newRestoreRequest(testRestore).NamespacedName, stored)
	assert.True(t, apierrors.IsNotFound(err), "Kubernetes must finish deletion after the last finalizer is removed")
}

func TestEnsureDeletionFinalizedOnlyRemovesItsOwnFinalizer(t *testing.T) {
	restore := deletedRestore()
	restore.Finalizers = append(restore.Finalizers, "example.com/another-finalizer")
	recorderMock := newMockEventRecorder(t)
	recorderMock.EXPECT().Event(matchesRestoreNamed(testRestore), corev1.EventTypeNormal,
		k8sv1.DeleteEventReason, "Delete successful").Return()
	testClient := newTestClientWithParent(t, interceptor.Funcs{}, restore)
	reconciler := &restoreReconciler{k8sClient: testClient, recorder: recorderMock}

	_, outcome := reconciler.ensureDeletionFinalized(testCtx, restore)

	assert.Equal(t, abort(), outcome)
	stored := &k8sv1.Restore{}
	require.NoError(t, testClient.Get(testCtx, newRestoreRequest(testRestore).NamespacedName, stored))
	assert.NotContains(t, stored.Finalizers, k8sv1.RestoreFinalizer)
	assert.Contains(t, stored.Finalizers, "example.com/another-finalizer")
}

func TestEnsureDeletionFinalizedRetriesAFailedFinalizerRemoval(t *testing.T) {
	restore := deletedRestore()
	updateAttempts := 0
	failingFinalizerUpdate := interceptor.Funcs{
		Update: func(_ context.Context, _ client.WithWatch, _ client.Object, _ ...client.UpdateOption) error {
			updateAttempts++
			return assert.AnError
		},
	}
	recorderMock := newMockEventRecorder(t)
	recorderMock.EXPECT().Event(matchesRestoreNamed(testRestore), corev1.EventTypeWarning,
		k8sv1.DeleteEventReason, mock.MatchedBy(func(message string) bool {
			return assert.Contains(t, message, "Delete failed. Reason: failed to remove finalizer") &&
				assert.Contains(t, message, "assert.AnError")
		})).Return()
	testClient := newTestClientWithParent(t, failingFinalizerUpdate, restore)
	reconciler := &restoreReconciler{k8sClient: testClient, recorder: recorderMock}

	updated, outcome := reconciler.ensureDeletionFinalized(testCtx, restore)

	assert.Equal(t, restore, updated)
	require.Error(t, outcome.err)
	assert.ErrorIs(t, outcome.err, assert.AnError)
	assert.ErrorContains(t, outcome.err, "failed to remove finalizer cloudogu-restore-finalizer from restore test-restore")
	assert.Equal(t, actionRetry, outcome.action)
	assert.Zero(t, outcome.requeueAfter)
	assert.Equal(t, 1, updateAttempts)

	stored := &k8sv1.Restore{}
	require.NoError(t, testClient.Get(testCtx, newRestoreRequest(testRestore).NamespacedName, stored))
	assert.Contains(t, stored.Finalizers, k8sv1.RestoreFinalizer,
		"a failed update must leave the persisted finalizer in place")
}
