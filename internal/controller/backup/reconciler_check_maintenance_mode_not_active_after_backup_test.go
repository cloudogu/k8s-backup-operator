package backup

import (
	"context"
	"errors"
	"reflect"
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

func TestReconcilerCheckMaintenanceModeNotActiveAfterBackup(t *testing.T) {
	t.Run("If maintenance mode is active after the velero backup completed, deactivate it, set condition and proceed to the next step", func(t *testing.T) {
		backup := newBackupForControllerTest("ns", "backup")
		veleroBackup := newVeleroBackupForReconcilerTest("ns", "backup", velerov1.BackupPhaseCompleted)
		var veleroBackupGetCallCount = 0
		var patchCallCount = 0
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
				SubResourcePatch: func(ctx context.Context, client client.Client, subResourceName string, obj client.Object, patch client.Patch, opts ...client.SubResourcePatchOption) error {
					patchCallCount++
					return client.SubResource(subResourceName).Patch(ctx, obj, patch, opts...)
				},
			}).
			Build()
		maintenanceGatewayMock := newMockMaintenanceGateway(t)
		maintenanceGatewayMock.EXPECT().
			isMaintenanceModeActive(context.Background()).
			Return(true, nil)
		maintenanceGatewayMock.EXPECT().
			deactivateMaintenanceMode(context.Background()).
			Return(nil)
		reconciler := NewReconciler(fakeClient, maintenanceGatewayMock)

		nextAction, err := reconciler.checkMaintenanceModeNotActiveAfterBackup(context.Background(), backup, "ns", logr.Discard())

		assert.NoError(t, err)
		assert.Equal(t, Next, nextAction)

		completedCondition := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionCompleted)
		assert.NotNil(t, completedCondition)
		assert.Equal(t, metav1.ConditionTrue, completedCondition.Status)
		assert.Equal(t, reasonBackupCompleted, completedCondition.Reason)

		assert.False(t, backup.Status.CompletionTimestamp.IsZero())

		assert.Equal(t, 1, veleroBackupGetCallCount)
		assert.Equal(t, 1, patchCallCount)
	})

	t.Run("Abort if the maintenance mode check failed", func(t *testing.T) {
		backup := newBackupForControllerTest("ns", "backup")
		veleroBackup := newVeleroBackupForReconcilerTest("ns", "backup", velerov1.BackupPhaseCompleted)
		fakeClient := newFakeClientBuilder(t).
			WithObjects(backup, veleroBackup).
			WithStatusSubresource(backup).
			Build()
		maintenanceGatewayMock := newMockMaintenanceGateway(t)
		maintenanceGatewayMock.EXPECT().
			isMaintenanceModeActive(context.Background()).
			Return(false, errors.New("gateway error"))
		reconciler := NewReconciler(fakeClient, maintenanceGatewayMock)

		nextAction, err := reconciler.checkMaintenanceModeNotActiveAfterBackup(context.Background(), backup, "ns", logr.Discard())

		assert.Error(t, err)
		assert.ErrorContains(t, err, "gateway error")
		assert.Equal(t, Abort, nextAction)
	})

	t.Run("Abort if retrieving the Velero backup resource fails", func(t *testing.T) {
		backup := newBackupForControllerTest("ns", "backup")
		veleroBackup := newVeleroBackupForReconcilerTest("ns", "backup", velerov1.BackupPhaseCompleted)
		fakeClient := newFakeClientBuilder(t).
			WithObjects(backup, veleroBackup).
			WithStatusSubresource(backup).
			WithInterceptorFuncs(interceptor.Funcs{
				Get: func(ctx context.Context, client client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
					if reflect.TypeOf(obj) == reflect.TypeFor[*velerov1.Backup]() {
						return errors.New("velero backup get error")
					}
					return client.Get(ctx, key, obj, opts...)
				},
			}).
			Build()
		maintenanceGatewayMock := newMockMaintenanceGateway(t)
		reconciler := NewReconciler(fakeClient, maintenanceGatewayMock)

		nextAction, err := reconciler.checkMaintenanceModeNotActiveAfterBackup(context.Background(), backup, "ns", logr.Discard())

		assert.Error(t, err)
		assert.ErrorContains(t, err, "velero backup get error")
		assert.Equal(t, Abort, nextAction)
	})

	t.Run("Abort if patching the status fails.", func(t *testing.T) {
		backup := newBackupForControllerTest("ns", "backup")
		veleroBackup := newVeleroBackupForReconcilerTest("ns", "backup", velerov1.BackupPhaseCompleted)
		fakeClient := newFakeClientBuilder(t).
			WithObjects(backup, veleroBackup).
			WithStatusSubresource(backup).
			WithInterceptorFuncs(interceptor.Funcs{
				SubResourcePatch: func(ctx context.Context, client client.Client, subResourceName string, obj client.Object, patch client.Patch, opts ...client.SubResourcePatchOption) error {
					return errors.New("patch error")
				},
			}).
			Build()
		maintenanceGatewayMock := newMockMaintenanceGateway(t)
		maintenanceGatewayMock.EXPECT().
			isMaintenanceModeActive(context.Background()).
			Return(true, nil)
		maintenanceGatewayMock.EXPECT().
			deactivateMaintenanceMode(context.Background()).
			Return(nil)
		reconciler := NewReconciler(fakeClient, maintenanceGatewayMock)

		nextAction, err := reconciler.checkMaintenanceModeNotActiveAfterBackup(context.Background(), backup, "ns", logr.Discard())

		assert.Error(t, err)
		assert.ErrorContains(t, err, "patch error")
		assert.Equal(t, Abort, nextAction)
	})

}
