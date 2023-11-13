package retention

import v1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"

type keepAllStrategy struct{}

func (kas *keepAllStrategy) FilterForRemoval(allBackups []v1.Backup) (RemovedBackups, RetainedBackups, error) {
	return RemovedBackups{}, allBackups, nil
}

func (kas *keepAllStrategy) GetName() StrategyId {
	return KeepAllStrategy
}
