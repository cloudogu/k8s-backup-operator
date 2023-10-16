package maintenance

import (
	"encoding/json"
	"fmt"
	"github.com/cloudogu/cesapp-lib/registry"
)

const registryKeyMaintenance = "maintenance"

type maintenanceSwitch struct {
	globalConfig registry.ConfigurationContext
}

type maintenanceRegistryObject struct {
	Title string `json:"title,omitempty"`
	Text  string `json:"text,omitempty"`
}

// New create a new instance of maintenanceSwitch.
func New(globalConfig registry.ConfigurationContext) *maintenanceSwitch {
	return &maintenanceSwitch{globalConfig: globalConfig}
}

// ActivateMaintenanceMode activates the maintenance mode with given title and text by writing in the global config.
func (ms *maintenanceSwitch) ActivateMaintenanceMode(title string, text string) error {
	value := maintenanceRegistryObject{
		Title: title,
		Text:  text,
	}

	marshal, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal maintenance globalConfig value object [%+v]: %w", value, err)
	}

	// TODO Check if maintenance mode is already active. If yes something other might happen. Requeue
	return ms.globalConfig.Set(registryKeyMaintenance, string(marshal))
}

// DeactivateMaintenanceMode deactivates the maintenance mode by deleting the maintenance key in the global config.
func (ms *maintenanceSwitch) DeactivateMaintenanceMode() error {
	return ms.globalConfig.Delete(registryKeyMaintenance)
}
