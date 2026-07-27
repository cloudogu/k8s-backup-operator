package restore

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
)

// observeVeleroRestorePhase maps a phase of the owned Velero restore to the status and reason of
// the tri-state ProviderRestoreSuccessful condition.
//
// Unknown means that the provider has not decided yet, so an unmapped or future phase is never
// reported as success or failure. A phase in which Velero is still executing stays Unknown even
// when Velero already knows that parts of it failed: Velero moves on to a terminal phase itself,
// and reacting earlier would let this operator recover workloads while the provider still writes
// to the cluster.
func observeVeleroRestorePhase(phase velerov1.RestorePhase) (metav1.ConditionStatus, string) {
	switch phase {
	case "", velerov1.RestorePhaseNew:
		return metav1.ConditionUnknown, ReasonVeleroRestorePending
	case velerov1.RestorePhaseInProgress,
		velerov1.RestorePhaseWaitingForPluginOperations,
		velerov1.RestorePhaseWaitingForPluginOperationsPartiallyFailed,
		velerov1.RestorePhaseFinalizing,
		velerov1.RestorePhaseFinalizingPartiallyFailed:
		return metav1.ConditionUnknown, ReasonVeleroRestoreRunning
	case velerov1.RestorePhaseCompleted:
		return metav1.ConditionTrue, ReasonVeleroRestoreCompleted
	case velerov1.RestorePhaseFailedValidation,
		velerov1.RestorePhasePartiallyFailed,
		velerov1.RestorePhaseFailed:
		return metav1.ConditionFalse, ReasonVeleroRestoreFailed
	default:
		return metav1.ConditionUnknown, ReasonVeleroRestorePhaseUnknown
	}
}
