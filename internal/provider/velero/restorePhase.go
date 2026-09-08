package velero

import (
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
)

// RestoreState is the provider-independent state of a provider restore.
type RestoreState string

const (
	// RestorePending means the provider accepted the restore but has not started working on it.
	RestorePending RestoreState = "Pending"
	// RestoreRunning means the provider is still working on the restore.
	RestoreRunning RestoreState = "Running"
	// RestoreSucceeded means the provider finished the restore successfully.
	RestoreSucceeded RestoreState = "Succeeded"
	// RestoreFailed means the provider finished the restore with a terminal failure, including
	// validation and partial failures.
	RestoreFailed RestoreState = "Failed"
	// RestoreStateUnknown means the provider reported something this operator does not know, for
	// example a phase a later Velero version added. It is never an outcome.
	RestoreStateUnknown RestoreState = "Unknown"
)

// ObserveRestorePhase maps a phase of the owned Velero restore to a RestoreState.
//
// A phase in which Velero is still executing yields RestoreRunning even when Velero already knows
// that parts of it failed: Velero moves on to a terminal phase itself, and reacting earlier would let
// this operator recover workloads while the provider still writes to the cluster. An unmapped or
// future phase is never reported as success or failure.
func ObserveRestorePhase(phase velerov1.RestorePhase) RestoreState {
	switch phase {
	case "", velerov1.RestorePhaseNew:
		return RestorePending
	case velerov1.RestorePhaseInProgress,
		velerov1.RestorePhaseWaitingForPluginOperations,
		velerov1.RestorePhaseWaitingForPluginOperationsPartiallyFailed,
		velerov1.RestorePhaseFinalizing,
		velerov1.RestorePhaseFinalizingPartiallyFailed:
		return RestoreRunning
	case velerov1.RestorePhaseCompleted:
		return RestoreSucceeded
	case velerov1.RestorePhaseFailedValidation,
		velerov1.RestorePhasePartiallyFailed,
		velerov1.RestorePhaseFailed:
		return RestoreFailed
	default:
		return RestoreStateUnknown
	}
}
