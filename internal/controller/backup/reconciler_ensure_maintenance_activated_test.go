package backup

import (
	"context"
	"testing"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestReconcilerEnsureMaintenanceActivated(t *testing.T) {
	t.Run("If the maintenance mode is active, proceed to the next step", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		fakeClient := newFakeClientBuilder(t).Build()
		maintenanceGatewayMock := newMockMaintenanceGateway(t)
		maintenanceGatewayMock.EXPECT().
			isMaintenanceModeActive(context.Background()).
			Return(true, nil)
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), maintenanceGatewayMock, newRealClock(), "default")

		nextAction, err := reconciler.ensureMaintenanceActivated(context.Background(), backup)

		assert.NoError(t, err)
		assert.Equal(t, Next, nextAction)
	})

	t.Run("If maintenance mode is not active and backup succeeded, abort", func(t *testing.T) {
		backup := newBackupWithSucceededStatusForReconcilerTest("ns", "backup", metav1.ConditionTrue)
		fakeClient := newFakeClientBuilder(t).Build()
		maintenanceGatewayMock := newMockMaintenanceGateway(t)
		maintenanceGatewayMock.EXPECT().
			isMaintenanceModeActive(context.Background()).
			Return(false, nil)
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), maintenanceGatewayMock, newRealClock(), "default")

		nextAction, err := reconciler.ensureMaintenanceActivated(context.Background(), backup)

		assert.NoError(t, err)
		assert.Equal(t, Abort, nextAction)
	})

	t.Run("If maintenance mode is not active and backup failed, abort", func(t *testing.T) {
		backup := newBackupWithSucceededStatusForReconcilerTest("ns", "backup", metav1.ConditionFalse)
		fakeClient := newFakeClientBuilder(t).Build()
		maintenanceGatewayMock := newMockMaintenanceGateway(t)
		maintenanceGatewayMock.EXPECT().
			isMaintenanceModeActive(context.Background()).
			Return(false, nil)
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), maintenanceGatewayMock, newRealClock(), "default")

		nextAction, err := reconciler.ensureMaintenanceActivated(context.Background(), backup)

		assert.NoError(t, err)
		assert.Equal(t, Abort, nextAction)
	})

	t.Run("If the maintenance mode is not active and the backup is not completed (failed or succeeded), "+
		"activate it, set succeeded to unknown and retry", func(t *testing.T) {
		backup := newBackupWithSucceededStatusForReconcilerTest("ns", "backup", metav1.ConditionUnknown)
		counter := &callCounter{}
		fakeClient := newFakeClientBuilderWithCounter(t, counter).
			WithObjects(backup).
			WithStatusSubresource(backup).
			Build()
		maintenanceGatewayMock := newMockMaintenanceGateway(t)
		maintenanceGatewayMock.EXPECT().
			isMaintenanceModeActive(context.Background()).
			Return(false, nil)
		maintenanceGatewayMock.EXPECT().
			activateMaintenanceMode(context.Background(), maintenanceModeTitle, maintenanceModeText).
			Return(nil)
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), maintenanceGatewayMock, newRealClock(), "default")

		nextAction, err := reconciler.ensureMaintenanceActivated(context.Background(), backup)

		assert.NoError(t, err)
		assert.Equal(t, Retry, nextAction)

		succeededCondition := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionSucceeded)
		assert.NotNil(t, succeededCondition)
		assert.Equal(t, metav1.ConditionUnknown, succeededCondition.Status)
		assert.Equal(t, reasonMaintenanceModesIsNotActive, succeededCondition.Reason)

		assert.Equal(t, 1, counter.subResourcePatchCount)
	})

	t.Run("Abort if the maintenance mode check failed", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		fakeClient := newFakeClientBuilder(t).
			Build()
		maintenanceGatewayMock := newMockMaintenanceGateway(t)
		maintenanceGatewayMock.EXPECT().
			isMaintenanceModeActive(context.Background()).
			Return(false, assert.AnError)
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), maintenanceGatewayMock, newRealClock(), "default")

		nextAction, err := reconciler.ensureMaintenanceActivated(context.Background(), backup)

		assert.Error(t, err)
		assert.Equal(t, Abort, nextAction)
	})

	t.Run("If activation of maintenance mode failed, then abort", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		fakeClient := newFakeClientBuilder(t).Build()
		maintenanceGatewayMock := newMockMaintenanceGateway(t)
		maintenanceGatewayMock.EXPECT().
			isMaintenanceModeActive(context.Background()).
			Return(false, nil)
		maintenanceGatewayMock.EXPECT().
			activateMaintenanceMode(context.Background(), maintenanceModeTitle, maintenanceModeText).
			Return(assert.AnError)
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), maintenanceGatewayMock, newRealClock(), "default")

		nextAction, err := reconciler.ensureMaintenanceActivated(context.Background(), backup)

		assert.Error(t, err)
		assert.Equal(t, Abort, nextAction)
	})

}
