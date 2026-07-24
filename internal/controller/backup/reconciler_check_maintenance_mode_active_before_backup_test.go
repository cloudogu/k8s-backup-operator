package backup

import (
	"context"
	"testing"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestReconcilerCheckMaintenanceModeActiveBeforeBackup(t *testing.T) {
	t.Run("If the maintenance mode is not active and the backup was not started "+
		"activate it, set condition, set start time and retry", func(t *testing.T) {
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
		maintenanceGatewayMock := newMockMaintenanceGateway(t)
		maintenanceGatewayMock.EXPECT().
			isMaintenanceModeActive(context.Background()).
			Return(false, nil)
		maintenanceGatewayMock.EXPECT().
			activateMaintenanceMode(context.Background(), maintenanceModeTitle, maintenanceModeText).
			Return(nil)
		reconciler := NewReconciler(fakeClient, maintenanceGatewayMock, DefaultClock{})

		nextAction, err := reconciler.checkMaintenanceModeActiveBeforeBackup(context.Background(), backup, "ns", logr.Discard())

		assert.NoError(t, err)
		assert.Equal(t, Retry, nextAction)

		completedCondition := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionCompleted)
		assert.NotNil(t, completedCondition)
		assert.Equal(t, metav1.ConditionFalse, completedCondition.Status)
		assert.Equal(t, reasonMaintenanceModesIsNotActive, completedCondition.Reason)

		assert.False(t, backup.Status.StartTimestamp.IsZero())

		assert.Equal(t, 1, patchCallCount)
	})

	t.Run("If the maintenance mode is not active and the backup was completed proceed to the next step", func(t *testing.T) {
		backup := newBackupForControllerTest("ns", "backup")
		// The backup was completed
		backup.Status.CompletionTimestamp = metav1.Now()
		fakeClient := newFakeClientBuilder(t).Build()
		maintenanceGatewayMock := newMockMaintenanceGateway(t)
		maintenanceGatewayMock.EXPECT().
			isMaintenanceModeActive(context.Background()).
			Return(false, nil)
		reconciler := NewReconciler(fakeClient, maintenanceGatewayMock, DefaultClock{})

		nextAction, err := reconciler.checkMaintenanceModeActiveBeforeBackup(context.Background(), backup, "ns", logr.Discard())

		assert.NoError(t, err)
		assert.Equal(t, Next, nextAction)
	})

	t.Run("If the maintenance mode is active, proceed to the next step", func(t *testing.T) {
		backup := newBackupForControllerTest("ns", "backup")
		fakeClient := newFakeClientBuilder(t).
			Build()
		maintenanceGatewayMock := newMockMaintenanceGateway(t)
		maintenanceGatewayMock.EXPECT().
			isMaintenanceModeActive(context.Background()).
			Return(true, nil)
		reconciler := NewReconciler(fakeClient, maintenanceGatewayMock, DefaultClock{})

		nextAction, err := reconciler.checkMaintenanceModeActiveBeforeBackup(context.Background(), backup, "ns", logr.Discard())

		assert.NoError(t, err)
		assert.Equal(t, Next, nextAction)
	})

	t.Run("Abort if the maintenance mode check failed", func(t *testing.T) {
		backup := newBackupForControllerTest("ns", "backup")
		fakeClient := newFakeClientBuilder(t).
			Build()
		maintenanceGatewayMock := newMockMaintenanceGateway(t)
		maintenanceGatewayMock.EXPECT().
			isMaintenanceModeActive(context.Background()).
			Return(false, assert.AnError)
		reconciler := NewReconciler(fakeClient, maintenanceGatewayMock, DefaultClock{})

		nextAction, err := reconciler.checkMaintenanceModeActiveBeforeBackup(context.Background(), backup, "ns", logr.Discard())

		assert.Error(t, err)
		assert.Equal(t, Abort, nextAction)
	})

	t.Run("If activation of maintenance mode failed, then abort", func(t *testing.T) {
		backup := newBackupForControllerTest("ns", "backup")
		fakeClient := newFakeClientBuilder(t).
			Build()
		maintenanceGatewayMock := newMockMaintenanceGateway(t)
		maintenanceGatewayMock.EXPECT().
			isMaintenanceModeActive(context.Background()).
			Return(false, nil)
		maintenanceGatewayMock.EXPECT().
			activateMaintenanceMode(context.Background(), maintenanceModeTitle, maintenanceModeText).
			Return(assert.AnError)
		reconciler := NewReconciler(fakeClient, maintenanceGatewayMock, DefaultClock{})

		nextAction, err := reconciler.checkMaintenanceModeActiveBeforeBackup(context.Background(), backup, "ns", logr.Discard())

		assert.Error(t, err)
		assert.Equal(t, Abort, nextAction)
	})

}
