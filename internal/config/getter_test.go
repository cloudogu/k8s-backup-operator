package config

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetRetryLimit(t *testing.T) {
	var testCtx = context.TODO()
	t.Run("should succeed", func(t *testing.T) {
		// given
		mockConfigMap := NewMockConfigMapInterface(t)

		cm := &corev1.ConfigMap{Data: map[string]string{retryTimeLimitKey: "10"}}

		mockConfigMap.EXPECT().Get(testCtx, configMapName, metav1.GetOptions{}).Return(cm, nil)
		g := NewGetter(mockConfigMap)

		// when
		limit, err := g.GetRetryLimit(testCtx)

		// then
		require.NoError(t, err)
		assert.Equal(t, 10, limit)
	})
	t.Run("should fail getting config map", func(t *testing.T) {
		// given
		mockConfigMap := NewMockConfigMapInterface(t)

		mockConfigMap.EXPECT().Get(testCtx, configMapName, metav1.GetOptions{}).Return(nil, assert.AnError)
		g := NewGetter(mockConfigMap)

		// when
		_, err := g.GetRetryLimit(testCtx)

		// then
		require.Error(t, err, "failed to get config map [k8s-backup-operator-backup-config]: assert.AnError general error for testing")
	})
	t.Run("should fail converting env var", func(t *testing.T) {
		// given
		mockConfigMap := NewMockConfigMapInterface(t)

		cm := &corev1.ConfigMap{Data: map[string]string{retryTimeLimitKey: "invalid"}}

		mockConfigMap.EXPECT().Get(testCtx, configMapName, metav1.GetOptions{}).Return(cm, nil)
		g := NewGetter(mockConfigMap)

		// when
		_, err := g.GetRetryLimit(testCtx)

		// then
		require.Error(t, err, "failed to convert [invalid]: strconv.Atoi: parsing \"invalid\": invalid syntax")
	})
}
