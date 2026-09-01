package backup

import (
	"context"
	"testing"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestReconcilerEnsureMaintenanceDeactivated(t *testing.T) {
	t.Run("If the backup run is not finished, requeue without asking for the maintenance mode", func(t *testing.T) {
		backup := newBackupWithProviderSucceededStatusForReconcilerTest("ns", "backup", metav1.ConditionUnknown)
		fakeClient := newFakeClientBuilder(t).WithObjects(backup).Build()
		maintenanceGatewayMock := newMockMaintenanceGateway(t)
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), maintenanceGatewayMock, newRealClock(), "default")

		_, outcome := reconciler.ensureMaintenanceDeactivated(context.Background(), backup)

		assert.NoError(t, outcome.err)
		assert.Equal(t, actionRetry, outcome.action)
	})

	t.Run("If the backup is being deleted while its run is unfinished, deactivate the maintenance mode it holds", func(t *testing.T) {
		// A backup being deleted has no consistency left to protect and the deletion path is the last
		// reconcile that can give the maintenance mode back.
		backup := withDeletionTimestamp(newBackupWithProviderSucceededStatusForReconcilerTest("ns", "backup", metav1.ConditionUnknown))
		fakeClient := newFakeClientBuilder(t).WithObjects(backup, newHeldBackupLeaseForTest(backup)).Build()
		maintenanceGatewayMock := newMockMaintenanceGateway(t)
		maintenanceGatewayMock.EXPECT().
			isMaintenanceModeActive(context.Background()).
			Return(true, nil)
		maintenanceGatewayMock.EXPECT().
			deactivateMaintenanceMode(context.Background()).
			Return(nil)
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), maintenanceGatewayMock, newRealClock(), "default")

		_, outcome := reconciler.ensureMaintenanceDeactivated(context.Background(), backup)

		assert.NoError(t, outcome.err)
		assert.Equal(t, actionNext, outcome.action)
	})

	t.Run("If the backup is being deleted without holding the backup lease, leave the maintenance mode alone", func(t *testing.T) {
		backup := withDeletionTimestamp(newBackupWithProviderSucceededStatusForReconcilerTest("ns", "backup", metav1.ConditionUnknown))
		otherBackup := newBackupForTest("ns", "other-backup")
		fakeClient := newFakeClientBuilder(t).
			WithObjects(backup, otherBackup, newHeldBackupLeaseForTest(otherBackup)).
			Build()
		maintenanceGatewayMock := newMockMaintenanceGateway(t)
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), maintenanceGatewayMock, newRealClock(), "default")

		_, outcome := reconciler.ensureMaintenanceDeactivated(context.Background(), backup)

		assert.NoError(t, outcome.err)
		assert.Equal(t, actionNext, outcome.action)
	})

	t.Run("If the backup does not hold the backup lease, proceed without touching the maintenance mode", func(t *testing.T) {
		// The maintenance mode belongs to the lease holder. A backup that owns no lease must never
		// switch off the maintenance mode of a concurrently running backup.
		backup := newBackupWithProviderSucceededStatusForReconcilerTest("ns", "backup", metav1.ConditionTrue)
		otherBackup := newBackupForTest("ns", "other-backup")
		lease := newHeldBackupLeaseForTest(otherBackup)
		fakeClient := newFakeClientBuilder(t).WithObjects(backup, otherBackup, lease).Build()
		maintenanceGatewayMock := newMockMaintenanceGateway(t)
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), maintenanceGatewayMock, newRealClock(), "default")

		_, outcome := reconciler.ensureMaintenanceDeactivated(context.Background(), backup)

		assert.NoError(t, outcome.err)
		assert.Equal(t, actionNext, outcome.action)
	})

	t.Run("If there is no backup lease at all, proceed without touching the maintenance mode", func(t *testing.T) {
		backup := newBackupWithProviderSucceededStatusForReconcilerTest("ns", "backup", metav1.ConditionTrue)
		fakeClient := newFakeClientBuilder(t).WithObjects(backup).Build()
		maintenanceGatewayMock := newMockMaintenanceGateway(t)
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), maintenanceGatewayMock, newRealClock(), "default")

		_, outcome := reconciler.ensureMaintenanceDeactivated(context.Background(), backup)

		assert.NoError(t, outcome.err)
		assert.Equal(t, actionNext, outcome.action)
	})

	t.Run("If maintenance mode is not active, proceed to the next step", func(t *testing.T) {
		backup := newBackupWithProviderSucceededStatusForReconcilerTest("ns", "backup", metav1.ConditionTrue)
		fakeClient := newFakeClientBuilder(t).WithObjects(backup, newHeldBackupLeaseForTest(backup)).Build()
		maintenanceGatewayMock := newMockMaintenanceGateway(t)
		maintenanceGatewayMock.EXPECT().
			isMaintenanceModeActive(context.Background()).
			Return(false, nil)
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), maintenanceGatewayMock, newRealClock(), "default")

		_, outcome := reconciler.ensureMaintenanceDeactivated(context.Background(), backup)

		assert.NoError(t, outcome.err)
		assert.Equal(t, actionNext, outcome.action)
	})

	deactivatingRuns := []struct {
		name   string
		backup *backupv1.Backup
	}{
		{
			name:   "provider backup succeeded",
			backup: newBackupWithProviderSucceededStatusForReconcilerTest("ns", "backup", metav1.ConditionTrue),
		},
		{
			name:   "provider backup failed",
			backup: newBackupWithProviderSucceededStatusForReconcilerTest("ns", "backup", metav1.ConditionFalse),
		},
		{
			name:   "backup was canceled",
			backup: withCondition(newBackupForTest("ns", "backup"), backupv1.ConditionCanceled, metav1.ConditionTrue),
		},
	}
	for _, test := range deactivatingRuns {
		// Deactivating hands over to the lease release and the terminal condition in the same pass,
		// so this stage must not requeue.
		t.Run("If maintenance mode is active and the "+test.name+", deactivate it and proceed", func(t *testing.T) {
			fakeClient := newFakeClientBuilder(t).WithObjects(test.backup, newHeldBackupLeaseForTest(test.backup)).Build()
			maintenanceGatewayMock := newMockMaintenanceGateway(t)
			maintenanceGatewayMock.EXPECT().
				isMaintenanceModeActive(context.Background()).
				Return(true, nil)
			maintenanceGatewayMock.EXPECT().
				deactivateMaintenanceMode(context.Background()).
				Return(nil)
			reconciler := NewReconciler(fakeClient, newTestEventRecorder(), maintenanceGatewayMock, newRealClock(), "default")

			_, outcome := reconciler.ensureMaintenanceDeactivated(context.Background(), test.backup)

			assert.NoError(t, outcome.err)
			assert.Equal(t, actionNext, outcome.action)
		})
	}

	t.Run("If the maintenance mode state cannot be read, retry with error", func(t *testing.T) {
		backup := newBackupWithProviderSucceededStatusForReconcilerTest("ns", "backup", metav1.ConditionTrue)
		fakeClient := newFakeClientBuilder(t).WithObjects(backup, newHeldBackupLeaseForTest(backup)).Build()
		maintenanceGatewayMock := newMockMaintenanceGateway(t)
		maintenanceGatewayMock.EXPECT().
			isMaintenanceModeActive(context.Background()).
			Return(false, assert.AnError)
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), maintenanceGatewayMock, newRealClock(), "default")

		_, outcome := reconciler.ensureMaintenanceDeactivated(context.Background(), backup)

		assert.ErrorIs(t, outcome.err, assert.AnError)
		assert.Equal(t, actionRetry, outcome.action)
	})

	t.Run("If the maintenance mode cannot be deactivated, retry with error", func(t *testing.T) {
		backup := newBackupWithProviderSucceededStatusForReconcilerTest("ns", "backup", metav1.ConditionTrue)
		fakeClient := newFakeClientBuilder(t).WithObjects(backup, newHeldBackupLeaseForTest(backup)).Build()
		maintenanceGatewayMock := newMockMaintenanceGateway(t)
		maintenanceGatewayMock.EXPECT().
			isMaintenanceModeActive(context.Background()).
			Return(true, nil)
		maintenanceGatewayMock.EXPECT().
			deactivateMaintenanceMode(context.Background()).
			Return(assert.AnError)
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), maintenanceGatewayMock, newRealClock(), "default")

		_, outcome := reconciler.ensureMaintenanceDeactivated(context.Background(), backup)

		assert.ErrorIs(t, outcome.err, assert.AnError)
		assert.Equal(t, actionRetry, outcome.action)
	})
}
