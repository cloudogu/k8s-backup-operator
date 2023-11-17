package retention

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testCtx = context.Background()

func TestNewConfigGetter(t *testing.T) {
	// when
	getter := NewConfigGetter()

	// then
	assert.NotEmpty(t, getter)
}

func TestConfigGetter_GetConfig(t *testing.T) {
	t.Run("should fail to get configmap", func(t *testing.T) {
		// given
		tempDir := t.TempDir()
		nonexistant := filepath.Join(tempDir, "nonexistant")

		sut := &ConfigGetter{configFilePath: nonexistant}

		// when
		actual, err := sut.GetConfig(testCtx)

		// then
		assert.Empty(t, actual)
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to find retention configuration")
	})
	t.Run("should use default strategy if configmap does not contain strategy", func(t *testing.T) {
		// given
		tempDir := t.TempDir()

		sut := &ConfigGetter{configFilePath: tempDir}

		// when
		actual, err := sut.GetConfig(testCtx)

		// then
		require.NoError(t, err)
		require.NotEmpty(t, actual)
		assert.Equal(t, StrategyId("keepAll"), actual.Strategy)
	})
	t.Run("should fail on invalid strategy", func(t *testing.T) {
		// given
		tempDir := t.TempDir()
		strategyFile := filepath.Join(tempDir, "strategy")
		err := os.WriteFile(strategyFile, []byte("invalid"), 0644)
		require.NoError(t, err)

		sut := &ConfigGetter{configFilePath: tempDir}

		// when
		actual, err := sut.GetConfig(testCtx)

		// then
		require.Error(t, err)
		require.Empty(t, actual)
		assert.ErrorContains(t, err, "unknown retention strategy \"invalid\"")
	})
	t.Run("should succeed on valid strategy", func(t *testing.T) {
		// given
		tempDir := t.TempDir()
		strategyFile := filepath.Join(tempDir, "strategy")
		err := os.WriteFile(strategyFile, []byte("keepLastSevenDays"), 0644)
		require.NoError(t, err)

		sut := &ConfigGetter{configFilePath: tempDir}

		// when
		actual, err := sut.GetConfig(testCtx)

		// then
		require.NoError(t, err)
		require.NotEmpty(t, actual)
		assert.Equal(t, StrategyId("keepLastSevenDays"), actual.Strategy)
	})
}
