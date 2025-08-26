package retention

import (
	"slices"

	v1 "github.com/cloudogu/k8s-backup-lib/pkg/api/v1"
)

type removeAllButKeepLatestStrategy struct{}

func (kls *removeAllButKeepLatestStrategy) FilterForRemoval(allBackups []v1.Backup) (RemovedBackups, RetainedBackups) {
	if len(allBackups) == 0 {
		return RemovedBackups{}, RetainedBackups{}
	}

	var latestBackupIndex int
	for i, backup := range allBackups {
		moreRecent := backup.Status.StartTimestamp.Time.
			After(allBackups[latestBackupIndex].Status.StartTimestamp.Time)
		if moreRecent {
			latestBackupIndex = i
		}
	}

	// We'll have to create a copy here, since `slices.Delete` modifies the original slice.
	backupCopy := make([]v1.Backup, len(allBackups))
	copy(backupCopy, allBackups)
	removed := slices.Delete(backupCopy, latestBackupIndex, latestBackupIndex+1)
	retained := RetainedBackups{allBackups[latestBackupIndex]}
	return removed, retained
}

func (kls *removeAllButKeepLatestStrategy) GetName() StrategyId {
	return RemoveAllButKeepLatestStrategy
}
