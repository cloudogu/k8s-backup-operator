package backup

import (
	"context"
	"errors"
	"testing"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/k8s-backup-operator/internal/leases"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	coordinationv1 "k8s.io/api/coordination/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestBackupLease(t *testing.T) {
	ctx := context.Background()

	t.Run("acquires the shared lease idempotently", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		backup.UID = types.UID("backup-uid")
		k8sClient := newFakeClientBuilder(t).WithObjects(backup).Build()
		reconciler := NewReconciler(k8sClient, nil, newRealClock(), "default")

		nextAction, err := reconciler.ensureActiveBackupLease(ctx, backup)
		require.NoError(t, err)
		assert.Equal(t, Retry, nextAction)

		lease := &coordinationv1.Lease{}
		require.NoError(t, k8sClient.Get(ctx, client.ObjectKey{Namespace: backup.Namespace, Name: leases.DefaultName}, lease))
		require.NotNil(t, lease.Spec.HolderIdentity)
		assert.Equal(t, string(backup.UID), *lease.Spec.HolderIdentity)
		assert.Equal(t, backup.Name, lease.Annotations[leases.HolderNameAnnotation])
		assert.Equal(t, backupLeaseHolderKind, lease.Annotations[leases.HolderKindAnnotation])

		nextAction, err = reconciler.ensureActiveBackupLease(ctx, backup)
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

		nextAction, err := reconciler.ensureActiveBackupLease(ctx, backup)
		require.NoError(t, err)
		assert.Equal(t, Retry, nextAction)

		stored := &coordinationv1.Lease{}
		require.NoError(t, k8sClient.Get(ctx, client.ObjectKeyFromObject(lease), stored))
		assert.Equal(t, "restore", stored.Annotations[leases.HolderNameAnnotation])
		assert.Equal(t, "Restore", stored.Annotations[leases.HolderKindAnnotation])
	})

	t.Run("rejects an invalid lease without holder identity", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		backup.UID = types.UID("backup-uid")
		lease := &coordinationv1.Lease{ObjectMeta: metav1.ObjectMeta{Namespace: "ns", Name: leases.DefaultName}}
		k8sClient := newFakeClientBuilder(t).WithObjects(backup, lease).Build()
		reconciler := NewReconciler(k8sClient, nil, newRealClock(), "default")

		nextAction, err := reconciler.ensureActiveBackupLease(ctx, backup)

		assert.Equal(t, Abort, nextAction)
		require.ErrorContains(t, err, "blocked by invalid lease")
	})

	t.Run("reports an error while acquiring the lease", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		k8sClient := newFakeClientBuilder(t).WithInterceptorFuncs(interceptor.Funcs{
			Get: func(_ context.Context, _ client.WithWatch, _ client.ObjectKey, _ client.Object, _ ...client.GetOption) error {
				return errors.New("get failed")
			},
		}).Build()
		reconciler := NewReconciler(k8sClient, nil, newRealClock(), "default")

		nextAction, err := reconciler.ensureActiveBackupLease(ctx, backup)

		assert.Equal(t, Abort, nextAction)
		require.ErrorContains(t, err, "acquire backup lease")
	})

	t.Run("rejects an unknown acquisition state", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")

		nextAction, err := backupLeaseAction(ctx, backup, leases.Result{State: leases.State(99)}, nil)

		assert.Equal(t, Abort, nextAction)
		require.ErrorContains(t, err, "unknown backup lease acquisition state 99")
	})
}

func TestEnsureBackupLeaseReleased(t *testing.T) {
	ctx := context.Background()

	t.Run("keeps the lease while the backup is running", func(t *testing.T) {
		backup := newBackupWithSucceededStatusForReconcilerTest("ns", "backup", metav1.ConditionUnknown)
		reconciler := NewReconciler(nil, nil, newRealClock(), "default")

		nextAction, err := reconciler.ensureBackupLeaseReleased(ctx, backup)

		require.NoError(t, err)
		assert.Equal(t, Next, nextAction)
	})

	tests := []struct {
		name   string
		backup *backupv1.Backup
	}{
		{name: "successful backup", backup: newBackupWithSucceededStatusForReconcilerTest("ns", "backup", metav1.ConditionTrue)},
		{name: "failed backup", backup: newBackupWithSucceededStatusForReconcilerTest("ns", "backup", metav1.ConditionFalse)},
		{name: "canceled backup", backup: backupWithCondition(metav1.Condition{Type: backupv1.ConditionCanceled, Status: metav1.ConditionTrue})},
		{name: "deleting backup", backup: deletingBackupForLeaseTest()},
	}
	for _, tt := range tests {
		t.Run("releases the lease for "+tt.name, func(t *testing.T) {
			tt.backup.UID = types.UID("backup-uid")
			lease := leases.NewLease("ns", leases.DefaultName, tt.backup, backupLeaseHolderKind)
			k8sClient := newFakeClientBuilder(t).WithObjects(lease).Build()
			reconciler := NewReconciler(k8sClient, nil, newRealClock(), "default")

			nextAction, err := reconciler.ensureBackupLeaseReleased(ctx, tt.backup)

			require.NoError(t, err)
			assert.Equal(t, Next, nextAction)
			stored := &coordinationv1.Lease{}
			assert.Error(t, k8sClient.Get(ctx, client.ObjectKeyFromObject(lease), stored))
		})
	}

	t.Run("reports an error while releasing the lease", func(t *testing.T) {
		backup := newBackupWithSucceededStatusForReconcilerTest("ns", "backup", metav1.ConditionTrue)
		k8sClient := newFakeClientBuilder(t).WithInterceptorFuncs(interceptor.Funcs{
			Get: func(_ context.Context, _ client.WithWatch, _ client.ObjectKey, _ client.Object, _ ...client.GetOption) error {
				return errors.New("get failed")
			},
		}).Build()
		reconciler := NewReconciler(k8sClient, nil, newRealClock(), "default")

		nextAction, err := reconciler.ensureBackupLeaseReleased(ctx, backup)

		assert.Equal(t, Abort, nextAction)
		require.ErrorContains(t, err, "release backup lease")
	})
}

func TestBackupHolderResolver(t *testing.T) {
	ctx := context.Background()
	first := newBackupForTest("ns", "first")
	first.UID = types.UID("first-uid")
	second := newBackupForTest("ns", "second")
	second.UID = types.UID("second-uid")
	resolver := backupHolderResolver{client: newFakeClientBuilder(t).WithObjects(first, second).Build()}

	assert.Equal(t, backupLeaseHolderKind, resolver.Kind())

	found, err := resolver.Get(ctx, "ns", "first")
	require.NoError(t, err)
	assert.Equal(t, first.Name, found.GetName())

	_, err = resolver.Get(ctx, "ns", "missing")
	assert.Error(t, err)

}

func TestBackupHolderResolverTerminalStates(t *testing.T) {
	tests := []struct {
		name     string
		holder   client.Object
		terminal bool
	}{
		{name: "different object type", holder: &corev1.ConfigMap{}, terminal: false},
		{name: "backup without conditions", holder: newBackupForTest("ns", "backup"), terminal: false},
		{name: "running backup", holder: newBackupWithSucceededStatusForReconcilerTest("ns", "backup", metav1.ConditionUnknown), terminal: false},
		{name: "successful backup", holder: newBackupWithSucceededStatusForReconcilerTest("ns", "backup", metav1.ConditionTrue), terminal: true},
		{name: "failed backup", holder: newBackupWithSucceededStatusForReconcilerTest("ns", "backup", metav1.ConditionFalse), terminal: true},
		{name: "canceled backup", holder: backupWithCondition(metav1.Condition{Type: backupv1.ConditionCanceled, Status: metav1.ConditionTrue}), terminal: true},
	}

	resolver := backupHolderResolver{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.terminal, resolver.IsTerminal(tt.holder))
		})
	}
}

func backupWithCondition(condition metav1.Condition) *backupv1.Backup {
	backup := newBackupForTest("ns", "backup")
	backup.Status.Conditions = []metav1.Condition{condition}
	return backup
}

func deletingBackupForLeaseTest() *backupv1.Backup {
	backup := newBackupForTest("ns", "backup")
	deletionTimestamp := metav1.Now()
	backup.DeletionTimestamp = &deletionTimestamp
	return backup
}
