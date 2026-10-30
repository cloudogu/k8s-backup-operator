package garbagecollection

import (
	"context"
	"errors"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/k8s-backup-operator/internal/retention"
)

type manager struct {
	client         k8sClient
	namespace      string
	strategyName   retention.StrategyId
	strategyGetter strategyGetter
}

// NewManager creates an instance of a Manager capable of deleting old backups.
func NewManager(client client.Client, namespace string, strategyName string) Manager {
	return &manager{
		client:         client,
		namespace:      namespace,
		strategyName:   retention.StrategyId(strategyName),
		strategyGetter: retention.NewStrategyGetter(),
	}
}

// CollectGarbage deletes backups according to the configured retention strategy.
func (m *manager) CollectGarbage(ctx context.Context) error {
	logger := log.FromContext(ctx).WithName("garbage-collection")

	var retentionStrategy strategy
	retentionStrategy, err := m.strategyGetter.Get(m.strategyName)
	if err != nil {
		return fmt.Errorf("failed to get retention strategy: %w", err)
	}

	logger.Info(fmt.Sprintf("using retention strategy %q to delete backups", retentionStrategy.GetName()))

	backupList := &v1.BackupList{}
	err = m.client.List(ctx, backupList, &client.ListOptions{Namespace: m.namespace})
	if err != nil {
		return fmt.Errorf("failed to list backups: %w", err)
	}

	completedBackups := filterCompleted(backupList.Items)

	toRemove, _ := retentionStrategy.FilterForRemoval(completedBackups)

	var errs []error
	for _, backup := range toRemove {
		err := m.client.Delete(ctx, &backup)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to delete backup %q: %w", backup.Name, err))
		}
	}

	err = errors.Join(errs...)
	if err != nil {
		return err
	}

	logger.Info(fmt.Sprintf("garbage collection completed: deleted %d backups", len(toRemove)))
	return nil
}

func filterCompleted(backups []v1.Backup) (completed []v1.Backup) {
	for _, backup := range backups {
		if backup.Status.Status == v1.BackupStatusCompleted { //nolint:staticcheck // legacy backup status compatibility
			completed = append(completed, backup)
		}
	}

	return completed
}
