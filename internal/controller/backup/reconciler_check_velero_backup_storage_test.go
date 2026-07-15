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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestReconcilerCheckVeleroBackupStorage(t *testing.T) {
	t.Run("If the velero backup storage is not available set the 'prepared' condition to false and requeue", func(t *testing.T) {
		backup := newBackupForControllerTest("ns", "backup")
		veleroBackupStorageLocation := &velerov1.BackupStorageLocation{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "default",
			},
			Status: velerov1.BackupStorageLocationStatus{
				Phase: velerov1.BackupStorageLocationPhaseUnavailable,
			},
		}
		var subResourcePatched = false
		fakeClient := newFakeClientBuilder(t).
			WithObjects(backup, veleroBackupStorageLocation).
			WithStatusSubresource(backup).
			WithInterceptorFuncs(interceptor.Funcs{
				SubResourcePatch: func(ctx context.Context, client client.Client, subResourceName string, obj client.Object, patch client.Patch, opts ...client.SubResourcePatchOption) error {
					subResourcePatched = true
					return client.SubResource(subResourceName).Patch(ctx, obj, patch, opts...)
				},
			}).
			Build()
		reconciler := newReconciler(fakeClient)

		result, err := reconciler.checkVeleroBackupStorageLocation(context.Background(), backup, "ns", logr.Discard())

		assert.NoError(t, err)
		assert.Equal(t, defaultRequeueAfterTime, result.RequeueAfter)

		preparedCondition := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionPrepared)
		assert.NotNil(t, preparedCondition)
		assert.Equal(t, metav1.ConditionFalse, preparedCondition.Status)
		assert.Equal(t, veleroBackupStorageNotAvailable, preparedCondition.Reason)
		assert.True(t, subResourcePatched)
	})

	t.Run("If the velero backup storage resource was not found set the 'prepared' condition to false and requeue", func(t *testing.T) {
		backup := newBackupForControllerTest("ns", "backup")
		var subResourcePatched = false
		fakeClient := newFakeClientBuilder(t).
			WithObjects(backup).
			WithStatusSubresource(backup).
			WithInterceptorFuncs(interceptor.Funcs{
				SubResourcePatch: func(ctx context.Context, client client.Client, subResourceName string, obj client.Object, patch client.Patch, opts ...client.SubResourcePatchOption) error {
					subResourcePatched = true
					return client.SubResource(subResourceName).Patch(ctx, obj, patch, opts...)
				},
			}).
			Build()
		reconciler := newReconciler(fakeClient)

		result, err := reconciler.checkVeleroBackupStorageLocation(context.Background(), backup, "ns", logr.Discard())

		assert.Error(t, err)
		assert.Equal(t, defaultRequeueAfterTime, result.RequeueAfter)

		preparedCondition := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionPrepared)
		assert.NotNil(t, preparedCondition)
		assert.Equal(t, metav1.ConditionFalse, preparedCondition.Status)
		assert.Equal(t, veleroBackupStorageNotAvailable, preparedCondition.Reason)
		assert.True(t, subResourcePatched)
	})

	t.Run("If the velero backup storage is available set the 'prepared' condition to true and don't requeue", func(t *testing.T) {
		backup := newBackupForControllerTest("ns", "backup")
		veleroBackupStorageLocation := &velerov1.BackupStorageLocation{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "default",
			},
			Status: velerov1.BackupStorageLocationStatus{
				Phase: velerov1.BackupStorageLocationPhaseAvailable,
			},
		}
		var subResourcePatched = false
		fakeClient := newFakeClientBuilder(t).
			WithObjects(backup, veleroBackupStorageLocation).
			WithStatusSubresource(backup).
			WithInterceptorFuncs(interceptor.Funcs{
				SubResourcePatch: func(ctx context.Context, client client.Client, subResourceName string, obj client.Object, patch client.Patch, opts ...client.SubResourcePatchOption) error {
					subResourcePatched = true
					return client.SubResource(subResourceName).Patch(ctx, obj, patch, opts...)
				},
			}).
			Build()
		reconciler := newReconciler(fakeClient)

		result, err := reconciler.checkVeleroBackupStorageLocation(context.Background(), backup, "ns", logr.Discard())

		assert.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, result)

		preparedCondition := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionPrepared)
		assert.NotNil(t, preparedCondition)
		assert.Equal(t, metav1.ConditionTrue, preparedCondition.Status)
		assert.Equal(t, veleroBackupStorageAvailable, preparedCondition.Reason)
		assert.True(t, subResourcePatched)
	})
}
