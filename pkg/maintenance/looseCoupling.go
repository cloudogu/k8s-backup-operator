package maintenance

import (
	"context"
)

type looselyCoupledMaintenanceSwitch struct {
	maintenanceModeSwitch
}

// NewWithLooseCoupling creates a switch that checks if the configuration registry exists before switching.
// If the registry does not exist, no switch is executed.
func NewWithLooseCoupling(globalConfigRepository globalConfigRepository) *looselyCoupledMaintenanceSwitch {
	return &looselyCoupledMaintenanceSwitch{
		maintenanceModeSwitch: New(globalConfigRepository),
	}
}

// ActivateMaintenanceMode activates the maintenance mode if the global registry exists and is ready.
// This loose coupling enables us to perform restores on an empty cluster.
func (lcms *looselyCoupledMaintenanceSwitch) ActivateMaintenanceMode(ctx context.Context, title string, text string) error {
	return lcms.maintenanceModeSwitch.ActivateMaintenanceMode(ctx, title, text)
}

// DeactivateMaintenanceMode waits until the global registry is ready and then deactivates the maintenance mode.
// While this is not directly loose coupling, we trust that an instance of the global registry will be restored.
func (lcms *looselyCoupledMaintenanceSwitch) DeactivateMaintenanceMode(ctx context.Context) error {
	return lcms.maintenanceModeSwitch.DeactivateMaintenanceMode(ctx)
}
