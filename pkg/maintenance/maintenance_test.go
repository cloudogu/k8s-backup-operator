package maintenance

import (
	"github.com/cloudogu/k8s-backup-operator/pkg/requeue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNew(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// when
		result := New(nil)

		// then
		require.NotNil(t, result)
	})
}

func Test_maintenanceSwitch_ActivateMaintenanceMode(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		globalConfigMock := newMockGlobalConfig(t)
		globalConfigMock.EXPECT().Exists("maintenance").Return(false, nil)
		globalConfigMock.EXPECT().Set("maintenance", "{\"title\":\"title\",\"text\":\"text\"}").Return(nil)
		sut := maintenanceSwitch{globalConfig: globalConfigMock}

		// when
		err := sut.ActivateMaintenanceMode("title", "text")

		// then
		require.NoError(t, err)
	})

	t.Run("should return error on failing exists", func(t *testing.T) {
		// given
		globalConfigMock := newMockGlobalConfig(t)
		globalConfigMock.EXPECT().Exists("maintenance").Return(false, assert.AnError)
		sut := maintenanceSwitch{globalConfig: globalConfigMock}

		// when
		err := sut.ActivateMaintenanceMode("title", "text")

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to check if maintenance mode is active")
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("should return a requeueable error is the maintenance mode is active before activation", func(t *testing.T) {
		// given
		globalConfigMock := newMockGlobalConfig(t)
		globalConfigMock.EXPECT().Exists("maintenance").Return(true, nil)
		sut := maintenanceSwitch{globalConfig: globalConfigMock}

		// when
		err := sut.ActivateMaintenanceMode("title", "text")

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "maybe currently other critical processes running: requeue: error: maintenance mode is active but should be inactive")
		assert.IsType(t, &requeue.GenericRequeueableError{}, err)
	})
}

func Test_maintenanceSwitch_DeactivateMaintenanceMode(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		globalConfigMock := newMockGlobalConfig(t)
		globalConfigMock.EXPECT().Delete("maintenance").Return(nil)
		sut := maintenanceSwitch{globalConfig: globalConfigMock}

		// when
		err := sut.DeactivateMaintenanceMode()

		// then
		require.NoError(t, err)
	})
}
