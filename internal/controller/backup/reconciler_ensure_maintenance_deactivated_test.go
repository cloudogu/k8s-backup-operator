package backup

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestReconcilerEnsureMaintenanceDeactivated(t *testing.T) {
	t.Run("If maintenance mode is not active, proceed to the next step", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		fakeClient := newFakeClientBuilder(t).Build()
		maintenanceGatewayMock := newMockMaintenanceGateway(t)
		maintenanceGatewayMock.EXPECT().
			isMaintenanceModeActive(context.Background()).
			Return(false, nil)
		reconciler := NewReconciler(fakeClient, maintenanceGatewayMock, DefaultClock{}, nil)

		nextAction, err := reconciler.ensureMaintenanceDeactivated(context.Background(), backup, logr.Discard())

		assert.NoError(t, err)
		assert.Equal(t, Next, nextAction)
	})

	t.Run("If maintenance mode is active and backup succeeded, deactivate maintenance mode and retry", func(t *testing.T) {
		backup := newBackupWithSucceededStatusForReconcilerTest("ns", "backup", metav1.ConditionTrue)
		fakeClient := newFakeClientBuilder(t).
			WithObjects(backup).
			Build()
		maintenanceGatewayMock := newMockMaintenanceGateway(t)
		maintenanceGatewayMock.EXPECT().
			isMaintenanceModeActive(context.Background()).
			Return(true, nil)
		maintenanceGatewayMock.EXPECT().
			deactivateMaintenanceMode(context.Background()).
			Return(nil)
		reconciler := NewReconciler(fakeClient, maintenanceGatewayMock, DefaultClock{}, nil)

		nextAction, err := reconciler.ensureMaintenanceDeactivated(context.Background(), backup, logr.Discard())

		assert.NoError(t, err)
		assert.Equal(t, Retry, nextAction)
	})

	t.Run("If maintenance mode is active and backup failed, deactivate maintenance mode and retry", func(t *testing.T) {
		backup := newBackupWithSucceededStatusForReconcilerTest("ns", "backup", metav1.ConditionFalse)
		fakeClient := newFakeClientBuilder(t).
			WithObjects(backup).
			Build()
		maintenanceGatewayMock := newMockMaintenanceGateway(t)
		maintenanceGatewayMock.EXPECT().
			isMaintenanceModeActive(context.Background()).
			Return(true, nil)
		maintenanceGatewayMock.EXPECT().
			deactivateMaintenanceMode(context.Background()).
			Return(nil)
		reconciler := NewReconciler(fakeClient, maintenanceGatewayMock, DefaultClock{}, nil)

		nextAction, err := reconciler.ensureMaintenanceDeactivated(context.Background(), backup, logr.Discard())

		assert.NoError(t, err)
		assert.Equal(t, Retry, nextAction)
	})

	t.Run("If maintenance mode is active and backup is in progress, proceed to the next step", func(t *testing.T) {
		backup := newBackupWithSucceededStatusForReconcilerTest("ns", "backup", metav1.ConditionUnknown)
		fakeClient := newFakeClientBuilder(t).
			WithObjects(backup).
			Build()
		maintenanceGatewayMock := newMockMaintenanceGateway(t)
		maintenanceGatewayMock.EXPECT().
			isMaintenanceModeActive(context.Background()).
			Return(true, nil)
		reconciler := NewReconciler(fakeClient, maintenanceGatewayMock, DefaultClock{}, nil)

		nextAction, err := reconciler.ensureMaintenanceDeactivated(context.Background(), backup, logr.Discard())

		assert.NoError(t, err)
		assert.Equal(t, Next, nextAction)
	})
}
