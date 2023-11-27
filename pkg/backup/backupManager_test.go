package backup

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNewBackupManager(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		registryMock := newMockEtcdRegistry(t)
		globalMock := newMockConfigurationContext(t)
		registryMock.EXPECT().GlobalConfig().Return(globalMock)

		// when
		manager := NewBackupManager(nil, testNamespace, nil, registryMock)

		// then
		require.NotNil(t, manager)
	})
}
