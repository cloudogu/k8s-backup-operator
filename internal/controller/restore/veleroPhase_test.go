package restore

import (
	"testing"

	"github.com/stretchr/testify/assert"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestObserveVeleroRestorePhaseMapsEveryPhaseOfTheSupportedVeleroVersion(t *testing.T) {
	tests := []struct {
		phase      velerov1.RestorePhase
		wantStatus metav1.ConditionStatus
		wantReason string
	}{
		{phase: "", wantStatus: metav1.ConditionUnknown, wantReason: ReasonVeleroRestorePending},
		{phase: velerov1.RestorePhaseNew, wantStatus: metav1.ConditionUnknown, wantReason: ReasonVeleroRestorePending},
		{phase: velerov1.RestorePhaseInProgress, wantStatus: metav1.ConditionUnknown, wantReason: ReasonVeleroRestoreRunning},
		{phase: velerov1.RestorePhaseWaitingForPluginOperations, wantStatus: metav1.ConditionUnknown, wantReason: ReasonVeleroRestoreRunning},
		{phase: velerov1.RestorePhaseWaitingForPluginOperationsPartiallyFailed, wantStatus: metav1.ConditionUnknown, wantReason: ReasonVeleroRestoreRunning},
		{phase: velerov1.RestorePhaseFinalizing, wantStatus: metav1.ConditionUnknown, wantReason: ReasonVeleroRestoreRunning},
		{phase: velerov1.RestorePhaseFinalizingPartiallyFailed, wantStatus: metav1.ConditionUnknown, wantReason: ReasonVeleroRestoreRunning},
		{phase: velerov1.RestorePhaseCompleted, wantStatus: metav1.ConditionTrue, wantReason: ReasonVeleroRestoreCompleted},
		{phase: velerov1.RestorePhaseFailedValidation, wantStatus: metav1.ConditionFalse, wantReason: ReasonVeleroRestoreFailed},
		{phase: velerov1.RestorePhasePartiallyFailed, wantStatus: metav1.ConditionFalse, wantReason: ReasonVeleroRestoreFailed},
		{phase: velerov1.RestorePhaseFailed, wantStatus: metav1.ConditionFalse, wantReason: ReasonVeleroRestoreFailed},
		{phase: velerov1.RestorePhase("SomePhaseAFutureVeleroAdded"), wantStatus: metav1.ConditionUnknown, wantReason: ReasonVeleroRestorePhaseUnknown},
	}

	for _, test := range tests {
		t.Run(string(test.phase), func(t *testing.T) {
			status, reason := observeVeleroRestorePhase(test.phase)

			assert.Equal(t, test.wantStatus, status)
			assert.Equal(t, test.wantReason, reason)
		})
	}
}

func TestObserveVeleroRestorePhaseNeverReportsAnUnhandledPhaseAsAnOutcome(t *testing.T) {
	unhandled := []velerov1.RestorePhase{"", "Unknown", "Deleting", "Succeeded", "someLowercasePhase"}

	for _, phase := range unhandled {
		status, _ := observeVeleroRestorePhase(phase)

		assert.Equal(t, metav1.ConditionUnknown, status, "phase %q must not be reported as an outcome", phase)
	}
}

// TestVeleroRestoreIsOnlyTerminalWhenVeleroStoppedWorking pins the interpretation the workflow
// relies on: terminality is derived from the condition status, so a phase in which Velero still
// runs must never produce True or False.
func TestVeleroRestoreIsOnlyTerminalWhenVeleroStoppedWorking(t *testing.T) {
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

	for _, phase := range stillRunning {
		status, _ := observeVeleroRestorePhase(phase)

		assert.Equal(t, metav1.ConditionUnknown, status, "phase %q is still running and must not be terminal", phase)
	}

	for _, phase := range stopped {
		status, _ := observeVeleroRestorePhase(phase)

		assert.NotEqual(t, metav1.ConditionUnknown, status, "phase %q stopped and must be terminal", phase)
	}
}
