package schedule

import (
	"errors"
	"fmt"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/robfig/cron/v3"
)

type defaultValidator struct{}

// validate checks that the BackupSchedule contains a non-empty standard cron expression.
func (v defaultValidator) validate(schedule *backupv1.BackupSchedule) error {
	if schedule.Spec.Schedule == "" {
		return errors.New("schedule must not be empty")
	}

	_, err := cron.ParseStandard(schedule.Spec.Schedule)
	if err != nil {
		return fmt.Errorf("invalid cron schedule %q: %w", schedule.Spec.Schedule, err)
	}

	return nil
}
