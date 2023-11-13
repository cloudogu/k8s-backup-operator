package garbagecollection

import (
	"context"
	"errors"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/cloudogu/k8s-backup-operator/pkg/api/ecosystem"
	"github.com/cloudogu/k8s-backup-operator/pkg/retention"
)

type manager struct {
	clientSet ecosystemClientSet
	namespace string

	configGetter   configGetter
	strategyGetter strategyGetter
}

func NewManager(clientSet ecosystem.Interface, namespace string) Manager {
	configMapClient := clientSet.CoreV1().ConfigMaps(namespace)
	return &manager{
		clientSet:      clientSet,
		namespace:      namespace,
		configGetter:   retention.NewConfigGetter(configMapClient),
		strategyGetter: retention.NewStrategyGetter(),
	}
}

func (m *manager) CollectGarbage(ctx context.Context) error {
	logger := log.FromContext(ctx)

	retentionConfig, err := m.configGetter.GetConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to get retention config: %w", err)
	}

	retentionStrategy, err := m.strategyGetter.Get(retentionConfig.Strategy)
	if err != nil {
		return fmt.Errorf("failed to get retention strategy: %w", err)
	}

	logger.Info(fmt.Sprintf("using retention strategy %q to delete backups", retentionStrategy.GetName()))

	backupClient := m.clientSet.EcosystemV1Alpha1().Backups(m.namespace)
	backupList, err := backupClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list backups: %w", err)
	}

	toRemove, _, err := retentionStrategy.FilterForRemoval(backupList.Items)
	if err != nil {
		return fmt.Errorf("failed to filter backups for removal: %w", err)
	}

	var errs []error
	for _, backup := range toRemove {
		err := backupClient.Delete(ctx, backup.Name, metav1.DeleteOptions{})
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to delete backup %q: %w", backup.Name, err))
		}
	}

	return errors.Join(errs...)
}
