package maintenance

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudogu/k8s-backup-operator/pkg/requeue"
	"github.com/cloudogu/k8s-registry-lib/config"
)

const registryKeyMaintenance = "maintenance"

type maintenanceSwitch struct {
	globalConfigRepository globalConfigRepository
}

type maintenanceRegistryObject struct {
	Title string `json:"title,omitempty"`
	Text  string `json:"text,omitempty"`
}

// New create a new instance of maintenanceSwitch.
func New(globalConfigRepository globalConfigRepository) *maintenanceSwitch {
	return &maintenanceSwitch{globalConfigRepository: globalConfigRepository}
}

// ActivateMaintenanceMode activates the maintenance mode with given title and text by writing in the global config.
func (ms *maintenanceSwitch) ActivateMaintenanceMode(ctx context.Context, title string, text string) error {
	isActive, err := ms.isActive(ctx)
	if err != nil {
		return err
	}

	if isActive {
		return &requeue.GenericRequeueableError{
			ErrMsg: "maybe currently other critical processes running: requeue",
			Err:    fmt.Errorf("error: maintenance mode is active but should be inactive"),
		}
	}

	return ms.activate(ctx, title, text)
}

// DeactivateMaintenanceMode deactivates the maintenance mode by deleting the maintenance key in the global config.
func (ms *maintenanceSwitch) DeactivateMaintenanceMode(ctx context.Context) error {
	globalConfig, err := ms.globalConfigRepository.Get(ctx)
	if err != nil {
		return fmt.Errorf("failed to get global config: %w", err)
	}
	cfg := globalConfig.Delete(registryKeyMaintenance)
	_, err = ms.globalConfigRepository.Update(ctx, config.GlobalConfig{
		Config: cfg,
	})
	return err
}

func (ms *maintenanceSwitch) isActive(ctx context.Context) (bool, error) {
	globalConfig, err := ms.globalConfigRepository.Get(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get global config: %w", err)
	}

	_, exists := globalConfig.Get(registryKeyMaintenance)

	return exists, nil
}

func (ms *maintenanceSwitch) activate(ctx context.Context, title, text string) error {
	value := maintenanceRegistryObject{
		Title: title,
		Text:  text,
	}

	marshal, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal maintenance globalConfig value object [%+v]: %w", value, err)
	}

	globalConfig, err := ms.globalConfigRepository.Get(ctx)
	if err != nil {
		return fmt.Errorf("failed to get global config: %w", err)
	}
	cfg, err := globalConfig.Set(registryKeyMaintenance, config.Value(marshal))
	if err != nil {
		return fmt.Errorf("failed to set global config: %w", err)
	}
	_, err = ms.globalConfigRepository.Update(ctx, config.GlobalConfig{
		Config: cfg,
	})
	return err
}
