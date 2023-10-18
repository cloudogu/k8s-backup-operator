package maintenance

import (
	"github.com/cloudogu/cesapp-lib/registry"
	"github.com/cloudogu/k8s-backup-operator/pkg/api/ecosystem"
)

type maintenanceModeSwitch interface {
	// ActivateMaintenanceMode activates the maintenance mode.
	ActivateMaintenanceMode(title string, text string) error
	// DeactivateMaintenanceMode deactivates the maintenance mode.
	DeactivateMaintenanceMode() error
}

type ecosystemInterface interface {
	ecosystem.Interface
}

type globalConfig interface {
	registry.ConfigurationContext
}
