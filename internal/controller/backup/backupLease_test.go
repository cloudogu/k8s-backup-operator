package backup

import (
	"context"
	"testing"

	"github.com/cloudogu/k8s-backup-operator/internal/leases"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	coordinationv1 "k8s.io/api/coordination/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestBackupLease(t *testing.T) {
	ctx := context.Background()

	t.Run("acquires the shared lease idempotently", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		backup.UID = types.UID("backup-uid")
		k8sClient := newFakeClientBuilder(t).WithObjects(backup).Build()
		reconciler := NewReconciler(k8sClient, nil, newRealClock(), "default")

		nextAction, err := reconciler.ensureActiveBackupLease(ctx, backup, logr.Discard())
		require.NoError(t, err)
		assert.Equal(t, Retry, nextAction)

		lease := &coordinationv1.Lease{}
		require.NoError(t, k8sClient.Get(ctx, client.ObjectKey{Namespace: backup.Namespace, Name: leases.DefaultName}, lease))
		assert.Equal(t, backup.Name, lease.Annotations[leases.HolderNameAnnotation])
		assert.Equal(t, backupLeaseHolderKind, lease.Annotations[leases.HolderKindAnnotation])

		nextAction, err = reconciler.ensureActiveBackupLease(ctx, backup, logr.Discard())
		require.NoError(t, err)
		assert.Equal(t, Next, nextAction)
	})

	t.Run("waits for a restore holding the same lease", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		backup.UID = types.UID("backup-uid")
		restoreHolder := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns", Name: "restore", UID: types.UID("restore-uid"),
		}}
		lease := leases.NewLease("ns", leases.DefaultName, restoreHolder, "Restore")
		k8sClient := newFakeClientBuilder(t).WithObjects(backup, lease).Build()
		reconciler := NewReconciler(k8sClient, nil, newRealClock(), "default")

		nextAction, err := reconciler.ensureActiveBackupLease(ctx, backup, logr.Discard())
		require.NoError(t, err)
		assert.Equal(t, Retry, nextAction)

		stored := &coordinationv1.Lease{}
		require.NoError(t, k8sClient.Get(ctx, client.ObjectKeyFromObject(lease), stored))
		assert.Equal(t, "restore", stored.Annotations[leases.HolderNameAnnotation])
		assert.Equal(t, "Restore", stored.Annotations[leases.HolderKindAnnotation])
	})
}
