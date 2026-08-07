package backup

import (
	"context"
	"testing"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestReconcilerEnsureBackupIsPrepared(t *testing.T) {
	t.Run("If the velero backup storage location was not found set prepared to false and retry", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		counter := &callCounter{}
		fakeClient := newFakeClientBuilderWithCounter(t, counter).
			WithObjects(backup).
			WithStatusSubresource(backup).
			Build()
		reconciler := NewReconciler(fakeClient, nil, DefaultClock{})

		nextAction, err := reconciler.ensureBackupIsPrepared(context.Background(), backup, logr.Discard())

		assert.NoError(t, err)
		assert.Equal(t, Retry, nextAction)

		preparedCondition := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionPrepared)
		assert.NotNil(t, preparedCondition)
		assert.Equal(t, metav1.ConditionFalse, preparedCondition.Status)
		assert.Equal(t, reasonProviderBackupStorageLocationNotFound, preparedCondition.Reason)

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
		reconciler := NewReconciler(fakeClient, nil, DefaultClock{})

		nextAction, err := reconciler.ensureBackupIsPrepared(context.Background(), backup, logr.Discard())

		assert.NoError(t, err)
		assert.Equal(t, Retry, nextAction)

		preparedCondition := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionPrepared)
		assert.NotNil(t, preparedCondition)
		assert.Equal(t, metav1.ConditionFalse, preparedCondition.Status)
		assert.Equal(t, reasonProviderBackupStorageLocationNotAvailable, preparedCondition.Reason)

		assert.Equal(t, 1, counter.subResourcePatchCount)
	})

	t.Run("If the velero backup storage is available set prepared to true and proceed to the next step", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		veleroBackupStorageLocation := newVeleroBackupStorageLocationForReconcilerTest(velerov1.BackupStorageLocationPhaseAvailable)
		counter := &callCounter{}
		fakeClient := newFakeClientBuilderWithCounter(t, counter).
			WithObjects(backup, veleroBackupStorageLocation).
			WithStatusSubresource(backup).
			Build()
		reconciler := NewReconciler(fakeClient, nil, DefaultClock{})

		nextAction, err := reconciler.ensureBackupIsPrepared(context.Background(), backup, logr.Discard())

		assert.NoError(t, err)
		assert.Equal(t, Next, nextAction)

		preparedCondition := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionPrepared)
		assert.NotNil(t, preparedCondition)
		assert.Equal(t, metav1.ConditionTrue, preparedCondition.Status)
		assert.Equal(t, reasonProviderBackupStorageLocationAvailable, preparedCondition.Reason)

		assert.Equal(t, 1, counter.subResourcePatchCount)
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
		reconciler := NewReconciler(fakeClient, nil, DefaultClock{})

		nextAction, err := reconciler.ensureBackupIsPrepared(context.Background(), backup, logr.Discard())

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
		reconciler := NewReconciler(fakeClient, nil, DefaultClock{})

		nextAction, err := reconciler.ensureBackupIsPrepared(context.Background(), backup, logr.Discard())

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
		reconciler := NewReconciler(fakeClient, nil, DefaultClock{})

		nextAction, err := reconciler.ensureBackupIsPrepared(context.Background(), backup, logr.Discard())

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
		reconciler := NewReconciler(fakeClient, nil, DefaultClock{})

		nextAction, err := reconciler.ensureBackupIsPrepared(context.Background(), backup, logr.Discard())

		assert.Error(t, err)
		assert.Equal(t, Abort, nextAction)

	})
}
