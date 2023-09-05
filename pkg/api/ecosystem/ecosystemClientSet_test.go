package ecosystem

import (
	"k8s.io/apimachinery/pkg/runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"k8s.io/client-go/rest"

	v1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
)

func TestNewClientSet(t *testing.T) {
	t.Run("should fail to add to scheme", func(t *testing.T) {
		// given
		oldAddToSchemeFunc := v1.AddToScheme
		defer func() { v1.AddToScheme = oldAddToSchemeFunc }()
		v1.AddToScheme = func(_ *runtime.Scheme) error {
			return assert.AnError
		}

		restCfg := &rest.Config{}

		// when
		actual, err := NewClientSet(restCfg, nil)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.Nil(t, actual)
	})
	t.Run("should fail to create rest client", func(t *testing.T) {
		// given
		restCfg := &rest.Config{}
		restCfg.Host = "\\"

		// when
		actual, err := NewClientSet(restCfg, nil)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "parse \"http://\\\\\": invalid character \"\\\\\" in host name")
		assert.Nil(t, actual)
	})
	t.Run("should succeed", func(t *testing.T) {
		// given
		restCfg := &rest.Config{}

		// when
		actual, err := NewClientSet(restCfg, nil)

		// then
		require.NoError(t, err)
		assert.NotNil(t, actual)
		assert.NotNil(t, actual.ecosystemV1Alpha1)
	})
}

func TestClientSet_EcosystemV1Alpha1(t *testing.T) {
	t.Run("should return getter", func(t *testing.T) {
		// given
		alpha1InterfaceMock := NewMockV1Alpha1Interface(t)
		sut := &ClientSet{ecosystemV1Alpha1: alpha1InterfaceMock}

		// when
		actual := sut.EcosystemV1Alpha1()

		// then
		assert.Same(t, alpha1InterfaceMock, actual)
	})
}

func TestV1Alpha1Client_Backups(t *testing.T) {
	t.Run("should return backup client", func(t *testing.T) {
		// given
		restClientMock := newMockRestInterface(t)
		sut := &V1Alpha1Client{restClient: restClientMock}

		// when
		actual := sut.Backups("ecosystem")

		// then
		require.NotNil(t, actual)
		assert.IsType(t, &backupClient{}, actual)
		assert.Same(t, restClientMock, actual.(*backupClient).client)
		assert.Equal(t, "ecosystem", actual.(*backupClient).ns)
	})
	t.Run("should return restore client", func(t *testing.T) {
		// given
		restClientMock := newMockRestInterface(t)
		sut := &V1Alpha1Client{restClient: restClientMock}

		// when
		actual := sut.Restores("ecosystem")

		// then
		require.NotNil(t, actual)
		assert.IsType(t, &restoreClient{}, actual)
		assert.Same(t, restClientMock, actual.(*restoreClient).client)
		assert.Equal(t, "ecosystem", actual.(*restoreClient).ns)
	})
}
