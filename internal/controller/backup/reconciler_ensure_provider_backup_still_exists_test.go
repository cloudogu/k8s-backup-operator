package backup

import (
	"context"
	"testing"
	"time"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func backupWithStartedProviderBackup(namespace string, name string) *backupv1.Backup {
	backup := newBackupForTest(namespace, name)
	backup.Status.StartTimestamp = metav1.NewTime(time.Now())

	return backup
}

func TestReconcilerEnsureProviderBackupStillExists(t *testing.T) {
	t.Run("Delete the backup when its provider backup is gone", func(t *testing.T) {
		backup := backupWithStartedProviderBackup("ns", "backup")
		fakeClient := newFakeClientBuilder(t).WithObjects(backup).Build()
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default")

		nextAction, err := reconciler.ensureProviderBackupStillExists(context.Background(), backup)

		assert.NoError(t, err)
		assert.Equal(t, Retry, nextAction, "the deletion has to be finalized even if its update event is lost")

		err = fakeClient.Get(context.Background(), client.ObjectKeyFromObject(backup), &backupv1.Backup{})
		assert.True(t, apierrors.IsNotFound(err), "the backup of a missing provider backup must be deleted")
	})

	t.Run("Keep the backup when its provider backup still exists", func(t *testing.T) {
		backup := backupWithStartedProviderBackup("ns", "backup")
		veleroBackup := newVeleroBackupForReconcilerTest("ns", "backup", velerov1.BackupPhaseCompleted)
		fakeClient := newFakeClientBuilder(t).WithObjects(backup, veleroBackup).Build()
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default")

		nextAction, err := reconciler.ensureProviderBackupStillExists(context.Background(), backup)

		assert.NoError(t, err)
		assert.Equal(t, Next, nextAction)
		require.NoError(t, fakeClient.Get(context.Background(), client.ObjectKeyFromObject(backup), &backupv1.Backup{}))
	})

	t.Run("Keep a backup that never started, it mirrors no provider backup", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		require.True(t, backup.Status.StartTimestamp.IsZero())
		fakeClient := newFakeClientBuilder(t).WithObjects(backup).Build()
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default")

		nextAction, err := reconciler.ensureProviderBackupStillExists(context.Background(), backup)

		assert.NoError(t, err)
		assert.Equal(t, Next, nextAction)
		require.NoError(t, fakeClient.Get(context.Background(), client.ObjectKeyFromObject(backup), &backupv1.Backup{}),
			"a canceled backup that never reached the provider must survive")
	})

	t.Run("Reconcile with backoff without deleting when the provider backup cannot be read", func(t *testing.T) {
		backup := backupWithStartedProviderBackup("ns", "backup")
		counter := &callCounter{getCallError: assert.AnError}
		fakeClient := newFakeClientBuilderWithCounter(t, counter).WithObjects(backup).Build()
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default")

		nextAction, err := reconciler.ensureProviderBackupStillExists(context.Background(), backup)

		require.ErrorIs(t, err, assert.AnError)
		assert.Equal(t, Abort, nextAction)
	})
}
