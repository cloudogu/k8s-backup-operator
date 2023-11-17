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

type Config struct {
	Strategy StrategyId
}

type ConfigGetter struct {
	configFilePath string
}

func NewConfigGetter() *ConfigGetter {
	return &ConfigGetter{configFilePath: defaultConfigFilePath}
}

func (sg *ConfigGetter) GetConfig(ctx context.Context) (Config, error) {
	logger := log.FromContext(ctx, "GetConfig")

	if _, err := os.Stat(sg.configFilePath); errors.Is(err, os.ErrNotExist) {
		return Config{}, fmt.Errorf("failed to find retention configuration: %w", err)
	}

	config := Config{}
	strategyPath := filepath.Join(sg.configFilePath, strategyKey)
	if _, err := os.Stat(strategyPath); errors.Is(err, os.ErrNotExist) {
		logger.Info(fmt.Sprintf("could not find key %q in config map %q", strategyKey, configmapName))
		logger.Info(fmt.Sprintf("using default strategy %q", DefaultStrategy))
		config.Strategy = DefaultStrategy
	} else {
		strategyBytes, err := os.ReadFile(strategyPath)
		if err != nil {
			return Config{}, fmt.Errorf("failed to read strategy: %w", err)
		}

		strategy := string(strategyBytes)

		err = validateStrategy(strategy)
		if err != nil {
			return Config{}, err
		}

		config.Strategy = StrategyId(strategy)
	}

	return config, nil
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
