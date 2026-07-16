package backup

import (
	"context"
	"testing"

	"github.com/cloudogu/k8s-registry-lib/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestDefaultMaintenanceGateway(t *testing.T) {
	t.Run("The call to 'isMaintenanceModeActive' should be delegated to 'GetStatus'", func(t *testing.T) {
		maintenanceModeAdapterMock := newMockMaintenanceModeAdapter(t)
		maintenanceModeAdapterMock.EXPECT().
			GetStatus(mock.Anything).
			Return(repository.MaintenanceModeDescription{}, true, nil)

		gateway := defaultMaintenanceGateway{maintenanceModeAdapter: maintenanceModeAdapterMock}

		isActive, err := gateway.isMaintenanceModeActive(context.Background())

		assert.NoError(t, err)
		assert.True(t, isActive)
	})

	t.Run("The call to 'activateMaintenanceMode' should be delegated to 'Activate'", func(t *testing.T) {
		maintenanceModeAdapterMock := newMockMaintenanceModeAdapter(t)
		maintenanceModeAdapterMock.EXPECT().
			Activate(mock.Anything, repository.MaintenanceModeDescription{Title: "title", Text: "text"}, false).
			Return(nil)

		gateway := defaultMaintenanceGateway{maintenanceModeAdapter: maintenanceModeAdapterMock}

		err := gateway.activateMaintenanceMode(context.Background(), "title", "text")

		assert.NoError(t, err)
	})

	t.Run("The call to 'deactivateMaintenanceMode' should be delegated to 'Deactivate'", func(t *testing.T) {
		maintenanceModeAdapterMock := newMockMaintenanceModeAdapter(t)
		maintenanceModeAdapterMock.EXPECT().
			Deactivate(mock.Anything, false).
			Return(nil)

		gateway := defaultMaintenanceGateway{maintenanceModeAdapter: maintenanceModeAdapterMock}

		err := gateway.deactivateMaintenanceMode(context.Background())

		assert.NoError(t, err)
	})
}
