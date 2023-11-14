package retention

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

var testCtx = context.Background()

func TestNewConfigGetter(t *testing.T) {
	// given
	cmClient := newMockConfigMapClient(t)

	// when
	getter := NewConfigGetter(cmClient)

	// then
	assert.NotEmpty(t, getter)
}

func TestConfigGetter_GetConfig(t *testing.T) {
	t.Run("should fail to get configmap", func(t *testing.T) {
		// given
		cmClient := newMockConfigMapClient(t)
		cmClient.EXPECT().Get(testCtx, "k8s-backup-operator-retention", metav1.GetOptions{}).Return(nil, assert.AnError)
		sut := &ConfigGetter{cmClient}

		// when
		actual, err := sut.GetConfig(testCtx)

		// then
		assert.Empty(t, actual)
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to get retention config from config map \"k8s-backup-operator-retention\"")
	})
	t.Run("should use default strategy if configmap does not contain strategy", func(t *testing.T) {
		// given
		configMap := &corev1.ConfigMap{Data: make(map[string]string)}

		cmClient := newMockConfigMapClient(t)
		cmClient.EXPECT().Get(testCtx, "k8s-backup-operator-retention", metav1.GetOptions{}).Return(configMap, nil)
		sut := &ConfigGetter{cmClient}

		// when
		actual, err := sut.GetConfig(testCtx)

		// then
		require.NoError(t, err)
		require.NotEmpty(t, actual)
		assert.Equal(t, StrategyId("keepAll"), actual.Strategy)
	})
	t.Run("should fail on invalid strategy", func(t *testing.T) {
		// given
		configMap := &corev1.ConfigMap{Data: map[string]string{"strategy": "invalid"}}

		cmClient := newMockConfigMapClient(t)
		cmClient.EXPECT().Get(testCtx, "k8s-backup-operator-retention", metav1.GetOptions{}).Return(configMap, nil)
		sut := &ConfigGetter{cmClient}

		// when
		actual, err := sut.GetConfig(testCtx)

		// then
		require.Error(t, err)
		require.Empty(t, actual)
		assert.ErrorContains(t, err, "unknown retention strategy \"invalid\"")
	})
	t.Run("should succeed on valid strategy", func(t *testing.T) {
		// given
		configMap := &corev1.ConfigMap{Data: map[string]string{"strategy": "keepLastSevenDays"}}

		cmClient := newMockConfigMapClient(t)
		cmClient.EXPECT().Get(testCtx, "k8s-backup-operator-retention", metav1.GetOptions{}).Return(configMap, nil)
		sut := &ConfigGetter{cmClient}

		// when
		actual, err := sut.GetConfig(testCtx)

		// then
		require.NoError(t, err)
		require.NotEmpty(t, actual)
		assert.Equal(t, StrategyId("keepLastSevenDays"), actual.Strategy)
	})
}
