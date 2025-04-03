package backup

import (
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNewBackupManager(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		ownerReferenceBackupMock := newMockOwnerReferenceBackup(t)
		globalConfigRepositoryMock := newMockGlobalConfigRepository(t)
		configMapMock := newMockBackupConfigMapInterface(t)
		corev1Client := newMockBackupCoreV1Interface(t)
		corev1Client.EXPECT().ConfigMaps(mock.Anything).Return(configMapMock)
		clientSetMock := newMockEcosystemInterface(t)
		clientSetMock.EXPECT().CoreV1().Return(corev1Client)

		// when
		manager := NewBackupManager(clientSetMock, testNamespace, nil, globalConfigRepositoryMock, ownerReferenceBackupMock)

		// then
		require.NotNil(t, manager)
	})
}
