package backup

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestReconcilerEnsureCompletedBackupIsIgnored(t *testing.T) {
	t.Run("If the backup has failed then abort", func(t *testing.T) {
		t.Skip("TODO")
		backup := newBackupForControllerTest("ns", "backup")
		fakeClient := newFakeClientBuilder(t).Build()
		maintenanceGatewayMock := newMockMaintenanceGateway(t)
		providerBackupStatusMock := newMockProviderBackupStatus(t)
		providerBackupStatusMock.EXPECT().
			hasFailed(mock.Anything).
			Return(true)

		reconciler := NewReconciler(fakeClient, maintenanceGatewayMock, DefaultClock{}, providerBackupStatusMock)

		nextAction, err := reconciler.ensureCompletedBackupIsIgnored(context.Background(), backup, logr.Discard())

		assert.NoError(t, err)
		assert.Equal(t, Abort, nextAction)
	})

	t.Run("If the backup has succeeded then abort", func(t *testing.T) {
		t.Skip("TODO")
		backup := newBackupForControllerTest("ns", "backup")
		fakeClient := newFakeClientBuilder(t).Build()
		maintenanceGatewayMock := newMockMaintenanceGateway(t)
		providerBackupStatusMock := newMockProviderBackupStatus(t)
		providerBackupStatusMock.EXPECT().
			isCompleted(mock.Anything).
			Return(true)

		reconciler := NewReconciler(fakeClient, maintenanceGatewayMock, DefaultClock{}, providerBackupStatusMock)

		nextAction, err := reconciler.ensureCompletedBackupIsIgnored(context.Background(), backup, logr.Discard())

		assert.NoError(t, err)
		assert.Equal(t, Abort, nextAction)
	})

	t.Run("if the backup is in progress to the next step", func(t *testing.T) {
		t.Skip("TODO")
		backup := newBackupForControllerTest("ns", "backup")
		fakeClient := newFakeClientBuilder(t).Build()
		maintenanceGatewayMock := newMockMaintenanceGateway(t)
		providerBackupStatusMock := newMockProviderBackupStatus(t)
		providerBackupStatusMock.EXPECT().
			isInProgress(mock.Anything).
			Return(true)

		reconciler := NewReconciler(fakeClient, maintenanceGatewayMock, DefaultClock{}, providerBackupStatusMock)

		nextAction, err := reconciler.ensureCompletedBackupIsIgnored(context.Background(), backup, logr.Discard())

		assert.NoError(t, err)
		assert.Equal(t, Next, nextAction)
	})
}
