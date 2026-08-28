package retention

import v1 "github.com/cloudogu/k8s-backup-lib/api/v1"

type keepAllStrategy struct{}

func (kas *keepAllStrategy) FilterForRemoval(allBackups []v1.Backup) (RemovedBackups, RetainedBackups) {
	return RemovedBackups{}, allBackups
}

func (kas *keepAllStrategy) GetName() StrategyId {
	return KeepAllStrategy
}
