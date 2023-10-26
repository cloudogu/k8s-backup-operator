package maintenance

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudogu/k8s-backup-operator/pkg/requeue"
)

const registryKeyMaintenance = "maintenance"

type maintenanceSwitch struct {
	globalConfig globalConfig
}

type maintenanceRegistryObject struct {
	Title string `json:"title,omitempty"`
	Text  string `json:"text,omitempty"`
}

// New create a new instance of maintenanceSwitch.
func New(globalConfig globalConfig) *maintenanceSwitch {
	return &maintenanceSwitch{globalConfig: globalConfig}
}

// ActivateMaintenanceMode activates the maintenance mode with given title and text by writing in the global config.
func (ms *maintenanceSwitch) ActivateMaintenanceMode(_ context.Context, title string, text string) error {
	isActive, err := ms.isActive()
	if err != nil {
		return err
	}

	if isActive {
		return &requeue.GenericRequeueableError{
			ErrMsg: "maybe currently other critical processes running: requeue",
			Err:    fmt.Errorf("error: maintenance mode is active but should be inactive"),
		}
	}

	return ms.activate(title, text)
}

// DeactivateMaintenanceMode deactivates the maintenance mode by deleting the maintenance key in the global config.
func (ms *maintenanceSwitch) DeactivateMaintenanceMode(_ context.Context) error {
	return ms.globalConfig.Delete(registryKeyMaintenance)
}

func (ms *maintenanceSwitch) isActive() (bool, error) {
	exists, err := ms.globalConfig.Exists(registryKeyMaintenance)
	if err != nil {
		return false, fmt.Errorf("failed to check if maintenance mode is active: %w", err)
	}

	return exists, nil
}

func (ms *maintenanceSwitch) activate(title, text string) error {
	value := maintenanceRegistryObject{
		Title: title,
		Text:  text,
	}

	marshal, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal maintenance globalConfig value object [%+v]: %w", value, err)
	}

	return ms.globalConfig.Set(registryKeyMaintenance, string(marshal))
}
