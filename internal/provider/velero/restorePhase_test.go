package velero

import (
	"testing"

	"github.com/stretchr/testify/assert"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
)

func TestObserveRestorePhaseMapsEveryPhaseOfTheSupportedVeleroVersion(t *testing.T) {
	tests := []struct {
		phase     velerov1.RestorePhase
		wantState RestoreState
	}{
		{phase: "", wantState: RestorePending},
		{phase: velerov1.RestorePhaseNew, wantState: RestorePending},
		{phase: velerov1.RestorePhaseInProgress, wantState: RestoreRunning},
		{phase: velerov1.RestorePhaseWaitingForPluginOperations, wantState: RestoreRunning},
		{phase: velerov1.RestorePhaseWaitingForPluginOperationsPartiallyFailed, wantState: RestoreRunning},
		{phase: velerov1.RestorePhaseFinalizing, wantState: RestoreRunning},
		{phase: velerov1.RestorePhaseFinalizingPartiallyFailed, wantState: RestoreRunning},
		{phase: velerov1.RestorePhaseCompleted, wantState: RestoreSucceeded},
		{phase: velerov1.RestorePhaseFailedValidation, wantState: RestoreFailed},
		{phase: velerov1.RestorePhasePartiallyFailed, wantState: RestoreFailed},
		{phase: velerov1.RestorePhaseFailed, wantState: RestoreFailed},
		{phase: velerov1.RestorePhase("SomePhaseAFutureVeleroAdded"), wantState: RestoreStateUnknown},
	}

	for _, test := range tests {
		t.Run(string(test.phase), func(t *testing.T) {
			assert.Equal(t, test.wantState, ObserveRestorePhase(test.phase))
		})
	}
}

func TestObserveRestorePhaseNeverReportsAnUnhandledPhaseAsAnOutcome(t *testing.T) {
	unhandled := []velerov1.RestorePhase{"", "Unknown", "Deleting", "Succeeded", "someLowercasePhase"}

	for _, phase := range unhandled {
		state := ObserveRestorePhase(phase)

		assert.NotEqual(t, RestoreSucceeded, state, "phase %q must not be reported as success", phase)
		assert.NotEqual(t, RestoreFailed, state, "phase %q must not be reported as failure", phase)
	}
}

// TestRestoreIsOnlyFinishedWhenVeleroStoppedWorking pins the interpretation the workflow relies on:
// a phase in which Velero still runs must never produce a terminal state.
func TestRestoreIsOnlyFinishedWhenVeleroStoppedWorking(t *testing.T) {
	stillRunning := []velerov1.RestorePhase{
		velerov1.RestorePhaseNew,
		velerov1.RestorePhaseInProgress,
		velerov1.RestorePhaseWaitingForPluginOperations,
		velerov1.RestorePhaseWaitingForPluginOperationsPartiallyFailed,
		velerov1.RestorePhaseFinalizing,
		velerov1.RestorePhaseFinalizingPartiallyFailed,
	}
	stopped := []velerov1.RestorePhase{
		velerov1.RestorePhaseCompleted,
		velerov1.RestorePhaseFailedValidation,
		velerov1.RestorePhasePartiallyFailed,
		velerov1.RestorePhaseFailed,
	}

	terminal := []RestoreState{RestoreSucceeded, RestoreFailed}
	for _, phase := range stillRunning {
		assert.NotContains(t, terminal, ObserveRestorePhase(phase),
			"phase %q is still running and must not be terminal", phase)
	}

	for _, phase := range stopped {
		assert.Contains(t, terminal, ObserveRestorePhase(phase),
			"phase %q stopped and must be terminal", phase)
	}
}
