package backup

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNewBackupManager(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		globalConfigRepositoryMock := newMockGlobalConfigRepository(t)

		// when
		manager := NewBackupManager(nil, testNamespace, nil, globalConfigRepositoryMock)

		// then
		require.NotNil(t, manager)
	})
}
