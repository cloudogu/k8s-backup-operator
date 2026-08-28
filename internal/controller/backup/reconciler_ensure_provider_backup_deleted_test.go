package backup

import (
	"context"
	"errors"
	"reflect"
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

func TestReconcilerEnsureProviderBackupDeleted(t *testing.T) {
	t.Run("If the backup is deleted and the velero backup does not exist, remove the finalizer and abort", func(t *testing.T) {
		backup := newDeletedBackupForReconcilerTest("ns", "backup")
		var veleroBackupGetCallCount = 0
		var backupUpdateCallCount = 0
		fakeClient := newFakeClientBuilder(t).
			WithObjects(backup).
			WithInterceptorFuncs(interceptor.Funcs{
				Get: func(ctx context.Context, client client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
					if reflect.TypeOf(obj) == reflect.TypeFor[*velerov1.Backup]() {
						veleroBackupGetCallCount++
					}
					return client.Get(ctx, key, obj, opts...)
				},
				Update: func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
					backupUpdateCallCount++
					return client.Update(ctx, obj, opts...)
				},
			}).
			Build()
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default")

		nextAction, err := reconciler.ensureProviderBackupDeleted(context.Background(), backup)

		assert.NoError(t, err)
		assert.Equal(t, Abort, nextAction)

		assert.Empty(t, backup.Finalizers)

		assert.Equal(t, 1, veleroBackupGetCallCount)
		assert.Equal(t, 1, backupUpdateCallCount)
	})

	t.Run("If the backup is deleted and the velero backup exists and no deletion request exits, "+
		"create deletion request, set condition and retry", func(t *testing.T) {
		backup := newDeletedBackupForReconcilerTest("ns", "backup")
		veleroBackup := newVeleroBackupForReconcilerTest("ns", "backup", velerov1.BackupPhaseCompleted)
		var patchCallCount = 0
		var veleroBackupGetCallCount = 0
		var createDeleteBackupRequestCallCount = 0
		fakeClient := newFakeClientBuilder(t).
			WithObjects(backup, veleroBackup).
			WithStatusSubresource(backup).
			WithInterceptorFuncs(interceptor.Funcs{
				Get: func(ctx context.Context, client client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
					if reflect.TypeOf(obj) == reflect.TypeFor[*velerov1.Backup]() {
						veleroBackupGetCallCount++
					}
					return client.Get(ctx, key, obj, opts...)
				},
				Create: func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
					if reflect.TypeOf(obj) == reflect.TypeFor[*velerov1.DeleteBackupRequest]() {
						createDeleteBackupRequestCallCount++
					}
					return client.Create(ctx, obj, opts...)
				},
				SubResourcePatch: func(ctx context.Context, client client.Client, subResourceName string, obj client.Object, patch client.Patch, opts ...client.SubResourcePatchOption) error {
					patchCallCount++
					return client.SubResource(subResourceName).Patch(ctx, obj, patch, opts...)
				},
			}).
			Build()
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default")

		nextAction, err := reconciler.ensureProviderBackupDeleted(context.Background(), backup)

		assert.NoError(t, err)
		assert.Equal(t, Retry, nextAction)

		assert.Contains(t, backup.Finalizers, backupv1.BackupFinalizer)

		assert.Equal(t, 1, createDeleteBackupRequestCallCount)

		completedCondition := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionDeleting)
		assert.NotNil(t, completedCondition)
		assert.Equal(t, metav1.ConditionTrue, completedCondition.Status)
		assert.Equal(t, reasonBackupDeleting, completedCondition.Reason)

		assert.Equal(t, 1, patchCallCount)
		assert.Equal(t, 1, veleroBackupGetCallCount)
	})

	t.Run("If the backup is deleted, the velero backup exists and the deletion request exits, retry", func(t *testing.T) {
		backup := newDeletedBackupForReconcilerTest("ns", "backup")
		veleroBackup := newVeleroBackupForReconcilerTest("ns", "backup", velerov1.BackupPhaseCompleted)
		deleteBackupRequest := &velerov1.DeleteBackupRequest{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: backup.Namespace,
				Name:      backup.Name,
			},
		}
		var patchStatusCallCount = 0
		var createDeleteBackupRequestCallCount = 0
		fakeClient := newFakeClientBuilder(t).
			WithObjects(backup, veleroBackup, deleteBackupRequest).
			WithStatusSubresource(backup).
			WithInterceptorFuncs(interceptor.Funcs{
				Create: func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
					if reflect.TypeOf(obj) == reflect.TypeFor[*velerov1.DeleteBackupRequest]() {
						createDeleteBackupRequestCallCount++
					}
					return client.Create(ctx, obj, opts...)
				},
				SubResourcePatch: func(ctx context.Context, client client.Client, subResourceName string, obj client.Object, patch client.Patch, opts ...client.SubResourcePatchOption) error {
					patchStatusCallCount++
					return client.SubResource(subResourceName).Patch(ctx, obj, patch, opts...)
				},
			}).
			Build()
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default")

		nextAction, err := reconciler.ensureProviderBackupDeleted(context.Background(), backup)

		assert.NoError(t, err)
		assert.Equal(t, Retry, nextAction)

		// Do not create delete backup request
		assert.Equal(t, 0, createDeleteBackupRequestCallCount)
		assert.Equal(t, 1, patchStatusCallCount)
	})

	t.Run("If the velero backup is in progress, remove an existing deletion request and wait for completion", func(t *testing.T) {
		backup := newDeletedBackupForReconcilerTest("ns", "backup")
		veleroBackup := newVeleroBackupForReconcilerTest("ns", "backup", velerov1.BackupPhaseInProgress)
		deleteBackupRequest := &velerov1.DeleteBackupRequest{ObjectMeta: metav1.ObjectMeta{Namespace: backup.Namespace, Name: backup.Name}}
		var createCallCount int
		var deleteCallCount int
		fakeClient := newFakeClientBuilder(t).
			WithObjects(backup, veleroBackup, deleteBackupRequest).
			WithStatusSubresource(backup).
			WithInterceptorFuncs(interceptor.Funcs{
				Create: func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
					if reflect.TypeOf(obj) == reflect.TypeFor[*velerov1.DeleteBackupRequest]() {
						createCallCount++
					}
					return client.Create(ctx, obj, opts...)
				},
				Delete: func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.DeleteOption) error {
					if reflect.TypeOf(obj) == reflect.TypeFor[*velerov1.DeleteBackupRequest]() {
						deleteCallCount++
					}
					return client.Delete(ctx, obj, opts...)
				},
			}).Build()
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default")

		nextAction, err := reconciler.ensureProviderBackupDeleted(context.Background(), backup)

		require.NoError(t, err)
		assert.Equal(t, Retry, nextAction)
		assert.Equal(t, 0, createCallCount)
		assert.Equal(t, 1, deleteCallCount)
		deletingCondition := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionDeleting)
		require.NotNil(t, deletingCondition)
		assert.Equal(t, metav1.ConditionTrue, deletingCondition.Status)
		assert.Equal(t, reasonWaitingForProviderBackupCompletion, deletingCondition.Reason)
		assert.Contains(t, deletingCondition.Message, "InProgress")
	})

	t.Run("If retrieving the Velero backup resource failed, abort.", func(t *testing.T) {
		backup := newDeletedBackupForReconcilerTest("ns", "backup")
		veleroBackup := newVeleroBackupForReconcilerTest("ns", "backup", velerov1.BackupPhaseCompleted)
		fakeClient := newFakeClientBuilder(t).
			WithObjects(veleroBackup).
			WithInterceptorFuncs(interceptor.Funcs{
				Get: func(ctx context.Context, client client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
					return errors.New("get error")
				},
			}).
			Build()

		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default")

		nextAction, err := reconciler.ensureProviderBackupDeleted(context.Background(), backup)

		assert.Error(t, err)
		assert.ErrorContains(t, err, "get error")
		assert.Equal(t, Abort, nextAction)
	})

	t.Run("If retrieving the delete backup request failed, abort.", func(t *testing.T) {
		backup := newDeletedBackupForReconcilerTest("ns", "backup")
		veleroBackup := newVeleroBackupForReconcilerTest("ns", "backup", velerov1.BackupPhaseCompleted)
		fakeClient := newFakeClientBuilder(t).
			WithObjects(veleroBackup).
			WithInterceptorFuncs(interceptor.Funcs{
				Get: func(ctx context.Context, client client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
					if reflect.TypeOf(obj) == reflect.TypeFor[*velerov1.DeleteBackupRequest]() {
						return errors.New("get error")
					}
					return client.Get(ctx, key, obj, opts...)
				},
			}).
			Build()

		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default")

		nextAction, err := reconciler.ensureProviderBackupDeleted(context.Background(), backup)

		assert.Error(t, err)
		assert.ErrorContains(t, err, "get error")
		assert.Equal(t, Abort, nextAction)
	})

	t.Run("If creating the delete backup request failed, abort.", func(t *testing.T) {
		backup := newDeletedBackupForReconcilerTest("ns", "backup")
		veleroBackup := newVeleroBackupForReconcilerTest("ns", "backup", velerov1.BackupPhaseCompleted)
		fakeClient := newFakeClientBuilder(t).
			WithObjects(veleroBackup).
			WithInterceptorFuncs(interceptor.Funcs{
				Create: func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
					if reflect.TypeOf(obj) == reflect.TypeFor[*velerov1.DeleteBackupRequest]() {
						return errors.New("create error")
					}
					return client.Create(ctx, obj, opts...)
				},
			}).
			Build()

		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default")

		nextAction, err := reconciler.ensureProviderBackupDeleted(context.Background(), backup)

		assert.Error(t, err)
		assert.ErrorContains(t, err, "create error")
		assert.Equal(t, Abort, nextAction)
	})

	t.Run("If patching status the backup failed, abort.", func(t *testing.T) {
		backup := newDeletedBackupForReconcilerTest("ns", "backup")
		veleroBackup := newVeleroBackupForReconcilerTest("ns", "backup", velerov1.BackupPhaseCompleted)
		fakeClient := newFakeClientBuilder(t).
			WithObjects(backup, veleroBackup).
			WithStatusSubresource(backup).
			WithInterceptorFuncs(interceptor.Funcs{
				SubResourcePatch: func(ctx context.Context, client client.Client, subResourceName string, obj client.Object, patch client.Patch, opts ...client.SubResourcePatchOption) error {
					return errors.New("patch status error")
				},
			}).
			Build()

		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default")

		nextAction, err := reconciler.ensureProviderBackupDeleted(context.Background(), backup)

		assert.Error(t, err)
		assert.ErrorContains(t, err, "patch status error")
		assert.Equal(t, Abort, nextAction)
	})

}

func TestReconcilerReportsDeletionProgress(t *testing.T) {
	t.Run("renders the delete request phase and the elapsed deletion time into the condition", func(t *testing.T) {
		deletionStart := time.Date(2026, 8, 21, 5, 3, 0, 0, time.UTC)
		backup := newDeletedBackupForReconcilerTest("ns", "backup")
		// The deletion already started, so the wait is measured from that transition.
		backup.Status.Conditions = []metav1.Condition{{
			Type:               backupv1.ConditionDeleting,
			Status:             metav1.ConditionTrue,
			Reason:             reasonBackupDeleting,
			Message:            "Backup is deleting (phase: New, running for less than 1m)",
			LastTransitionTime: metav1.NewTime(deletionStart),
		}}
		veleroBackup := newVeleroBackupForReconcilerTest("ns", "backup", velerov1.BackupPhaseCompleted)
		deleteRequest := &velerov1.DeleteBackupRequest{
			ObjectMeta: metav1.ObjectMeta{Namespace: "ns", Name: "backup"},
			Status: velerov1.DeleteBackupRequestStatus{
				Phase:  velerov1.DeleteBackupRequestPhaseInProgress,
				Errors: []string{"provider error"},
			},
		}
		fakeClient := newFakeClientBuilder(t).
			WithObjects(backup, veleroBackup, deleteRequest).
			WithStatusSubresource(backup).
			Build()
		clock := NewMockClock(t)
		clock.EXPECT().Now().Return(deletionStart.Add(2 * time.Minute)).Once()
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, clock, "default")

		nextAction, err := reconciler.ensureProviderBackupDeleted(context.Background(), backup)

		require.NoError(t, err)
		assert.Equal(t, Retry, nextAction)

		deletingCondition := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionDeleting)
		require.NotNil(t, deletingCondition)
		assert.Equal(t, "Backup is deleting (phase: InProgress, running for 2m)", deletingCondition.Message)
	})
}

func newDeletedBackupForReconcilerTest(namespace string, name string) *backupv1.Backup {
	backup := newBackupForTest(namespace, name)
	backup.Finalizers = []string{backupv1.BackupFinalizer}
	deletionTimestamp := metav1.Now()
	backup.DeletionTimestamp = &deletionTimestamp
	return backup
}
