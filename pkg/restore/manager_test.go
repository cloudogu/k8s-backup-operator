package restore

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewRestoreManager(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		globalConfigRepositoryMock := newMockGlobalConfigRepository(t)
		clientSetMock := newMockEcosystemInterface(t)

		// when
		manager := NewRestoreManager(clientSetMock, testNamespace, nil, globalConfigRepositoryMock, nil)

		// then
		require.NotNil(t, manager)
	})
}
