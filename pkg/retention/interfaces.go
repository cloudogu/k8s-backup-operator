package retention

import (
	k8sv1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
	time2 "github.com/cloudogu/k8s-backup-operator/pkg/time"
)

// Strategy filters a set over all backups into sets of backups that should be removed by the RetentionManager
// as well as sets of backups that should be kept.
//
// Implementation Note:
// While Backup processes have a fixed start date, the actual run time of that process may be varying.
// Because of this, time-measuring implementations (i.e., keep backups for the last seven days) should
// consider the start date for counting backups.
// This would allow avoiding timing-related edge-cases of backups whose backup process may swap over to
// the next day.
type Strategy interface {
	// FilterForRemoval filters all backups which should or should not be removed by a RetentionManager.
	FilterForRemoval(allBackups []k8sv1.Backup) (RemovedBackups, RetainedBackups)

	// GetName returns a name of the Strategy implementation.
	// The name should roughly describe how the strategy works.
	// As the name may be used to be selected by the user,
	// the name must only consist of latin characters and ciphers.
	GetName() StrategyId
}

type timeProvider interface {
	time2.TimeProvider
}
