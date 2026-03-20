package backup

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewBackupManager(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		globalConfigRepositoryMock := newMockGlobalConfigRepository(t)
		clientSetMock := newMockEcosystemInterface(t)
		clientMock := newMockK8sClient(t)
		backupRetryTimeLimit := 10

		blueprintInterface := newMockBlueprintInterface(t)

		// when
		manager := NewBackupManager(clientMock, clientSetMock, blueprintInterface, testNamespace, nil, globalConfigRepositoryMock, backupRetryTimeLimit)

		// then
		require.NotNil(t, manager)
	})
}
