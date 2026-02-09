package restore

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewRestoreManager(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		clientSetMock := newMockEcosystemInterface(t)
		clientMock := newMockK8sClient(t)

		// when
		manager := NewRestoreManager(clientMock, clientSetMock, testNamespace, nil, nil)

		// then
		require.NotNil(t, manager)
	})
}
