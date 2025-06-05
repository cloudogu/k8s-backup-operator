package restore

import (
	"github.com/stretchr/testify/mock"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewRestoreManager(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		corev1Client := newMockCoreV1Interface(t)
		configMapMock := newMockConfigMapInterface(t)
		corev1Client.EXPECT().ConfigMaps(mock.Anything).Return(configMapMock)
		clientSetMock := newMockEcosystemInterface(t)
		clientSetMock.EXPECT().CoreV1().Return(corev1Client)
		ownerReferenceRestoreMock := newMockOwnerReferenceRestore(t)
		clientMock := newMockK8sClient(t)

		// when
		manager := NewRestoreManager(clientMock, clientSetMock, testNamespace, nil, nil, ownerReferenceRestoreMock)

		// then
		require.NotNil(t, manager)
	})
}
