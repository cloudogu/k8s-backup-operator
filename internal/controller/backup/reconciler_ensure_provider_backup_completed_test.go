package backup

import (
	"context"
	"errors"
	"testing"
	"time"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestReconcilerEnsureProviderBackupCompleted(t *testing.T) {
	t.Run("If the velero backup is in progress, set succeeded to unknown and retry", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		veleroBackup := newVeleroBackupForReconcilerTest("ns", "backup", velerov1.BackupPhaseInProgress)
		counter := &callCounter{}
		fakeClient := newFakeClientBuilderWithCounter(t, counter).
			WithObjects(backup, veleroBackup).
			WithStatusSubresource(backup).
			Build()
		reconciler := NewReconciler(fakeClient, nil, DefaultClock{})

		nextAction, err := reconciler.ensureProviderBackupCompleted(context.Background(), backup, logr.Discard())

		assert.NoError(t, err)
		assert.Equal(t, Retry, nextAction)

		succeededCondition := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionSucceeded)
		assert.NotNil(t, succeededCondition)
		assert.Equal(t, metav1.ConditionUnknown, succeededCondition.Status)
		assert.Equal(t, reasonProviderBackupInProgress, succeededCondition.Reason)

		assert.Equal(t, 1, counter.subResourcePatchCount)
	})

	t.Run("If the velero backup has failed, set succeeded to false, set completion time and proceed to the next step", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		veleroBackup := newVeleroBackupForReconcilerTest("ns", "backup", velerov1.BackupPhaseFailed)
		counter := &callCounter{}
		fakeClient := newFakeClientBuilderWithCounter(t, counter).
			WithObjects(backup, veleroBackup).
			WithStatusSubresource(backup).
			Build()
		reconciler := NewReconciler(fakeClient, nil, DefaultClock{})

		require.True(t, backup.Status.CompletionTimestamp.IsZero())

		nextAction, err := reconciler.ensureProviderBackupCompleted(context.Background(), backup, logr.Discard())

		assert.NoError(t, err)
		assert.Equal(t, Next, nextAction)

		succeededCondition := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionSucceeded)
		assert.NotNil(t, succeededCondition)
		assert.Equal(t, metav1.ConditionFalse, succeededCondition.Status)
		assert.Equal(t, reasonProviderBackupFailed, succeededCondition.Reason)

		assert.False(t, backup.Status.CompletionTimestamp.IsZero())

		assert.Equal(t, 1, counter.subResourcePatchCount)
	})

	t.Run("If the velero backup has failed, set succeeded to false, set completion time only once and "+
		"proceed to the next step", func(t *testing.T) {

		// The completion time is set by using the function time.Now(). If we used time.Now() here as base time
		// we would not be able to detect if the completion time was modified. The time difference would be too small.
		baseTime := metav1.NewTime(time.Now().Add(time.Minute))
		backup := newBackupForTest("ns", "backup")
		backup.Status.CompletionTimestamp = baseTime
		veleroBackup := newVeleroBackupForReconcilerTest("ns", "backup", velerov1.BackupPhaseFailed)
		counter := &callCounter{}
		fakeClient := newFakeClientBuilderWithCounter(t, counter).
			WithObjects(backup, veleroBackup).
			WithStatusSubresource(backup).
			Build()
		reconciler := NewReconciler(fakeClient, nil, DefaultClock{})

		require.False(t, backup.Status.CompletionTimestamp.IsZero())

		nextAction, err := reconciler.ensureProviderBackupCompleted(context.Background(), backup, logr.Discard())

		assert.NoError(t, err)
		assert.Equal(t, Next, nextAction)

		succeededCondition := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionSucceeded)
		assert.NotNil(t, succeededCondition)
		assert.Equal(t, metav1.ConditionFalse, succeededCondition.Status)
		assert.Equal(t, reasonProviderBackupFailed, succeededCondition.Reason)

		assert.WithinDuration(t, baseTime.Time, backup.Status.CompletionTimestamp.Time, time.Second)

		assert.Equal(t, 1, counter.subResourcePatchCount)
	})

	t.Run("If the velero backup has succeed, set succeeded to true, set completion time only once and "+
		"proceed to the next step", func(t *testing.T) {

		// The completion time is set by using the function time.Now(). If we used time.Now() here as base time
		// we would not be able to detect if the completion time was modified. The time difference would be too small.
		baseTime := metav1.NewTime(time.Now().Add(time.Minute))
		backup := newBackupForTest("ns", "backup")
		backup.Status.CompletionTimestamp = baseTime
		veleroBackup := newVeleroBackupForReconcilerTest("ns", "backup", velerov1.BackupPhaseCompleted)
		counter := &callCounter{}
		fakeClient := newFakeClientBuilderWithCounter(t, counter).
			WithObjects(backup, veleroBackup).
			WithStatusSubresource(backup).
			Build()
		reconciler := NewReconciler(fakeClient, nil, DefaultClock{})

		nextAction, err := reconciler.ensureProviderBackupCompleted(context.Background(), backup, logr.Discard())

		assert.NoError(t, err)
		assert.Equal(t, Next, nextAction)

		succeededCondition := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionSucceeded)
		assert.NotNil(t, succeededCondition)
		assert.Equal(t, metav1.ConditionTrue, succeededCondition.Status)
		assert.Equal(t, reasonProviderBackupSucceeded, succeededCondition.Reason)

		assert.WithinDuration(t, baseTime.Time, backup.Status.CompletionTimestamp.Time, time.Second)

		assert.Equal(t, 1, counter.subResourcePatchCount)
	})

	t.Run("If the velero backup has succeed, set succeeded to true, set completion time and "+
		"proceed to the next step", func(t *testing.T) {

		backup := newBackupForTest("ns", "backup")
		veleroBackup := newVeleroBackupForReconcilerTest("ns", "backup", velerov1.BackupPhaseCompleted)
		counter := &callCounter{}
		fakeClient := newFakeClientBuilderWithCounter(t, counter).
			WithObjects(backup, veleroBackup).
			WithStatusSubresource(backup).
			Build()
		reconciler := NewReconciler(fakeClient, nil, DefaultClock{})

		nextAction, err := reconciler.ensureProviderBackupCompleted(context.Background(), backup, logr.Discard())

		assert.NoError(t, err)
		assert.Equal(t, Next, nextAction)

		succeededCondition := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionSucceeded)
		assert.NotNil(t, succeededCondition)
		assert.Equal(t, metav1.ConditionTrue, succeededCondition.Status)
		assert.Equal(t, reasonProviderBackupSucceeded, succeededCondition.Reason)

		assert.False(t, backup.Status.CompletionTimestamp.IsZero())

		assert.Equal(t, 1, counter.subResourcePatchCount)
	})

	t.Run("If the velero backup is not in status InProgress, Failed or Succeeded, abort", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		veleroBackup := newVeleroBackupForReconcilerTest("ns", "backup", velerov1.BackupPhaseDeleting)
		counter := &callCounter{}
		fakeClient := newFakeClientBuilderWithCounter(t, counter).
			WithObjects(backup, veleroBackup).
			WithStatusSubresource(backup).
			Build()
		reconciler := NewReconciler(fakeClient, nil, DefaultClock{})

		nextAction, err := reconciler.ensureProviderBackupCompleted(context.Background(), backup, logr.Discard())

		assert.Error(t, err)
		assert.Equal(t, Abort, nextAction)
	})

	t.Run("If getting the velero backup resource failed, abort", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		counter := &callCounter{
			getCallError: errors.New("get error"),
		}
		fakeClient := newFakeClientBuilderWithCounter(t, counter).
			WithObjects(backup).
			WithStatusSubresource(backup).
			Build()
		reconciler := NewReconciler(fakeClient, nil, DefaultClock{})

		nextAction, err := reconciler.ensureProviderBackupCompleted(context.Background(), backup, logr.Discard())

		assert.Error(t, err)
		assert.ErrorContains(t, err, "get error")
		assert.Equal(t, Abort, nextAction)
	})

	t.Run("If patching the status fails, abort", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		veleroBackup := newVeleroBackupForReconcilerTest("ns", "backup", velerov1.BackupPhaseInProgress)
		counter := &callCounter{
			subResourcePatchCallError: errors.New("patch error"),
		}
		fakeClient := newFakeClientBuilderWithCounter(t, counter).
			WithObjects(backup, veleroBackup).
			WithStatusSubresource(backup).
			Build()
		reconciler := NewReconciler(fakeClient, nil, DefaultClock{})

		nextAction, err := reconciler.ensureProviderBackupCompleted(context.Background(), backup, logr.Discard())

		assert.Error(t, err)
		assert.ErrorContains(t, err, "patch error")
		assert.Equal(t, Abort, nextAction)
	})

}
