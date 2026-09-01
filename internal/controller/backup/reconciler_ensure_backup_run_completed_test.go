package backup

import (
	"context"
	"testing"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/errors"
	apiMeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestEnsureBackupRunCompleted(t *testing.T) {
	t.Run("It should wait while the backup run is still in progress", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		patches := 0
		fakeClient := newFakeClientBuilder(t).
			WithObjects(backup).
			WithStatusSubresource(backup).
			WithInterceptorFuncs(interceptor.Funcs{
				SubResourcePatch: func(
					ctx context.Context,
					c client.Client,
					subResourceName string,
					obj client.Object,
					patch client.Patch,
					opts ...client.SubResourcePatchOption,
				) error {
					return c.Status().Patch(ctx, obj, patch, opts...)
				},
			}).
			Build()
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default")

		_, outcome := reconciler.ensureBackupRunCompleted(context.Background(), backup)

		require.NoError(t, outcome.err)
		assert.Equal(t, actionRetry, outcome.action)
		assert.Zero(t, patches, "the terminal condition must not be written before the run is over")
	})

	t.Run("It should adopt the provider result of a succeeded run", func(t *testing.T) {
		backup := withProviderResult(newBackupForTest("ns", "backup"),
			metav1.ConditionTrue, reasonProviderBackupSucceeded, "Provider backup has succeeded.")
		recorder := record.NewFakeRecorder(100)
		fakeClient := newFakeClientBuilder(t).WithObjects(backup).WithStatusSubresource(backup).Build()
		reconciler := NewReconciler(fakeClient, recorder, nil, newRealClock(), "default")

		_, outcome := reconciler.ensureBackupRunCompleted(context.Background(), backup)

		require.NoError(t, outcome.err)
		assert.Equal(t, actionAbort, outcome.action)

		stored2 := &backupv1.Backup{}
		require.NoError(t, fakeClient.Get(context.Background(), backup.GetNamespacedName(), stored2))
		stored := stored2
		succeeded := apiMeta.FindStatusCondition(stored.Status.Conditions, backupv1.ConditionSucceeded)
		require.NotNil(t, succeeded)
		assert.Equal(t, metav1.ConditionTrue, succeeded.Status)
		assert.Equal(t, reasonProviderBackupSucceeded, succeeded.Reason)
		assert.Equal(t, "Provider backup has succeeded.", succeeded.Message)
		assert.Equal(t, backupv1.BackupStatusCompleted, stored.Status.Status) //nolint:staticcheck // legacy backup status compatibility

		require.Len(t, recorder.Events, 1)
		assert.Contains(t, <-recorder.Events, reasonBackupSucceeded)
	})

	t.Run("It should adopt the provider result of a failed run without reporting success", func(t *testing.T) {
		backup := withProviderResult(newBackupForTest("ns", "backup"),
			metav1.ConditionFalse, reasonProviderBackupFailed, "Provider backup has failed.")
		recorder := record.NewFakeRecorder(100)
		fakeClient := newFakeClientBuilder(t).WithObjects(backup).WithStatusSubresource(backup).Build()
		reconciler := NewReconciler(fakeClient, recorder, nil, newRealClock(), "default")

		_, outcome := reconciler.ensureBackupRunCompleted(context.Background(), backup)

		require.NoError(t, outcome.err)
		assert.Equal(t, actionAbort, outcome.action)

		stored2 := &backupv1.Backup{}
		require.NoError(t, fakeClient.Get(context.Background(), backup.GetNamespacedName(), stored2))
		stored := stored2
		succeeded := apiMeta.FindStatusCondition(stored.Status.Conditions, backupv1.ConditionSucceeded)
		require.NotNil(t, succeeded)
		assert.Equal(t, metav1.ConditionFalse, succeeded.Status)
		assert.Equal(t, reasonProviderBackupFailed, succeeded.Reason)
		assert.Equal(t, backupv1.BackupStatusFailed, stored.Status.Status) //nolint:staticcheck // legacy backup status compatibility

		assert.Empty(t, recorder.Events, "a failed run must not be reported as a success")
	})

	t.Run("It should complete a canceled run that never reached the provider", func(t *testing.T) {
		backup := withCondition(newBackupForTest("ns", "backup"), backupv1.ConditionCanceled, metav1.ConditionTrue)
		fakeClient := newFakeClientBuilder(t).WithObjects(backup).WithStatusSubresource(backup).Build()
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default")

		_, outcome := reconciler.ensureBackupRunCompleted(context.Background(), backup)

		require.NoError(t, outcome.err)
		assert.Equal(t, actionAbort, outcome.action)

		stored2 := &backupv1.Backup{}
		require.NoError(t, fakeClient.Get(context.Background(), backup.GetNamespacedName(), stored2))
		stored := stored2
		succeeded := apiMeta.FindStatusCondition(stored.Status.Conditions, backupv1.ConditionSucceeded)
		require.NotNil(t, succeeded)
		assert.Equal(t, metav1.ConditionFalse, succeeded.Status)
		assert.Equal(t, reasonBackupCanceled, succeeded.Reason,
			"without a provider result the cancellation is the outcome of the run")
	})

	t.Run("It should keep the run open with a wrapped error when the status write fails", func(t *testing.T) {
		backup := withProviderResult(newBackupForTest("ns", "backup"),
			metav1.ConditionTrue, reasonProviderBackupSucceeded, "Provider backup has succeeded.")
		patchErr := errors.NewInternalError(assert.AnError)
		fakeClient := newFakeClientBuilder(t).
			WithObjects(backup).
			WithStatusSubresource(backup).
			WithInterceptorFuncs(interceptor.Funcs{
				SubResourcePatch: func(
					ctx context.Context,
					c client.Client,
					subResourceName string,
					obj client.Object,
					patch client.Patch,
					opts ...client.SubResourcePatchOption,
				) error {
					return patchErr
				},
			}).
			Build()
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default")

		_, outcome := reconciler.ensureBackupRunCompleted(context.Background(), backup)

		assert.Equal(t, actionRetry, outcome.action)
		require.Error(t, outcome.err)
		assert.ErrorIs(t, outcome.err, patchErr)
		assert.ErrorContains(t, outcome.err, "patch status to complete the backup run")
	})
}

// withProviderResult marks the provider stage of the backup as finished, which is what makes the
// backup post-processing and lets ensureBackupRunCompleted derive the terminal condition from it.
func withProviderResult(
	backup *backupv1.Backup,
	status metav1.ConditionStatus,
	reason string,
	message string,
) *backupv1.Backup {
	apiMeta.SetStatusCondition(&backup.Status.Conditions, metav1.Condition{
		Type:    backupv1.ConditionProviderSucceeded,
		Status:  status,
		Reason:  reason,
		Message: message,
	})
	return backup
}
