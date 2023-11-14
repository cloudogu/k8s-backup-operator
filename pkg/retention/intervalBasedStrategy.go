package retention

import (
	"slices"

	v1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
)

func newIntervalBasedStrategy(name StrategyId, ic intervalCalendar, clock timeProvider) *intervalBasedStrategy {
	return &intervalBasedStrategy{name, ic, clock}
}

// intervalBasedStrategy can be used to change the storage behavior of backups for a given time period
// to enhance memory usage and backup speed.
type intervalBasedStrategy struct {
	name             StrategyId
	intervalCalendar intervalCalendar
	clock            timeProvider
}

func (ibs *intervalBasedStrategy) GetName() StrategyId {
	return ibs.name
}

func (ibs *intervalBasedStrategy) FilterForRemoval(allBackups []v1.Backup) (RemovedBackups, RetainedBackups) {
	var intervalBackupMappings = make(map[timeInterval][]v1.Backup)
	var removedBackups []v1.Backup
	var retainedBackups []v1.Backup

	for _, currentBackup := range allBackups {
		// create mapping of intervals and backups and add all backups that don't fit in any interval to removal
		intervalBackupMappings, removedBackups = ibs.mapBackupsToIntervals(intervalBackupMappings, removedBackups, currentBackup)
	}

	// filterByRetentionMode checks the interval retention modes and updates the lists as defined (see timeInterval)
	removedBackups, retainedBackups = filterByRetentionMode(intervalBackupMappings, removedBackups, retainedBackups)

	return removedBackups, retainedBackups
}

func (ibs *intervalBasedStrategy) mapBackupsToIntervals(intervalBackupMappings map[timeInterval][]v1.Backup, removedBackups RemovedBackups, currentBackup v1.Backup) (map[timeInterval][]v1.Backup, RemovedBackups) {
	now := ibs.clock.Now()
	// Use backup start time instead of end time so broken backups are handled as well
	currentBackupTimestamp := currentBackup.Status.StartTimestamp.Time

	var found = false
	for _, interval := range ibs.intervalCalendar.timeIntervals {
		if interval.isTimestampInInterval(currentBackupTimestamp, now) {
			if _, ok := intervalBackupMappings[interval]; ok {
				intervalBackupMappings[interval] = append(intervalBackupMappings[interval], currentBackup)
			} else {
				intervalBackupMappings[interval] = []v1.Backup{currentBackup}
			}
			found = true
		}
	}

	if !found {
		removedBackups = append(removedBackups, currentBackup)
	}
	return intervalBackupMappings, removedBackups
}

// check the interval retention mode (ALL = keep all; OLDEST = keep the oldest)
func filterByRetentionMode(intervalBackupMappings map[timeInterval][]v1.Backup, removedBackups RemovedBackups, retainedBackups RetainedBackups) (RemovedBackups, RetainedBackups) {

	for interval, backups := range intervalBackupMappings {
		var newRemoved, newRetained []v1.Backup

		if interval.retentionMode == keepAllIntervalMode {
			newRemoved, newRetained = retainAllBackups(backups)
		} else if interval.retentionMode == keepOldestIntervalMode {
			newRemoved, newRetained = retainOldestBackup(backups)
		}
		removedBackups = append(removedBackups, newRemoved...)
		retainedBackups = append(retainedBackups, newRetained...)
	}
	return removedBackups, retainedBackups
}

func retainOldestBackup(backups []v1.Backup) (RemovedBackups, RetainedBackups) {
	if len(backups) == 0 {
		return RemovedBackups{}, RetainedBackups{}
	}

	oldestIdx := 0
	for i, backup := range backups {
		isOlder := backup.Status.StartTimestamp.Time.Before(backups[oldestIdx].Status.StartTimestamp.Time)
		if isOlder {
			oldestIdx = i
		}
	}

	// we
	backupCopy := make([]v1.Backup, len(backups))
	copy(backupCopy, backups)

	removed := slices.Delete(backupCopy, oldestIdx, oldestIdx+1)
	retained := []v1.Backup{backups[oldestIdx]}
	return removed, retained
}

func retainAllBackups(backups []v1.Backup) (RemovedBackups, RetainedBackups) {
	return []v1.Backup{}, backups
}
