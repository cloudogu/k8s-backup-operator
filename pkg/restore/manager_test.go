package restore

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNewRestoreManager(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		registryMock := newMockCesRegistry(t)
		globalConfigMock := newMockConfigurationContext(t)
		registryMock.EXPECT().GlobalConfig().Return(globalConfigMock)

		// when
		manager := NewRestoreManager(nil, nil, nil, registryMock)

		// then
		require.NotNil(t, manager)
	})
}
