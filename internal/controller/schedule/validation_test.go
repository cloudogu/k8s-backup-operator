package schedule

import (
	"testing"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/stretchr/testify/require"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name       string
		schedule   string
		wantErr    bool
		wantErrMsg string
	}{
		{
			name:     "valid schedule",
			schedule: "0 2 * * *",
			wantErr:  false,
		},
		{
			name:     "invalid text",
			schedule: "this-is not-a-cron-expression",
			wantErr:  true,
		},
		{
			name:       "empty schedule",
			wantErr:    true,
			wantErrMsg: "schedule must not be empty",
		},
	}

	v := defaultValidator{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schedule := &backupv1.BackupSchedule{
				Spec: backupv1.BackupScheduleSpec{
					Schedule: tt.schedule,
				},
			}

			err := v.validate(schedule)

			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrMsg != "" {
					require.EqualError(t, err, tt.wantErrMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}
