package backup

import (
	"context"
	"testing"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestReconcilerEnsureBackupRunCompleted(t *testing.T) {
	t.Run("A canceled run has to be reconciled again to get rid of its provider backup", func(t *testing.T) {
		backup := canceledBackupForReconcilerTest("ns", "backup")
		fakeClient := newFakeClientBuilder(t).
			WithObjects(backup).
			WithStatusSubresource(backup).
			Build()
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default")

		nextAction, err := reconciler.ensureBackupRunCompleted(context.Background(), backup)

		assert.NoError(t, err)
		assert.Equal(t, Retry, nextAction)

		succeededCondition := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionSucceeded)
		require.NotNil(t, succeededCondition)
		assert.Equal(t, metav1.ConditionFalse, succeededCondition.Status)
		assert.Equal(t, reasonBackupCanceled, succeededCondition.Reason)
	})

	t.Run("A completed run is done and needs no further reconciliation", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		meta.SetStatusCondition(&backup.Status.Conditions, metav1.Condition{
			Type:    backupv1.ConditionProviderSucceeded,
			Status:  metav1.ConditionTrue,
			Reason:  reasonProviderBackupSucceeded,
			Message: "The provider backup succeeded.",
		})
		fakeClient := newFakeClientBuilder(t).
			WithObjects(backup).
			WithStatusSubresource(backup).
			Build()
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default")

		nextAction, err := reconciler.ensureBackupRunCompleted(context.Background(), backup)

		assert.NoError(t, err)
		assert.Equal(t, Abort, nextAction)

		succeededCondition := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionSucceeded)
		require.NotNil(t, succeededCondition)
		assert.Equal(t, metav1.ConditionTrue, succeededCondition.Status)
	})

	t.Run("A run that is not finished yet is retried", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		fakeClient := newFakeClientBuilder(t).WithObjects(backup).WithStatusSubresource(backup).Build()
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default")

		nextAction, err := reconciler.ensureBackupRunCompleted(context.Background(), backup)

		assert.NoError(t, err)
		assert.Equal(t, Retry, nextAction)
		assert.Nil(t, meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionSucceeded))
	})
}
