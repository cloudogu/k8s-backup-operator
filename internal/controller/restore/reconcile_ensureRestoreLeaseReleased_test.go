package restore

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	coordinationv1 "k8s.io/api/coordination/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestEnsureRestoreLeaseReleased(t *testing.T) {
	t.Run("releases the lease held by the restore", func(t *testing.T) {
		restore := newParentRestore()
		testClient := newTestClientWithParent(t, interceptor.Funcs{}, restore)
		reconciler := NewRestoreReconciler(testClient, nil, testNamespace, nil, nil, requeueAfterTest, testBackupStorage)

		actual, outcome := reconciler.ensureRestoreLeaseReleased(testCtx, restore)

		assert.Same(t, restore, actual)
		assert.Equal(t, next(), outcome)
		lease := &coordinationv1.Lease{}
		err := testClient.Get(testCtx, leaseKey(), lease)
		assert.True(t, apierrors.IsNotFound(err))
	})

	t.Run("does not release a lease held by another operation", func(t *testing.T) {
		restore := newParentRestore()
		other := restoreWithIdentity("other-restore", "other-uid")
		testClient := newTestClient(t, interceptor.Funcs{}, restore, newRestoreLease(other))
		reconciler := NewRestoreReconciler(testClient, nil, testNamespace, nil, nil, requeueAfterTest, testBackupStorage)

		_, outcome := reconciler.ensureRestoreLeaseReleased(testCtx, restore)

		assert.Equal(t, next(), outcome)
		lease := &coordinationv1.Lease{}
		require.NoError(t, testClient.Get(testCtx, leaseKey(), lease))
		assert.Equal(t, other.Name, lease.Annotations[restoreLeaseHolderNameAnnotation])
	})

	t.Run("retries when deleting the lease fails", func(t *testing.T) {
		restore := newParentRestore()
		testClient := newTestClientWithParent(t, interceptor.Funcs{
			Delete: func(context.Context, client.WithWatch, client.Object, ...client.DeleteOption) error {
				return assert.AnError
			},
		}, restore)
		reconciler := NewRestoreReconciler(testClient, nil, testNamespace, nil, nil, requeueAfterTest, testBackupStorage)

		_, outcome := reconciler.ensureRestoreLeaseReleased(testCtx, restore)

		require.Error(t, outcome.err)
		assert.ErrorIs(t, outcome.err, assert.AnError)
		assert.ErrorContains(t, outcome.err, "failed to release restore lease")
	})
}
