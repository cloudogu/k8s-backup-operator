package backup

import (
	"context"
	"testing"
	"time"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	veleroprovider "github.com/cloudogu/k8s-backup-operator/internal/provider/velero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/events"
)

func TestReconcilerEnsureBackupIsPrepared(t *testing.T) {
	t.Run("If the velero backup storage location was not found set prepared to false and retry", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		counter := &callCounter{}
		fakeClient := newFakeClientBuilderWithCounter(t, counter).
			WithObjects(backup).
			WithStatusSubresource(backup).
			Build()
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default")

		nextAction, err := reconciler.ensureBackupIsPrepared(context.Background(), backup)

		assert.NoError(t, err)
		assert.Equal(t, Retry, nextAction)

		preparedCondition := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionPrepared)
		assert.NotNil(t, preparedCondition)
		assert.Equal(t, metav1.ConditionFalse, preparedCondition.Status)
		assert.Equal(t, veleroprovider.ReasonVeleroBackupStorageLocationNotFound, preparedCondition.Reason)

		assert.Equal(t, 1, counter.subResourcePatchCount)
	})

	t.Run("If the velero backup storage is not available set prepared to false and retry", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		veleroBackupStorageLocation := newVeleroBackupStorageLocationForReconcilerTest(velerov1.BackupStorageLocationPhaseUnavailable)
		counter := &callCounter{}
		fakeClient := newFakeClientBuilderWithCounter(t, counter).
			WithObjects(backup, veleroBackupStorageLocation).
			WithStatusSubresource(backup).
			Build()
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default")

		nextAction, err := reconciler.ensureBackupIsPrepared(context.Background(), backup)

		assert.NoError(t, err)
		assert.Equal(t, Retry, nextAction)

		preparedCondition := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionPrepared)
		assert.NotNil(t, preparedCondition)
		assert.Equal(t, metav1.ConditionFalse, preparedCondition.Status)
		assert.Equal(t, veleroprovider.ReasonVeleroBackupStorageLocationNotAvailable, preparedCondition.Reason)

		assert.Equal(t, 1, counter.subResourcePatchCount)
	})

	t.Run("If the velero backup storage is available set prepared to true and proceed to the next stage", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		veleroBackupStorageLocation := newVeleroBackupStorageLocationForReconcilerTest(velerov1.BackupStorageLocationPhaseAvailable)
		counter := &callCounter{}
		fakeClient := newFakeClientBuilderWithCounter(t, counter).
			WithObjects(backup, veleroBackupStorageLocation).
			WithStatusSubresource(backup).
			Build()
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default")

		nextAction, err := reconciler.ensureBackupIsPrepared(context.Background(), backup)

		assert.NoError(t, err)
		assert.Equal(t, Next, nextAction)

		preparedCondition := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionPrepared)
		assert.NotNil(t, preparedCondition)
		assert.Equal(t, metav1.ConditionTrue, preparedCondition.Status)
		assert.Equal(t, veleroprovider.ReasonVeleroBackupStorageLocationAvailable, preparedCondition.Reason)

		assert.Equal(t, 1, counter.subResourcePatchCount)
	})

	t.Run("If a provider backup of another run is still running set prepared to false and retry", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		veleroBackupStorageLocation := newVeleroBackupStorageLocationForReconcilerTest(velerov1.BackupStorageLocationPhaseAvailable)
		orphanedVeleroBackup := newVeleroBackupForReconcilerTest("ns", "previous-backup", velerov1.BackupPhaseWaitingForPluginOperations)
		counter := &callCounter{}
		fakeClient := newFakeClientBuilderWithCounter(t, counter).
			WithObjects(backup, veleroBackupStorageLocation, orphanedVeleroBackup).
			WithStatusSubresource(backup).
			Build()
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default")

		nextAction, err := reconciler.ensureBackupIsPrepared(context.Background(), backup)

		assert.NoError(t, err)
		assert.Equal(t, Retry, nextAction)

		preparedCondition := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionPrepared)
		require.NotNil(t, preparedCondition)
		assert.Equal(t, metav1.ConditionFalse, preparedCondition.Status)
		assert.Equal(t, reasonOtherProviderBackupInProgress, preparedCondition.Reason)
		assert.Contains(t, preparedCondition.Message, "previous-backup")
		assert.Contains(t, preparedCondition.Message, velerov1.BackupPhaseWaitingForPluginOperations)
	})

	t.Run("The own velero backup does not block the preparation", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		veleroBackupStorageLocation := newVeleroBackupStorageLocationForReconcilerTest(velerov1.BackupStorageLocationPhaseAvailable)
		ownVeleroBackup := newVeleroBackupForReconcilerTest("ns", "backup", velerov1.BackupPhaseInProgress)
		fakeClient := newFakeClientBuilderWithCounter(t, &callCounter{}).
			WithObjects(backup, veleroBackupStorageLocation, ownVeleroBackup).
			WithStatusSubresource(backup).
			Build()
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default")

		nextAction, err := reconciler.ensureBackupIsPrepared(context.Background(), backup)

		assert.NoError(t, err)
		assert.Equal(t, Next, nextAction)

		preparedCondition := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionPrepared)
		require.NotNil(t, preparedCondition)
		assert.Equal(t, metav1.ConditionTrue, preparedCondition.Status)
	})

	t.Run("A running velero backup of another run does not block a backup that already started", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		backup.Status.StartTimestamp = metav1.Time{Time: time.Now()}
		veleroBackupStorageLocation := newVeleroBackupStorageLocationForReconcilerTest(velerov1.BackupStorageLocationPhaseAvailable)
		foreignVeleroBackup := newVeleroBackupForReconcilerTest("ns", "foreign-backup", velerov1.BackupPhaseInProgress)
		fakeClient := newFakeClientBuilderWithCounter(t, &callCounter{}).
			WithObjects(backup, veleroBackupStorageLocation, foreignVeleroBackup).
			WithStatusSubresource(backup).
			Build()
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default")

		nextAction, err := reconciler.ensureBackupIsPrepared(context.Background(), backup)

		assert.NoError(t, err)
		assert.Equal(t, Next, nextAction)

		preparedCondition := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionPrepared)
		require.NotNil(t, preparedCondition)
		assert.Equal(t, metav1.ConditionTrue, preparedCondition.Status)
	})

	t.Run("A finished velero backup of another run does not block the preparation", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		veleroBackupStorageLocation := newVeleroBackupStorageLocationForReconcilerTest(velerov1.BackupStorageLocationPhaseAvailable)
		previousVeleroBackup := newVeleroBackupForReconcilerTest("ns", "previous-backup", velerov1.BackupPhaseCompleted)
		failedVeleroBackup := newVeleroBackupForReconcilerTest("ns", "failed-backup", velerov1.BackupPhasePartiallyFailed)
		fakeClient := newFakeClientBuilderWithCounter(t, &callCounter{}).
			WithObjects(backup, veleroBackupStorageLocation, previousVeleroBackup, failedVeleroBackup).
			WithStatusSubresource(backup).
			Build()
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default")

		nextAction, err := reconciler.ensureBackupIsPrepared(context.Background(), backup)

		assert.NoError(t, err)
		assert.Equal(t, Next, nextAction)

		preparedCondition := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionPrepared)
		require.NotNil(t, preparedCondition)
		assert.Equal(t, metav1.ConditionTrue, preparedCondition.Status)
	})

	t.Run("A velero backup of another namespace does not block the preparation", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		veleroBackupStorageLocation := newVeleroBackupStorageLocationForReconcilerTest(velerov1.BackupStorageLocationPhaseAvailable)
		foreignVeleroBackup := newVeleroBackupForReconcilerTest("other-ns", "previous-backup", velerov1.BackupPhaseInProgress)
		fakeClient := newFakeClientBuilderWithCounter(t, &callCounter{}).
			WithObjects(backup, veleroBackupStorageLocation, foreignVeleroBackup).
			WithStatusSubresource(backup).
			Build()
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default")

		nextAction, err := reconciler.ensureBackupIsPrepared(context.Background(), backup)

		assert.NoError(t, err)
		assert.Equal(t, Next, nextAction)
	})

	t.Run("If listing the velero backups failed then abort", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		veleroBackupStorageLocation := newVeleroBackupStorageLocationForReconcilerTest(velerov1.BackupStorageLocationPhaseAvailable)
		counter := &callCounter{
			veleroBackupListCallError: assert.AnError,
		}
		fakeClient := newFakeClientBuilderWithCounter(t, counter).
			WithObjects(backup, veleroBackupStorageLocation).
			WithStatusSubresource(backup).
			Build()
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default")

		nextAction, err := reconciler.ensureBackupIsPrepared(context.Background(), backup)

		assert.Error(t, err)
		assert.Equal(t, Abort, nextAction)
	})

	t.Run("An unready provider is reported once, not on every retry", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		fakeClient := newFakeClientBuilderWithCounter(t, &callCounter{}).
			WithObjects(backup).
			WithStatusSubresource(backup).
			Build()
		recorder := events.NewFakeRecorder(100)
		reconciler := NewReconciler(fakeClient, recorder, nil, newRealClock(), "default")

		nextAction, err := reconciler.ensureBackupIsPrepared(context.Background(), backup)
		assert.NoError(t, err)
		assert.Equal(t, Retry, nextAction)
		// second call to verify only one report
		nextAction, err = reconciler.ensureBackupIsPrepared(context.Background(), backup)
		assert.NoError(t, err)
		assert.Equal(t, Retry, nextAction)

		require.Len(t, recorder.Events, 1, "the unchanged provider state must be reported only once")
		assert.Contains(t, <-recorder.Events, veleroprovider.ReasonVeleroBackupStorageLocationNotFound)
	})

	t.Run("If an error occurred while getting the backup storage location resource then abort", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		counter := &callCounter{
			getCallError: assert.AnError,
		}
		fakeClient := newFakeClientBuilderWithCounter(t, counter).
			WithObjects(backup).
			WithStatusSubresource(backup).
			Build()
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default")

		nextAction, err := reconciler.ensureBackupIsPrepared(context.Background(), backup)

		assert.Error(t, err)
		assert.Equal(t, Abort, nextAction)
	})

	t.Run("If the velero backup storage is unavailable and a patch error occurred then abort", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		veleroBackupStorageLocation := newVeleroBackupStorageLocationForReconcilerTest(velerov1.BackupStorageLocationPhaseUnavailable)
		counter := &callCounter{
			subResourcePatchCallError: assert.AnError,
		}
		fakeClient := newFakeClientBuilderWithCounter(t, counter).
			WithObjects(backup, veleroBackupStorageLocation).
			WithStatusSubresource(backup).
			Build()
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default")

		nextAction, err := reconciler.ensureBackupIsPrepared(context.Background(), backup)

		assert.Error(t, err)
		assert.Equal(t, Abort, nextAction)
	})

	t.Run("If the velero backup storage is available and a patch error occurred then abort", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		veleroBackupStorageLocation := newVeleroBackupStorageLocationForReconcilerTest(velerov1.BackupStorageLocationPhaseAvailable)
		counter := &callCounter{
			subResourcePatchCallError: assert.AnError,
		}
		fakeClient := newFakeClientBuilderWithCounter(t, counter).
			WithObjects(backup, veleroBackupStorageLocation).
			WithStatusSubresource(backup).
			Build()
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default")

		nextAction, err := reconciler.ensureBackupIsPrepared(context.Background(), backup)

		assert.Error(t, err)
		assert.Equal(t, Abort, nextAction)
	})

	t.Run("If the velero backup storage resource was not found and a patch error occurred then abort", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		counter := &callCounter{
			subResourcePatchCallError: assert.AnError,
		}
		fakeClient := newFakeClientBuilderWithCounter(t, counter).
			WithObjects(backup).
			WithStatusSubresource(backup).
			Build()
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default")

		nextAction, err := reconciler.ensureBackupIsPrepared(context.Background(), backup)

		assert.Error(t, err)
		assert.Equal(t, Abort, nextAction)

	})
}
