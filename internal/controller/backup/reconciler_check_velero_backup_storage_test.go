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
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestReconcilerCheckVeleroBackupStorage(t *testing.T) {
	t.Run("If the velero backup storage is unavailable set conditions and retry", func(t *testing.T) {
		backup := newBackupForControllerTest("ns", "backup")
		veleroBackupStorageLocation := newVeleroBackupStorageLocationForReconcilerTest(velerov1.BackupStorageLocationPhaseUnavailable)
		var patchCallCount = 0
		fakeClient := newFakeClientBuilder(t).
			WithObjects(backup, veleroBackupStorageLocation).
			WithStatusSubresource(backup).
			WithInterceptorFuncs(interceptor.Funcs{
				SubResourcePatch: func(ctx context.Context, client client.Client, subResourceName string, obj client.Object, patch client.Patch, opts ...client.SubResourcePatchOption) error {
					patchCallCount++
					return client.SubResource(subResourceName).Patch(ctx, obj, patch, opts...)
				},
			}).
			Build()
		reconciler := NewReconciler(fakeClient, nil)

		nextAction, err := reconciler.checkVeleroBackupStorage(context.Background(), backup, "ns", logr.Discard())

		assert.NoError(t, err)
		assert.Equal(t, Retry, nextAction)

		preparedCondition := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionPrepared)
		assert.NotNil(t, preparedCondition)
		assert.Equal(t, metav1.ConditionFalse, preparedCondition.Status)
		assert.Equal(t, reasonVeleroBackupStorageNotAvailable, preparedCondition.Reason)

		completedCondition := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionCompleted)
		assert.NotNil(t, completedCondition)
		assert.Equal(t, metav1.ConditionFalse, completedCondition.Status)
		assert.Equal(t, reasonPreparationNotCompleted, completedCondition.Reason)

		assert.Equal(t, 1, patchCallCount)
	})

	t.Run("If the velero backup storage resource was not found set the conditions and retry", func(t *testing.T) {
		backup := newBackupForControllerTest("ns", "backup")
		var patchCallCount = 0
		fakeClient := newFakeClientBuilder(t).
			WithObjects(backup).
			WithStatusSubresource(backup).
			WithInterceptorFuncs(interceptor.Funcs{
				SubResourcePatch: func(ctx context.Context, client client.Client, subResourceName string, obj client.Object, patch client.Patch, opts ...client.SubResourcePatchOption) error {
					patchCallCount++
					return client.SubResource(subResourceName).Patch(ctx, obj, patch, opts...)
				},
			}).
			Build()
		reconciler := NewReconciler(fakeClient, nil)

		nextAction, err := reconciler.checkVeleroBackupStorage(context.Background(), backup, "ns", logr.Discard())

		assert.Error(t, err)
		assert.Equal(t, Retry, nextAction)

		preparedCondition := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionPrepared)
		assert.NotNil(t, preparedCondition)
		assert.Equal(t, metav1.ConditionFalse, preparedCondition.Status)
		assert.Equal(t, reasonVeleroBackupStorageNotAvailable, preparedCondition.Reason)

		completedCondition := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionCompleted)
		assert.NotNil(t, completedCondition)
		assert.Equal(t, metav1.ConditionFalse, completedCondition.Status)
		assert.Equal(t, reasonPreparationNotCompleted, completedCondition.Reason)

		assert.Equal(t, 1, patchCallCount)
	})

	t.Run("If the velero backup storage is available set condition and proceed to the next step", func(t *testing.T) {
		backup := newBackupForControllerTest("ns", "backup")
		veleroBackupStorageLocation := newVeleroBackupStorageLocationForReconcilerTest(velerov1.BackupStorageLocationPhaseAvailable)
		var patchCallCount = 0
		fakeClient := newFakeClientBuilder(t).
			WithObjects(backup, veleroBackupStorageLocation).
			WithStatusSubresource(backup).
			WithInterceptorFuncs(interceptor.Funcs{
				SubResourcePatch: func(ctx context.Context, client client.Client, subResourceName string, obj client.Object, patch client.Patch, opts ...client.SubResourcePatchOption) error {
					patchCallCount++
					return client.SubResource(subResourceName).Patch(ctx, obj, patch, opts...)
				},
			}).
			Build()
		reconciler := NewReconciler(fakeClient, nil)

		nextAction, err := reconciler.checkVeleroBackupStorage(context.Background(), backup, "ns", logr.Discard())

		assert.NoError(t, err)
		assert.Equal(t, Next, nextAction)

		preparedCondition := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionPrepared)
		assert.NotNil(t, preparedCondition)
		assert.Equal(t, metav1.ConditionTrue, preparedCondition.Status)
		assert.Equal(t, reasonVeleroBackupStorageAvailable, preparedCondition.Reason)

		assert.Equal(t, 1, patchCallCount)
	})

	t.Run("If the velero backup storage is unavailable and a patch error occurred then abort", func(t *testing.T) {
		backup := newBackupForControllerTest("ns", "backup")
		veleroBackupStorageLocation := newVeleroBackupStorageLocationForReconcilerTest(velerov1.BackupStorageLocationPhaseUnavailable)
		fakeClient := newFakeClientBuilder(t).
			WithObjects(backup, veleroBackupStorageLocation).
			WithStatusSubresource(backup).
			WithInterceptorFuncs(interceptor.Funcs{
				SubResourcePatch: func(ctx context.Context, client client.Client, subResourceName string, obj client.Object, patch client.Patch, opts ...client.SubResourcePatchOption) error {
					return assert.AnError
				},
			}).
			Build()
		reconciler := NewReconciler(fakeClient, nil)

		nextAction, err := reconciler.checkVeleroBackupStorage(context.Background(), backup, "ns", logr.Discard())

		assert.Error(t, err)
		assert.Equal(t, Abort, nextAction)
	})

	t.Run("If the velero backup storage is available and a patch error occurred then abort", func(t *testing.T) {
		backup := newBackupForControllerTest("ns", "backup")
		veleroBackupStorageLocation := newVeleroBackupStorageLocationForReconcilerTest(velerov1.BackupStorageLocationPhaseAvailable)
		fakeClient := newFakeClientBuilder(t).
			WithObjects(backup, veleroBackupStorageLocation).
			WithStatusSubresource(backup).
			WithInterceptorFuncs(interceptor.Funcs{
				SubResourcePatch: func(ctx context.Context, client client.Client, subResourceName string, obj client.Object, patch client.Patch, opts ...client.SubResourcePatchOption) error {
					return assert.AnError
				},
			}).
			Build()
		reconciler := NewReconciler(fakeClient, nil)

		nextAction, err := reconciler.checkVeleroBackupStorage(context.Background(), backup, "ns", logr.Discard())

		assert.Error(t, err)
		assert.Equal(t, Abort, nextAction)
	})

	t.Run("If the velero backup storage resource was not found and a patch error occurred then abort", func(t *testing.T) {
		backup := newBackupForControllerTest("ns", "backup")
		fakeClient := newFakeClientBuilder(t).
			WithObjects(backup).
			WithStatusSubresource(backup).
			WithInterceptorFuncs(interceptor.Funcs{
				SubResourcePatch: func(ctx context.Context, client client.Client, subResourceName string, obj client.Object, patch client.Patch, opts ...client.SubResourcePatchOption) error {
					return assert.AnError
				},
			}).
			Build()
		reconciler := NewReconciler(fakeClient, nil)

		nextAction, err := reconciler.checkVeleroBackupStorage(context.Background(), backup, "ns", logr.Discard())

		assert.Error(t, err)
		assert.Equal(t, Abort, nextAction)

	})
}
