package retention

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	configmapName         = "k8s-backup-operator-retention"
	defaultConfigFilePath = "/config/retention"
	strategyKey           = "strategy"
	DefaultStrategy       = KeepAllStrategy
)

var defaultConfig = Config{Strategy: DefaultStrategy}

type Config struct {
	Strategy StrategyId
}

// ConfigGetter is capable of retrieving the retention configuration.
type ConfigGetter struct {
	configFilePath string
}

// NewConfigGetter creates something capable of retrieving the retention configuration.
func NewConfigGetter() *ConfigGetter {
	return &ConfigGetter{configFilePath: defaultConfigFilePath}
}

// GetConfig retrieves the retention configuration from a mounted configmap.
func (sg *ConfigGetter) GetConfig(ctx context.Context) (Config, error) {
	logger := log.FromContext(ctx, "GetConfig")

	if _, err := os.Stat(sg.configFilePath); errors.Is(err, os.ErrNotExist) {
		return Config{}, fmt.Errorf("failed to find retention configuration: %w", err)
	}

	strategyPath := filepath.Join(sg.configFilePath, strategyKey)
	if _, err := os.Stat(strategyPath); errors.Is(err, os.ErrNotExist) {
		logger.Info(fmt.Sprintf("could not find key %q in config map %q", strategyKey, configmapName))
		logger.Info(fmt.Sprintf("using default strategy %q", DefaultStrategy))
		return defaultConfig, nil
	}

	strategyBytes, err := os.ReadFile(strategyPath)
	if err != nil {
		return Config{}, fmt.Errorf("failed to read strategy: %w", err)
	}

	strategy := string(strategyBytes)

	err = validateStrategy(strategy)
	if err != nil {
		return Config{}, err
	}

	return Config{Strategy: StrategyId(strategy)}, nil
}

func validateStrategy(strategy string) error {
	switch StrategyId(strategy) {
	case KeepAllStrategy,
		RemoveAllButKeepLatestStrategy,
		KeepLastSevenDaysStrategy,
		KeepLast7DaysOldestOf1Month1Quarter1HalfYear1YearStrategy:
		return nil
	default:
		return fmt.Errorf("unknown retention strategy %q", strategy)
	}
}
