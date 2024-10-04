package maintenance

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

var testCtx = context.TODO()

func TestNewWithLooseCoupling(t *testing.T) {
	// when
	actual := NewWithLooseCoupling(nil)

	// then
	require.NotEmpty(t, actual)
}

func Test_looselyCoupledMaintenanceSwitch_ActivateMaintenanceMode(t *testing.T) {
	t.Run("should fail if activating the maintenance mode fails", func(t *testing.T) {
		// given
		maintenance := newMockMaintenanceModeSwitch(t)
		maintenance.EXPECT().ActivateMaintenanceMode(testCtx, "title", "text").Return(assert.AnError)

		sut := &looselyCoupledMaintenanceSwitch{
			maintenanceModeSwitch: maintenance,
		}

		// when
		err := sut.ActivateMaintenanceMode(testCtx, "title", "text")

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
	})
}

func Test_looselyCoupledMaintenanceSwitch_DeactivateMaintenanceMode(t *testing.T) {
	t.Run("should fail if deactivating maintenance fails", func(t *testing.T) {
		// given
		maintenance := newMockMaintenanceModeSwitch(t)
		maintenance.EXPECT().DeactivateMaintenanceMode(testCtx).Return(assert.AnError)

		sut := &looselyCoupledMaintenanceSwitch{
			maintenanceModeSwitch: maintenance,
		}

		// when
		err := sut.DeactivateMaintenanceMode(testCtx)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
	})
}
