package config

import (
	"context"
	"fmt"
	"strconv"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	configMapName     = "k8s-backup-operator-backup-config"
	retryTimeLimitKey = "retryTimeLimit"
)

type getter struct {
	configmapClient kubernetes.Interface
	namespace       string
}

func NewGetter(client kubernetes.Interface, namespace string) Getter {
	return &getter{configmapClient: client, namespace: namespace}
}

type Getter interface {
	// GetRetryLimit returns the configuration value for the retrylimit as int.
	GetRetryLimit(context.Context) (int, error)
}

func (g *getter) GetRetryLimit(ctx context.Context) (int, error) {
	cm, err := g.configmapClient.CoreV1().ConfigMaps(g.namespace).Get(ctx, configMapName, metav1.GetOptions{})
	if err != nil {
		return 0, fmt.Errorf("failed to get config map [%s]: %w", configMapName, err)
	}

	backupRetryTimeLimitStr := cm.Data[retryTimeLimitKey]

	retryLimit, err := strconv.Atoi(backupRetryTimeLimitStr)
	if err != nil {
		return 0, fmt.Errorf("failed to convert [%s]: %w", retryLimit, err)
	}
	return retryLimit, nil
}
