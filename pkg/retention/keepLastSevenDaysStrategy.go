package retention

import (
	v1 "github.com/cloudogu/k8s-backup-lib/pkg/api/v1"
	"github.com/cloudogu/k8s-backup-operator/pkg/time"
)

func newKeepLastSevenDaysStrategy() Strategy {
	clock := &time.Clock{}
	return &keepLastSevenDaysStrategy{clock: clock}
}

type keepLastSevenDaysStrategy struct {
	clock timeProvider
}

func (klsds *keepLastSevenDaysStrategy) FilterForRemoval(allBackups []v1.Backup) (RemovedBackups, RetainedBackups) {
	var removedBackups []v1.Backup
	var retainedBackups []v1.Backup

	sevenDaysBefore := klsds.clock.Now().AddDate(0, 0, -7)
	for _, currentBackup := range allBackups {
		// Use backup start time instead of end time so broken backups are handled as well
		currentBackupTimestamp := currentBackup.Status.StartTimestamp.Time

		if currentBackupTimestamp.Before(sevenDaysBefore) {
			removedBackups = append(removedBackups, currentBackup)
		} else {
			retainedBackups = append(retainedBackups, currentBackup)
		}
	}

	return removedBackups, retainedBackups
}

func (klsds *keepLastSevenDaysStrategy) GetName() StrategyId {
	return KeepLastSevenDaysStrategy
}
