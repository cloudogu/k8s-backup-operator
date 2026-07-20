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

func TestReconcilerCheckVeleroBackupResource(t *testing.T) {
	t.Run("If the velero backup resource does not exist, create it, set condition and retry", func(t *testing.T) {
		backup := newBackupForControllerTest("ns", "backup")
		var createCallCount = 0
		var patchCallCount = 0
		fakeClient := newFakeClientBuilder(t).
			WithObjects(backup).
			WithStatusSubresource(backup).
			WithInterceptorFuncs(interceptor.Funcs{
				Create: func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
					createCallCount++

					assert.IsType(t, obj, &velerov1.Backup{})
					assert.Equal(t, "ns", obj.GetNamespace())
					assert.Equal(t, "backup", obj.GetName())

					return client.Create(ctx, obj, opts...)
				},
				SubResourcePatch: func(ctx context.Context, client client.Client, subResourceName string, obj client.Object, patch client.Patch, opts ...client.SubResourcePatchOption) error {
					patchCallCount++
					return client.SubResource(subResourceName).Patch(ctx, obj, patch, opts...)
				},
			}).
			Build()
		reconciler := newReconciler(fakeClient, nil)

		nextAction, err := reconciler.checkVeleroBackupResource(context.Background(), backup, "ns", logr.Discard())

		completedCondition := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionCompleted)
		assert.NotNil(t, completedCondition)
		assert.Equal(t, metav1.ConditionFalse, completedCondition.Status)
		assert.Equal(t, reasonVeleroBackupResourceDoesNotExist, completedCondition.Reason)

		assert.NoError(t, err)
		assert.Equal(t, Retry, nextAction)

		assert.Equal(t, 1, patchCallCount)
		assert.Equal(t, 1, createCallCount)
	})

	t.Run("If the velero backup resource exists proceed to the next step", func(t *testing.T) {
		backup := newBackupForControllerTest("ns", "backup")
		veleroBackup := &velerov1.Backup{
			ObjectMeta: metav1.ObjectMeta{
				Name:      backup.Name,
				Namespace: backup.Namespace,
			},
		}
		fakeClient := newFakeClientBuilder(t).
			WithObjects(backup, veleroBackup).
			WithStatusSubresource(backup).
			WithInterceptorFuncs(interceptor.Funcs{
				Create: func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
					assert.Fail(t, "Unexpected call to Create")
					return nil
				},
				SubResourcePatch: func(ctx context.Context, client client.Client, subResourceName string, obj client.Object, patch client.Patch, opts ...client.SubResourcePatchOption) error {
					assert.Fail(t, "Unexpected call to SubResourcePatch")
					return nil
				},
			}).
			Build()
		reconciler := newReconciler(fakeClient, nil)

		nextAction, err := reconciler.checkVeleroBackupResource(context.Background(), backup, "ns", logr.Discard())

		assert.NoError(t, err)
		assert.Equal(t, Next, nextAction)
	})

	t.Run("If retrieving the Velero backup resource fails, abort.", func(t *testing.T) {
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
		reconciler := newReconciler(fakeClient, nil)

		nextAction, err := reconciler.checkVeleroBackupResource(context.Background(), backup, "ns", logr.Discard())

		assert.Error(t, err)
		assert.ErrorContains(t, err, "get error")
		assert.Equal(t, Abort, nextAction)
	})

	t.Run("Abort if patching the status fails.", func(t *testing.T) {
		backup := newBackupForControllerTest("ns", "backup")
		fakeClient := newFakeClientBuilder(t).
			WithObjects(backup).
			WithStatusSubresource(backup).
			WithInterceptorFuncs(interceptor.Funcs{
				SubResourcePatch: func(ctx context.Context, client client.Client, subResourceName string, obj client.Object, patch client.Patch, opts ...client.SubResourcePatchOption) error {
					return errors.New("patch error")
				},
			}).
			Build()
		reconciler := newReconciler(fakeClient, nil)

		nextAction, err := reconciler.checkVeleroBackupResource(context.Background(), backup, "ns", logr.Discard())

		assert.Error(t, err)
		assert.ErrorContains(t, err, "patch error")
		assert.Equal(t, Abort, nextAction)
	})

	t.Run("Abort if creating the Velero backup resource fails.", func(t *testing.T) {
		backup := newBackupForControllerTest("ns", "backup")
		fakeClient := newFakeClientBuilder(t).
			WithObjects(backup).
			WithStatusSubresource(backup).
			WithInterceptorFuncs(interceptor.Funcs{
				Create: func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
					return errors.New("create error")
				},
			}).
			Build()
		reconciler := newReconciler(fakeClient, nil)

		nextAction, err := reconciler.checkVeleroBackupResource(context.Background(), backup, "ns", logr.Discard())

		assert.Error(t, err)
		assert.ErrorContains(t, err, "create error")
		assert.Equal(t, Abort, nextAction)
	})
}
