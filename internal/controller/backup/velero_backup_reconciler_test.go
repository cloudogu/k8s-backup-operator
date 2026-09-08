package backup

import (
	"context"
	"testing"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

func TestVeleroBackupReconcilerDeleteBackupIfExists(t *testing.T) {
	key := types.NamespacedName{Namespace: "ns", Name: "backup"}

	t.Run("Delete the backup of a deleted velero backup", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		fakeClient := newFakeClientBuilder(t).WithObjects(backup).Build()
		reconciler := NewVeleroBackupReconciler(fakeClient)

		require.NoError(t, reconciler.deleteBackupIfExists(context.Background(), key))

		err := fakeClient.Get(context.Background(), key, &backupv1.Backup{})
		assert.True(t, apierrors.IsNotFound(err), "a backup whose velero backup is gone cannot be restored")
		require.NoError(t, fakeClient.Get(context.Background(), key, &velerov1.DeleteBackupRequest{}),
			"the deletion has to reach the backup storage location as well")
	})

	t.Run("Keep a canceled backup whose provider backup the operator deleted itself", func(t *testing.T) {
		backup := canceledBackupForReconcilerTest("ns", "backup")
		fakeClient := newFakeClientBuilder(t).WithObjects(backup).Build()
		reconciler := NewVeleroBackupReconciler(fakeClient)

		require.NoError(t, reconciler.deleteBackupIfExists(context.Background(), key))

		require.NoError(t, fakeClient.Get(context.Background(), key, &backupv1.Backup{}),
			"a canceled backup is the failure history of its run and must survive")
		require.NoError(t, fakeClient.Get(context.Background(), key, &velerov1.DeleteBackupRequest{}),
			"the abandoned provider backup must still be removed from the backup storage location")
	})

	t.Run("Do nothing when there is no backup for the velero backup", func(t *testing.T) {
		fakeClient := newFakeClientBuilder(t).Build()
		reconciler := NewVeleroBackupReconciler(fakeClient)

		require.NoError(t, reconciler.deleteBackupIfExists(context.Background(), key))

		err := fakeClient.Get(context.Background(), key, &velerov1.DeleteBackupRequest{})
		assert.True(t, apierrors.IsNotFound(err))
	})

	t.Run("Report an error when the backup cannot be read", func(t *testing.T) {
		counter := &callCounter{getCallError: assert.AnError}
		fakeClient := newFakeClientBuilderWithCounter(t, counter).Build()
		reconciler := NewVeleroBackupReconciler(fakeClient)

		assert.Error(t, reconciler.deleteBackupIfExists(context.Background(), key))
	})
}
