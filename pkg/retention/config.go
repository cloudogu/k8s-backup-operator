package retention

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/go-logr/logr"
)

const (
	retentionConfigmapName = "k8s-backup-operator-retention"
	strategyKey            = "strategy"
	DefaultStrategy        = KeepAllStrategy
)

type Config struct {
	Strategy StrategyId
}

type ConfigGetter struct {
	client configMapClient
}

func NewConfigGetter(client configMapClient) *ConfigGetter {
	return &ConfigGetter{client: client}
}

func (sg *ConfigGetter) GetConfig(ctx context.Context) (Config, error) {
	logger := log.FromContext(ctx, "GetConfig")

	configMap, err := sg.client.Get(ctx, retentionConfigmapName, metav1.GetOptions{})
	if err != nil {
		return Config{}, fmt.Errorf("failed to get retention config from config map %q: %w", retentionConfigmapName, err)
	}

	return configFromConfigMap(configMap, logger)
}

func configFromConfigMap(configMap *corev1.ConfigMap, logger logr.Logger) (Config, error) {
	config := Config{}

	strategy, exists := configMap.Data[strategyKey]
	if !exists {
		logger.Info(fmt.Sprintf("could not find key %q in config map %q", strategyKey, retentionConfigmapName))
		logger.Info(fmt.Sprintf("using default strategy %q", DefaultStrategy))
		config.Strategy = DefaultStrategy
	} else {
		err := validateStrategy(strategy)
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
