package backup

import (
	"context"
	"errors"
	"testing"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
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
		reconciler := NewReconciler(fakeClient, nil, DefaultClock{}, nil)

		nextAction, err := reconciler.ensureProviderBackupCompleted(context.Background(), backup, logr.Discard())

		assert.NoError(t, err)
		assert.Equal(t, Retry, nextAction)

		succeededCondition := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionSucceeded)
		assert.NotNil(t, succeededCondition)
		assert.Equal(t, metav1.ConditionUnknown, succeededCondition.Status)
		assert.Equal(t, reasonProviderBackupInProgress, succeededCondition.Reason)

		assert.Equal(t, 1, counter.subResourcePatchCount)
	})

	t.Run("If the velero backup has failed, set succeeded to false and abort", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		veleroBackup := newVeleroBackupForReconcilerTest("ns", "backup", velerov1.BackupPhaseFailed)
		counter := &callCounter{}
		fakeClient := newFakeClientBuilderWithCounter(t, counter).
			WithObjects(backup, veleroBackup).
			WithStatusSubresource(backup).
			Build()
		reconciler := NewReconciler(fakeClient, nil, DefaultClock{}, nil)

		nextAction, err := reconciler.ensureProviderBackupCompleted(context.Background(), backup, logr.Discard())

		assert.NoError(t, err)
		assert.Equal(t, Abort, nextAction)

		succeededCondition := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionSucceeded)
		assert.NotNil(t, succeededCondition)
		assert.Equal(t, metav1.ConditionFalse, succeededCondition.Status)
		assert.Equal(t, reasonProviderBackupFailed, succeededCondition.Reason)

		assert.Equal(t, 1, counter.subResourcePatchCount)
	})

	t.Run("If the velero backup has succeed, proceed to the next step", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		veleroBackup := newVeleroBackupForReconcilerTest("ns", "backup", velerov1.BackupPhaseCompleted)
		counter := &callCounter{}
		fakeClient := newFakeClientBuilderWithCounter(t, counter).
			WithObjects(backup, veleroBackup).
			WithStatusSubresource(backup).
			Build()
		reconciler := NewReconciler(fakeClient, nil, DefaultClock{}, nil)

		nextAction, err := reconciler.ensureProviderBackupCompleted(context.Background(), backup, logr.Discard())

		assert.NoError(t, err)
		assert.Equal(t, Next, nextAction)

		succeededCondition := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionSucceeded)
		assert.NotNil(t, succeededCondition)
		assert.Equal(t, metav1.ConditionTrue, succeededCondition.Status)
		assert.Equal(t, reasonProviderBackupSucceeded, succeededCondition.Reason)

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
		reconciler := NewReconciler(fakeClient, nil, DefaultClock{}, nil)

		nextAction, err := reconciler.ensureProviderBackupCompleted(context.Background(), backup, logr.Discard())

		assert.Error(t, err)
		assert.Equal(t, Abort, nextAction)
	})

	t.Run("If retrieving the Velero backup resource failed, abort.", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		counter := &callCounter{
			getCallError: errors.New("get error"),
		}
		fakeClient := newFakeClientBuilderWithCounter(t, counter).
			WithObjects(backup).
			WithStatusSubresource(backup).
			Build()
		reconciler := NewReconciler(fakeClient, nil, DefaultClock{}, nil)

		nextAction, err := reconciler.ensureProviderBackupCompleted(context.Background(), backup, logr.Discard())

		assert.Error(t, err)
		assert.ErrorContains(t, err, "get error")
		assert.Equal(t, Abort, nextAction)
	})

	t.Run("Abort if patching the status fails.", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		veleroBackup := newVeleroBackupForReconcilerTest("ns", "backup", velerov1.BackupPhaseInProgress)
		counter := &callCounter{
			subResourcePatchCallError: errors.New("patch error"),
		}
		fakeClient := newFakeClientBuilderWithCounter(t, counter).
			WithObjects(backup, veleroBackup).
			WithStatusSubresource(backup).
			Build()
		reconciler := NewReconciler(fakeClient, nil, DefaultClock{}, nil)

		nextAction, err := reconciler.ensureProviderBackupCompleted(context.Background(), backup, logr.Discard())

		assert.Error(t, err)
		assert.ErrorContains(t, err, "patch error")
		assert.Equal(t, Abort, nextAction)
	})

}
