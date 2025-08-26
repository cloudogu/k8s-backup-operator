package retention

import v1 "github.com/cloudogu/k8s-backup-lib/pkg/api/v1"

// RemovedBackups hold backups that are removed from the backup repository.
type RemovedBackups []v1.Backup

// RetainedBackups hold backups that continue to stay at the backup repository and are not removed.
type RetainedBackups []v1.Backup
