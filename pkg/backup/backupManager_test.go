package backup

import (
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
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
		clientMock := newMockK8sClient(t)
		blueprintInterface := newMockBlueprintInterface(t)

		// when
		manager := NewBackupManager(clientMock, clientSetMock, blueprintInterface, testNamespace, nil, globalConfigRepositoryMock, ownerReferenceBackupMock)

		// then
		require.NotNil(t, manager)
	})
}
