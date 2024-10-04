package maintenance

import (
	"github.com/cloudogu/k8s-backup-operator/pkg/requeue"
	"github.com/cloudogu/k8s-registry-lib/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNew(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// when
		result := New(nil)

		// then
		require.NotNil(t, result)
	})
}

func Test_maintenanceSwitch_ActivateMaintenanceMode(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		globalConfigRepositoryMock := newMockGlobalConfigRepository(t)
		cfgPrev := config.CreateConfig(make(config.Entries))
		gCfg := config.GlobalConfig{Config: cfgPrev}
		cfgAfter := config.CreateConfig(make(config.Entries))
		_, err := cfgAfter.Set("maintenance", "{\"title\":\"title\",\"text\":\"text\"}")
		assert.NoError(t, err)
		gCfgExpected := config.GlobalConfig{Config: cfgAfter}
		globalConfigRepositoryMock.EXPECT().Get(mock.Anything).Return(gCfg, nil)
		globalConfigRepositoryMock.EXPECT().Update(mock.Anything, mock.Anything).Return(gCfgExpected, nil)

		sut := maintenanceSwitch{globalConfigRepository: globalConfigRepositoryMock}

		// when
		err = sut.ActivateMaintenanceMode(testCtx, "title", "text")

		// then
		require.NoError(t, err)
	})

	t.Run("should return error on failing exists", func(t *testing.T) {
		// given
		globalConfigRepositoryMock := newMockGlobalConfigRepository(t)
		cfgPrev := config.CreateConfig(make(config.Entries))
		globalConfigRepositoryMock.EXPECT().Get(testCtx).Return(config.GlobalConfig{cfgPrev}, assert.AnError)
		sut := maintenanceSwitch{globalConfigRepository: globalConfigRepositoryMock}

		// when
		err := sut.ActivateMaintenanceMode(testCtx, "title", "text")

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get global config")
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("should return a requeueable error is the maintenance mode is active before activation", func(t *testing.T) {
		// given
		globalConfigRepositoryMock := newMockGlobalConfigRepository(t)
		cfgAfter := config.CreateConfig(make(config.Entries))
		_, err := cfgAfter.Set("maintenance", "{\"title\":\"title\",\"text\":\"text\"}")
		assert.NoError(t, err)
		gCfgExpected := config.GlobalConfig{Config: cfgAfter}
		globalConfigRepositoryMock.EXPECT().Get(testCtx).Return(gCfgExpected, nil)
		sut := maintenanceSwitch{globalConfigRepository: globalConfigRepositoryMock}

		// when
		err = sut.ActivateMaintenanceMode(testCtx, "title", "text")

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "maybe currently other critical processes running: requeue: error: maintenance mode is active but should be inactive")
		assert.IsType(t, &requeue.GenericRequeueableError{}, err)
	})
}

func Test_maintenanceSwitch_DeactivateMaintenanceMode(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		globalConfigRepositoryMock := newMockGlobalConfigRepository(t)
		cfgPrev := config.CreateConfig(make(config.Entries))
		_, err := cfgPrev.Set("maintenance", "{\"title\":\"title\",\"text\":\"text\"}")
		cfgAfter := config.CreateConfig(make(config.Entries))
		globalConfigRepositoryMock.EXPECT().Get(testCtx).Return(config.GlobalConfig{Config: cfgPrev}, nil)
		globalConfigRepositoryMock.EXPECT().Update(testCtx, mock.Anything).Return(config.GlobalConfig{Config: cfgAfter}, nil)
		sut := maintenanceSwitch{globalConfigRepository: globalConfigRepositoryMock}

		// when
		err = sut.DeactivateMaintenanceMode(testCtx)

		// then
		require.NoError(t, err)
	})
}
