package backup

import (
	"context"
	"testing"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	veleroprovider "github.com/cloudogu/k8s-backup-operator/internal/provider/velero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
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

		_, outcome := reconciler.ensureBackupIsPrepared(context.Background(), backup)

		assert.NoError(t, outcome.err)
		assert.Equal(t, actionRetry, outcome.action)

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

		_, outcome := reconciler.ensureBackupIsPrepared(context.Background(), backup)

		assert.NoError(t, outcome.err)
		assert.Equal(t, actionRetry, outcome.action)

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

		_, outcome := reconciler.ensureBackupIsPrepared(context.Background(), backup)

		assert.NoError(t, outcome.err)
		assert.Equal(t, actionNext, outcome.action)

		preparedCondition := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionPrepared)
		assert.NotNil(t, preparedCondition)
		assert.Equal(t, metav1.ConditionTrue, preparedCondition.Status)
		assert.Equal(t, veleroprovider.ReasonVeleroBackupStorageLocationAvailable, preparedCondition.Reason)

		assert.Equal(t, 1, counter.subResourcePatchCount)
	})

	t.Run("An unready provider is reported once, not on every retry", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		fakeClient := newFakeClientBuilderWithCounter(t, &callCounter{}).
			WithObjects(backup).
			WithStatusSubresource(backup).
			Build()
		recorder := record.NewFakeRecorder(100)
		reconciler := NewReconciler(fakeClient, recorder, nil, newRealClock(), "default")

		_, outcome := reconciler.ensureBackupIsPrepared(context.Background(), backup)
		assert.NoError(t, outcome.err)
		assert.Equal(t, actionRetry, outcome.action)
		// second call to verify only one report
		_, outcome = reconciler.ensureBackupIsPrepared(context.Background(), backup)
		assert.NoError(t, outcome.err)
		assert.Equal(t, actionRetry, outcome.action)

		require.Len(t, recorder.Events, 1, "the unchanged provider state must be reported only once")
		assert.Contains(t, <-recorder.Events, veleroprovider.ReasonVeleroBackupStorageLocationNotFound)
	})

	t.Run("If an error occurred while getting the backup storage location resource then retry with erro", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		counter := &callCounter{
			getCallError: assert.AnError,
		}
		fakeClient := newFakeClientBuilderWithCounter(t, counter).
			WithObjects(backup).
			WithStatusSubresource(backup).
			Build()
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default")

		_, outcome := reconciler.ensureBackupIsPrepared(context.Background(), backup)

		assert.Error(t, outcome.err)
		assert.Equal(t, actionRetry, outcome.action)
	})

	t.Run("If the velero backup storage is unavailable and a patch error occurred then retry with error", func(t *testing.T) {
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

		_, outcome := reconciler.ensureBackupIsPrepared(context.Background(), backup)

		assert.Error(t, outcome.err)
		assert.Equal(t, actionRetry, outcome.action)
	})

	t.Run("If the velero backup storage is available and a patch error occurred then retry with error", func(t *testing.T) {
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

		_, outcome := reconciler.ensureBackupIsPrepared(context.Background(), backup)

		assert.Error(t, outcome.err)
		assert.Equal(t, actionRetry, outcome.action)
	})

	t.Run("If the velero backup storage resource was not found and a patch error occurred then retry with error", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		counter := &callCounter{
			subResourcePatchCallError: assert.AnError,
		}
		fakeClient := newFakeClientBuilderWithCounter(t, counter).
			WithObjects(backup).
			WithStatusSubresource(backup).
			Build()
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default")

		_, outcome := reconciler.ensureBackupIsPrepared(context.Background(), backup)

		assert.Error(t, outcome.err)
		assert.Equal(t, actionRetry, outcome.action)

	})
}
