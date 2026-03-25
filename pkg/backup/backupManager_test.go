package backup

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewBackupManager(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		clientSetMock := newMockEcosystemInterface(t)
		clientMock := newMockK8sClient(t)
		blueprintInterface := newMockBlueprintInterface(t)

		// when
		manager := NewBackupManager(clientMock, clientSetMock, blueprintInterface, testNamespace, nil)

		// then
		require.NotNil(t, manager)
	})
}
