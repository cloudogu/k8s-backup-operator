package schedule

import backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"

type validator struct{}

func (v *validator) Validate(*backupv1.BackupSchedule) error {
	return nil
}
