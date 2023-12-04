package restore

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewRestoreManager(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		registryMock := newMockCesRegistry(t)
		globalConfigMock := newMockConfigurationContext(t)
		registryMock.EXPECT().GlobalConfig().Return(globalConfigMock)

		statefulSetMock := newMockStatefulSetInterface(t)
		serviceMock := newMockServiceInterface(t)
		appsV1Mock := newMockAppsV1Interface(t)
		appsV1Mock.EXPECT().StatefulSets(testNamespace).Return(statefulSetMock)
		coreV1Mock := newMockCoreV1Interface(t)
		coreV1Mock.EXPECT().Services(testNamespace).Return(serviceMock)
		clientSetMock := newMockEcosystemInterface(t)
		clientSetMock.EXPECT().AppsV1().Return(appsV1Mock)
		clientSetMock.EXPECT().CoreV1().Return(coreV1Mock)

		// when
		manager := NewRestoreManager(clientSetMock, testNamespace, nil, registryMock, nil)

		// then
		require.NotNil(t, manager)
	})
}
