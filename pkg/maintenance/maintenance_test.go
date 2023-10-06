package maintenance

import (
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
		globalConfigMock.EXPECT().Set("maintenance", "{\"title\":\"title\",\"text\":\"text\"}").Return(nil)
		sut := maintenanceSwitch{globalConfig: globalConfigMock}

		// when
		err := sut.ActivateMaintenanceMode("title", "text")

		// then
		require.NoError(t, err)
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
