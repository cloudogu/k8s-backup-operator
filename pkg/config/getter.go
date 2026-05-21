package config

import (
	"context"
	"fmt"
	"strconv"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

const (
	configMapName     = "k8s-backup-operator-backup-config"
	retryTimeLimitKey = "retryTimeLimit"
)

type ConfigMapInterface interface {
	v1.ConfigMapInterface
}

type getter struct {
	configmapClient ConfigMapInterface
	namespace       string
}

func NewGetter(client ConfigMapInterface) Getter {
	return &getter{configmapClient: client}
}

type Getter interface {
	// GetRetryLimit returns the configuration value for the retrylimit as int.
	GetRetryLimit(context.Context) (int, error)
}

func (g *getter) GetRetryLimit(ctx context.Context) (int, error) {
	cm, err := g.configmapClient.Get(ctx, configMapName, metav1.GetOptions{})
	if err != nil {
		return 0, fmt.Errorf("failed to get config map [%s]: %w", configMapName, err)
	}

	backupRetryTimeLimitStr := cm.Data[retryTimeLimitKey]

	retryLimit, err := strconv.Atoi(backupRetryTimeLimitStr)
	if err != nil {
		return 0, fmt.Errorf("failed to convert [%s]: %w", backupRetryTimeLimitStr, err)
	}
	return retryLimit, nil
}
