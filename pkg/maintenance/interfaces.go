package maintenance

import (
	"github.com/cloudogu/cesapp-lib/registry"
	appsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
)

type maintenanceModeSwitch interface {
	// ActivateMaintenanceMode activates the maintenance mode.
	ActivateMaintenanceMode(title string, text string) error
	// DeactivateMaintenanceMode deactivates the maintenance mode.
	DeactivateMaintenanceMode() error
}

type statefulSetInterface interface {
	appsv1.StatefulSetInterface
}

type globalConfig interface {
	registry.ConfigurationContext
}
