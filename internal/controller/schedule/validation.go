package schedule

import (
	"errors"
	"fmt"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/robfig/cron/v3"
)

type validator struct{}

func (v validator) Validate(schedule *backupv1.BackupSchedule) error {
	if schedule.Spec.Schedule == "" {
		return errors.New("schedule must not be empty")
	}

	_, err := cron.ParseStandard(schedule.Spec.Schedule)
	if err != nil {
		return fmt.Errorf("invalid cron schedule %q: %w", schedule.Spec.Schedule, err)
	}

	return nil
}
