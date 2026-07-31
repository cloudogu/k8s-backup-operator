package restore

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewRestoreManager(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		clientMock := newMockK8sClient(t)

		// when
		manager := NewRestoreManager(clientMock, testNamespace, nil)

		// then
		require.NotNil(t, manager)
	})
}
