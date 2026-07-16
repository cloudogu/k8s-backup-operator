package backup

import (
	"context"

	"github.com/cloudogu/k8s-registry-lib/repository"
)

type maintenanceModeAdapter interface {
	GetStatus(ctx context.Context) (repository.MaintenanceModeDescription, bool, error)
	Activate(ctx context.Context, content repository.MaintenanceModeDescription, force bool) error
	Deactivate(ctx context.Context, force bool) error
}

type defaultMaintenanceGateway struct {
	maintenanceModeAdapter maintenanceModeAdapter
}

func (d defaultMaintenanceGateway) isMaintenanceModeActive(ctx context.Context) (bool, error) {
	_, isActive, err := d.maintenanceModeAdapter.GetStatus(ctx)
	return isActive, err
}

func (d defaultMaintenanceGateway) activateMaintenanceMode(ctx context.Context, title string, text string) error {
	return d.maintenanceModeAdapter.Activate(ctx, repository.MaintenanceModeDescription{Title: title, Text: text}, false)
}

func (d defaultMaintenanceGateway) deactivateMaintenanceMode(ctx context.Context) error {
	return d.maintenanceModeAdapter.Deactivate(ctx, false)
}
