package backup

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
)

func TestProviderBackupStatus(t *testing.T) {
	tests := []struct {
		phase      velerov1.BackupPhase
		failed     bool
		inProgress bool
		succeeded  bool
	}{
		// in progress
		{phase: velerov1.BackupPhaseNew, succeeded: false, failed: false, inProgress: true},
		{phase: velerov1.BackupPhaseInProgress, succeeded: false, failed: false, inProgress: true},
		{phase: velerov1.BackupPhaseFinalizing, succeeded: false, failed: false, inProgress: true},
		{phase: velerov1.BackupPhaseWaitingForPluginOperations, succeeded: false, failed: false, inProgress: true},

		// has failed
		{phase: velerov1.BackupPhaseFailedValidation, succeeded: false, failed: true, inProgress: false},
		{phase: velerov1.BackupPhaseWaitingForPluginOperationsPartiallyFailed, succeeded: false, failed: true, inProgress: false},
		{phase: velerov1.BackupPhaseFinalizingPartiallyFailed, succeeded: false, failed: true, inProgress: false},
		{phase: velerov1.BackupPhasePartiallyFailed, succeeded: false, failed: true, inProgress: false},
		{phase: velerov1.BackupPhaseFailed, succeeded: false, failed: true, inProgress: false},

		// succeeded
		{phase: velerov1.BackupPhaseCompleted, succeeded: true, failed: false, inProgress: false},

		// deleting
		{phase: velerov1.BackupPhaseDeleting, succeeded: false, failed: false, inProgress: false},
	}

	for _, test := range tests {
		name := fmt.Sprintf(
			"If the velero backup in in phase %s then the status should be succeeded: %t, failed: %t, inProgress: %t}",
			test.phase, test.succeeded, test.failed, test.inProgress)
		t.Run(name, func(t *testing.T) {
			backup := &velerov1.Backup{Status: velerov1.BackupStatus{Phase: test.phase}}
			assert.Equal(t, test.succeeded, hasProviderBackupSucceeded(backup))
			assert.Equal(t, test.failed, hasProviderBackupFailed(backup))
			assert.Equal(t, test.inProgress, isProviderBackupInProgress(backup))
		})
	}
}
