package backup

import (
	"context"
	"errors"
	"testing"
	"time"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestCheckVeleroStatusSynced(t *testing.T) {
	t.Run("normal backup proceeds without reading Velero", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		reconciler := NewReconciler(newFakeClientBuilder(t).WithObjects(backup).Build(), newTestEventRecorder(), nil, newRealClock(), "default")

		nextAction, err := reconciler.ensureVeleroStatusSynced(context.Background(), backup)

		require.NoError(t, err)
		assert.Equal(t, Next, nextAction)
	})

	t.Run("completed Velero status and timestamps are synchronized", func(t *testing.T) {
		start := metav1.NewTime(time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC))
		completion := metav1.NewTime(start.Add(10 * time.Minute))
		backup := newBackupForTest("ns", "backup")
		backup.Spec.SyncedFromProvider = true
		veleroBackup := newVeleroBackupForReconcilerTest("ns", "backup", velerov1.BackupPhaseCompleted)
		veleroBackup.Status.StartTimestamp = &start
		veleroBackup.Status.CompletionTimestamp = &completion
		fakeClient := newFakeClientBuilder(t).
			WithObjects(backup, veleroBackup).
			WithStatusSubresource(backup).
			Build()
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default")

		nextAction, err := reconciler.ensureVeleroStatusSynced(context.Background(), backup)

		require.NoError(t, err)
		assert.Equal(t, Abort, nextAction)
		assert.Equal(t, backupv1.BackupStatusCompleted, backup.Status.Status)
		assert.True(t, backup.Status.StartTimestamp.Equal(&start))
		assert.True(t, backup.Status.CompletionTimestamp.Equal(&completion))
		assertCondition(t, backup, backupv1.ConditionPrepared, metav1.ConditionTrue, reasonVeleroStatusSynced)
		assertCondition(t, backup, backupv1.ConditionSucceeded, metav1.ConditionTrue, reasonVeleroStatusSynced)
		assertCondition(t, backup, backupv1.ConditionProviderSucceeded, metav1.ConditionTrue, reasonVeleroStatusSynced)
	})

	t.Run("running Velero backup is synchronized and retried", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		backup.Spec.SyncedFromProvider = true
		veleroBackup := newVeleroBackupForReconcilerTest("ns", "backup", velerov1.BackupPhaseInProgress)
		fakeClient := newFakeClientBuilder(t).
			WithObjects(backup, veleroBackup).
			WithStatusSubresource(backup).
			Build()
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default")

		nextAction, err := reconciler.ensureVeleroStatusSynced(context.Background(), backup)

		require.NoError(t, err)
		assert.Equal(t, Retry, nextAction)
		assert.Equal(t, backupv1.BackupStatusInProgress, backup.Status.Status)
		assertCondition(t, backup, backupv1.ConditionSucceeded, metav1.ConditionUnknown, reasonVeleroBackupRunning)
		assertCondition(t, backup, backupv1.ConditionProviderSucceeded, metav1.ConditionUnknown, reasonVeleroBackupRunning)
	})

	t.Run("failed Velero backup is synchronized and aborted", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		backup.Spec.SyncedFromProvider = true
		veleroBackup := newVeleroBackupForReconcilerTest("ns", "backup", velerov1.BackupPhaseFailed)
		fakeClient := newFakeClientBuilder(t).
			WithObjects(backup, veleroBackup).
			WithStatusSubresource(backup).
			Build()
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default")

		nextAction, err := reconciler.ensureVeleroStatusSynced(context.Background(), backup)

		require.NoError(t, err)
		assert.Equal(t, Abort, nextAction)
		assert.Equal(t, backupv1.BackupStatusFailed, backup.Status.Status)
		assertCondition(t, backup, backupv1.ConditionSucceeded, metav1.ConditionFalse, reasonVeleroBackupFailed)
		assertCondition(t, backup, backupv1.ConditionProviderSucceeded, metav1.ConditionFalse, reasonVeleroBackupFailed)
	})

	t.Run("reading the Velero backup fails", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		backup.Spec.SyncedFromProvider = true
		fakeClient := newFakeClientBuilder(t).
			WithObjects(backup).
			WithInterceptorFuncs(interceptor.Funcs{
				Get: func(context.Context, client.WithWatch, client.ObjectKey, client.Object, ...client.GetOption) error {
					return errors.New("read failed")
				},
			}).
			Build()
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default")

		nextAction, err := reconciler.ensureVeleroStatusSynced(context.Background(), backup)

		require.ErrorContains(t, err, "get velero backup to sync status")
		assert.Equal(t, Abort, nextAction)
	})
	t.Run("local backup does not read Velero", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		counter := &callCounter{}
		reconciler := NewReconciler(newFakeClientBuilderWithCounter(t, counter).WithObjects(backup).Build(), newTestEventRecorder(), nil, newRealClock(), "default")

		nextAction, err := reconciler.ensureVeleroStatusSynced(context.Background(), backup)

		require.NoError(t, err)
		assert.Equal(t, Next, nextAction)
		assert.Zero(t, counter.veleroBackupGetCount)
	})

	for _, phase := range veleroBackupFailedPhases {
		t.Run("failed phase "+string(phase), func(t *testing.T) {
			backup := newBackupForTest("ns", "backup")
			backup.Spec.SyncedFromProvider = true
			veleroBackup := newVeleroBackupForReconcilerTest("ns", "backup", phase)
			fakeClient := newFakeClientBuilder(t).WithObjects(backup, veleroBackup).WithStatusSubresource(backup).Build()
			reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default")

			nextAction, err := reconciler.ensureVeleroStatusSynced(context.Background(), backup)

			require.NoError(t, err)
			assert.Equal(t, Abort, nextAction)
			assert.Equal(t, backupv1.BackupStatusFailed, backup.Status.Status)
			assertCondition(t, backup, backupv1.ConditionSucceeded, metav1.ConditionFalse, reasonVeleroBackupFailed)
			assertCondition(t, backup, backupv1.ConditionProviderSucceeded, metav1.ConditionFalse, reasonVeleroBackupFailed)
		})
	}

	t.Run("unknown phase is treated as running", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		backup.Spec.SyncedFromProvider = true
		veleroBackup := newVeleroBackupForReconcilerTest("ns", "backup", velerov1.BackupPhase("Unexpected"))
		fakeClient := newFakeClientBuilder(t).WithObjects(backup, veleroBackup).WithStatusSubresource(backup).Build()
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default")

		nextAction, err := reconciler.ensureVeleroStatusSynced(context.Background(), backup)

		require.NoError(t, err)
		assert.Equal(t, Retry, nextAction)
		assert.Equal(t, backupv1.BackupStatusInProgress, backup.Status.Status) //nolint:staticcheck // legacy backup status compatibility
		assertCondition(t, backup, backupv1.ConditionSucceeded, metav1.ConditionUnknown, reasonVeleroBackupRunning)
		assertCondition(t, backup, backupv1.ConditionProviderSucceeded, metav1.ConditionUnknown, reasonVeleroBackupRunning)
	})

	t.Run("missing timestamps remain empty", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		backup.Spec.SyncedFromProvider = true
		veleroBackup := newVeleroBackupForReconcilerTest("ns", "backup", velerov1.BackupPhaseCompleted)
		fakeClient := newFakeClientBuilder(t).WithObjects(backup, veleroBackup).WithStatusSubresource(backup).Build()
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default")

		nextAction, err := reconciler.ensureVeleroStatusSynced(context.Background(), backup)

		require.NoError(t, err)
		assert.Equal(t, Abort, nextAction)
		assert.True(t, backup.Status.StartTimestamp.IsZero())
		assert.True(t, backup.Status.CompletionTimestamp.IsZero())
	})

	t.Run("missing Velero backup aborts", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		backup.Spec.SyncedFromProvider = true
		reconciler := NewReconciler(newFakeClientBuilder(t).WithObjects(backup).Build(), newTestEventRecorder(), nil, newRealClock(), "default")

		nextAction, err := reconciler.ensureVeleroStatusSynced(context.Background(), backup)

		require.ErrorContains(t, err, "get velero backup to sync status")
		assert.Equal(t, Abort, nextAction)
	})

	t.Run("status patch failure aborts", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		backup.Spec.SyncedFromProvider = true
		veleroBackup := newVeleroBackupForReconcilerTest("ns", "backup", velerov1.BackupPhaseCompleted)
		counter := &callCounter{subResourcePatchCallError: errors.New("patch failed")}
		fakeClient := newFakeClientBuilderWithCounter(t, counter).WithObjects(backup, veleroBackup).WithStatusSubresource(backup).Build()
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default")

		nextAction, err := reconciler.ensureVeleroStatusSynced(context.Background(), backup)

		require.ErrorContains(t, err, "patch backup status synchronized from Velero")
		assert.Equal(t, Abort, nextAction)
	})
}

func assertCondition(t *testing.T, backup *backupv1.Backup, conditionType string, status metav1.ConditionStatus, reason string) {
	t.Helper()
	condition := meta.FindStatusCondition(backup.Status.Conditions, conditionType)
	require.NotNil(t, condition)
	assert.Equal(t, status, condition.Status)
	assert.Equal(t, reason, condition.Reason)
}
