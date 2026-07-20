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
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestReconcilerCheckVeleroBackupCompletion(t *testing.T) {
	t.Run("If the velero backup is not completed, set condition and retry", func(t *testing.T) {
		backup := newBackupForControllerTest("ns", "backup")
		veleroBackup := newVeleroBackupForReconcilerTest("ns", "backup", velerov1.BackupPhaseInProgress)
		var patchCallCount = 0
		fakeClient := newFakeClientBuilder(t).
			WithObjects(backup, veleroBackup).
			WithStatusSubresource(backup).
			WithInterceptorFuncs(interceptor.Funcs{
				SubResourcePatch: func(ctx context.Context, client client.Client, subResourceName string, obj client.Object, patch client.Patch, opts ...client.SubResourcePatchOption) error {
					patchCallCount++
					return client.SubResource(subResourceName).Patch(ctx, obj, patch, opts...)
				},
			}).
			Build()
		reconciler := NewReconciler(fakeClient, nil)

		nextAction, err := reconciler.checkVeleroBackupCompletion(context.Background(), backup, "ns", logr.Discard())

		completedCondition := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionCompleted)
		assert.NotNil(t, completedCondition)
		assert.Equal(t, metav1.ConditionFalse, completedCondition.Status)
		assert.Equal(t, reasonVeleroBackupNotCompleted, completedCondition.Reason)

		assert.NoError(t, err)
		assert.Equal(t, Retry, nextAction)

		assert.Equal(t, 1, patchCallCount)
	})

	t.Run("If the velero backup is completed proceed to the next step", func(t *testing.T) {
		backup := newBackupForControllerTest("ns", "backup")
		veleroBackup := newVeleroBackupForReconcilerTest("ns", "backup", velerov1.BackupPhaseCompleted)
		fakeClient := newFakeClientBuilder(t).
			WithObjects(backup, veleroBackup).
			WithStatusSubresource(backup).
			WithInterceptorFuncs(interceptor.Funcs{
				SubResourcePatch: func(ctx context.Context, client client.Client, subResourceName string, obj client.Object, patch client.Patch, opts ...client.SubResourcePatchOption) error {
					assert.Fail(t, "Unexpected call to SubResourcePatch")
					return client.SubResource(subResourceName).Patch(ctx, obj, patch, opts...)
				},
			}).
			Build()
		reconciler := NewReconciler(fakeClient, nil)

		nextAction, err := reconciler.checkVeleroBackupCompletion(context.Background(), backup, "ns", logr.Discard())

		assert.NoError(t, err)
		assert.Equal(t, Next, nextAction)
	})

	t.Run("If retrieving the Velero backup resource failed, abort.", func(t *testing.T) {
		backup := newBackupForControllerTest("ns", "backup")
		fakeClient := newFakeClientBuilder(t).
			WithObjects(backup).
			WithStatusSubresource(backup).
			WithInterceptorFuncs(interceptor.Funcs{
				Get: func(ctx context.Context, client client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
					return errors.New("get error")
				},
			}).
			Build()
		reconciler := NewReconciler(fakeClient, nil)

		nextAction, err := reconciler.checkVeleroBackupCompletion(context.Background(), backup, "ns", logr.Discard())

		assert.Error(t, err)
		assert.ErrorContains(t, err, "get error")
		assert.Equal(t, Abort, nextAction)
	})

	t.Run("If retrieving the Velero backup resource failed, abort.", func(t *testing.T) {
		backup := newBackupForControllerTest("ns", "backup")
		fakeClient := newFakeClientBuilder(t).
			WithObjects(backup).
			WithStatusSubresource(backup).
			WithInterceptorFuncs(interceptor.Funcs{
				Get: func(ctx context.Context, client client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
					return errors.New("get error")
				},
			}).
			Build()
		reconciler := NewReconciler(fakeClient, nil)

		nextAction, err := reconciler.checkVeleroBackupCompletion(context.Background(), backup, "ns", logr.Discard())

		assert.Error(t, err)
		assert.ErrorContains(t, err, "get error")
		assert.Equal(t, Abort, nextAction)
	})

	t.Run("Abort if patching the status fails.", func(t *testing.T) {
		backup := newBackupForControllerTest("ns", "backup")
		veleroBackup := newVeleroBackupForReconcilerTest("ns", "backup", velerov1.BackupPhaseInProgress)
		fakeClient := newFakeClientBuilder(t).
			WithObjects(backup, veleroBackup).
			WithStatusSubresource(backup).
			WithInterceptorFuncs(interceptor.Funcs{
				SubResourcePatch: func(ctx context.Context, client client.Client, subResourceName string, obj client.Object, patch client.Patch, opts ...client.SubResourcePatchOption) error {
					return errors.New("patch error")
				},
			}).
			Build()
		reconciler := NewReconciler(fakeClient, nil)

		nextAction, err := reconciler.checkVeleroBackupCompletion(context.Background(), backup, "ns", logr.Discard())

		assert.Error(t, err)
		assert.ErrorContains(t, err, "patch error")
		assert.Equal(t, Abort, nextAction)
	})

}
