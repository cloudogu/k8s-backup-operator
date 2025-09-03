package garbagecollection

import (
	"context"
	"errors"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/cloudogu/k8s-backup-lib/api/ecosystem"
	v1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/k8s-backup-operator/pkg/retention"
)

type manager struct {
	clientSet      ecosystemClientSet
	namespace      string
	strategyName   retention.StrategyId
	strategyGetter strategyGetter
}

// NewManager creates an instance of a Manager capable of deleting old backups.
func NewManager(clientSet ecosystem.Interface, namespace string, strategyName string) Manager {
	return &manager{
		clientSet:      clientSet,
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

	backupClient := m.clientSet.EcosystemV1Alpha1().Backups(m.namespace)
	backupList, err := backupClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list backups: %w", err)
	}

	completedBackups := filterCompleted(backupList.Items)

	toRemove, _ := retentionStrategy.FilterForRemoval(completedBackups)

	var errs []error
	for _, backup := range toRemove {
		err := backupClient.Delete(ctx, backup.Name, metav1.DeleteOptions{})
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
		if backup.Status.Status == v1.BackupStatusCompleted {
			completed = append(completed, backup)
		}
	}

	return completed
}
