package backup

import (
	"context"
	"errors"
	"reflect"
	"testing"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestReconcilerCheckBackupDeletion(t *testing.T) {
	t.Run("If the backup is not deleted, set condition and proceed to the next step", func(t *testing.T) {
		backup := newBackupForControllerTest("ns", "backup")
		fakeClient := newFakeClientBuilder(t).
			WithObjects(backup).
			WithStatusSubresource(backup).
			Build()
		reconciler := NewReconciler(fakeClient, nil, DefaultClock{})

		require.True(t, backup.DeletionTimestamp.IsZero())

		nextAction, err := reconciler.checkBackupDeletion(context.Background(), backup, logr.Discard())

		assert.NoError(t, err)
		assert.Equal(t, Next, nextAction)

		completedCondition := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionDeleting)
		assert.NotNil(t, completedCondition)
		assert.Equal(t, metav1.ConditionFalse, completedCondition.Status)
		assert.Equal(t, reasonBackupNotDeleting, completedCondition.Reason)
	})

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
		reconciler := NewReconciler(fakeClient, nil, DefaultClock{})

		nextAction, err := reconciler.checkBackupDeletion(context.Background(), backup, logr.Discard())

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
		reconciler := NewReconciler(fakeClient, nil, DefaultClock{})

		nextAction, err := reconciler.checkBackupDeletion(context.Background(), backup, logr.Discard())

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
		reconciler := NewReconciler(fakeClient, nil, DefaultClock{})

		nextAction, err := reconciler.checkBackupDeletion(context.Background(), backup, logr.Discard())

		assert.NoError(t, err)
		assert.Equal(t, Retry, nextAction)

		// Do not create delete backup request
		assert.Equal(t, 0, createDeleteBackupRequestCallCount)
		assert.Equal(t, 1, patchStatusCallCount)
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

		reconciler := NewReconciler(fakeClient, nil, DefaultClock{})

		nextAction, err := reconciler.checkBackupDeletion(context.Background(), backup, logr.Discard())

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

		reconciler := NewReconciler(fakeClient, nil, DefaultClock{})

		nextAction, err := reconciler.checkBackupDeletion(context.Background(), backup, logr.Discard())

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

		reconciler := NewReconciler(fakeClient, nil, DefaultClock{})

		nextAction, err := reconciler.checkBackupDeletion(context.Background(), backup, logr.Discard())

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

		reconciler := NewReconciler(fakeClient, nil, DefaultClock{})

		nextAction, err := reconciler.checkBackupDeletion(context.Background(), backup, logr.Discard())

		assert.Error(t, err)
		assert.ErrorContains(t, err, "patch status error")
		assert.Equal(t, Abort, nextAction)
	})

}

func newDeletedBackupForReconcilerTest(namespace string, name string) *backupv1.Backup {
	backup := newBackupForControllerTest(namespace, name)
	backup.Finalizers = []string{backupv1.BackupFinalizer}
	deletionTimestamp := metav1.Now()
	backup.DeletionTimestamp = &deletionTimestamp
	return backup
}
