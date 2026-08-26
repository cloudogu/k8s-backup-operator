package restore

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	k8sv1 "github.com/cloudogu/k8s-backup-lib/api/v1"

	"github.com/cloudogu/k8s-backup-operator/internal/provider/velero"
)

const (
	// ReasonPending marks a restore that has been observed but not yet started.
	ReasonPending = "Pending"
	// ReasonWaitingForActiveRestore marks a restore that must wait because another restore
	// currently holds the namespace-wide restore arbitration.
	ReasonWaitingForActiveRestore = "WaitingForActiveRestore"
	// ReasonRestoreLeaseAcquired marks a restore that holds the namespace-wide restore lease and
	// may continue with the destructive workflow.
	ReasonRestoreLeaseAcquired = "RestoreLeaseAcquired"
	// ReasonInvalidRestoreLease marks a restore blocked by a namespace-wide restore lease whose
	// holder cannot be identified safely.
	ReasonInvalidRestoreLease = "InvalidRestoreLease"
	// ReasonCleanupCompleted marks a completed cleanup.
	ReasonCleanupCompleted = "CleanupCompleted"
	// ReasonCleanupFailed marks a failed cleanup.
	ReasonCleanupFailed = "CleanupFailed"
	// ReasonPreparing marks a running destructive preparation, i.e. scale-down and cleanup.
	ReasonPreparing = "Preparing"
	// ReasonPreparationFailed marks a failed preparation.
	ReasonPreparationFailed = "PreparationFailed"
	// ReasonPreparationCompleted marks a finished preparation: workloads are scaled down and the
	// resources to be restored are removed.
	ReasonPreparationCompleted = "PreparationCompleted"
	// ReasonProviderRestorePending marks an owned provider restore that exists but has not started.
	ReasonProviderRestorePending = "ProviderRestorePending"
	// ReasonProviderRestoreRunning marks an owned provider restore that is executing.
	ReasonProviderRestoreRunning = "ProviderRestoreRunning"
	// ReasonProviderRestoreCompleted marks an owned provider restore that finished successfully.
	ReasonProviderRestoreCompleted = "ProviderRestoreCompleted"
	// ReasonProviderRestoreFailed marks an owned provider restore that failed terminally,
	// including validation and partial failures.
	ReasonProviderRestoreFailed = "ProviderRestoreFailed"
	// ReasonProviderRestoreStateUnknown marks an owned provider restore whose state this operator
	// does not know, for example after a provider upgrade added one. Such a state is never success.
	ReasonProviderRestoreStateUnknown = "ProviderRestoreStateUnknown"
	// ReasonProviderRestoreConflict marks an existing provider restore of the expected name that
	// this operator may not adopt because its identity metadata does not prove our ownership.
	ReasonProviderRestoreConflict = "ProviderRestoreConflict"
	// ReasonRecoveringWorkloads marks a running workload recovery, i.e. scale-up and
	// maintenance mode deactivation.
	ReasonRecoveringWorkloads = "RecoveringWorkloads"
	// ReasonWorkloadRecoveryCompleted marks a finished workload recovery: the workloads are scaled up
	// again and the maintenance mode is switched off.
	ReasonWorkloadRecoveryCompleted = "WorkloadRecoveryCompleted"
	// ReasonWorkloadRecoveryFailed marks a failed workload recovery after provider success.
	ReasonWorkloadRecoveryFailed = "WorkloadRecoveryFailed"
	// ReasonRecoveryNotAttemptedAfterProviderFailure marks workloads that were deliberately not
	// scaled up because the provider restore failed terminally.
	ReasonRecoveryNotAttemptedAfterProviderFailure = "RecoveryNotAttemptedAfterProviderFailure"
	// ReasonRestoreCompleted marks the successfully finished restore workflow.
	ReasonRestoreCompleted = "RestoreCompleted"
	// reasonRestoreStarted marks a started restore workflow.
	ReasonRestoreStarted = "RestoreStarted"
	// ReasonMigratedFromLegacyStatus marks a condition that was not written by this workflow but
	// derived from the deprecated scalar status of a Restore created by an older operator. The
	// original cause of a legacy success or failure is not recoverable, so it is not claimed.
	ReasonMigratedFromLegacyStatus = "MigratedFromLegacyStatus"
	// ReasonScaleUpInitiated means that the desired replica counts have been
	// restored, but the workloads have not necessarily become ready yet.
	ReasonScaleUpInitiated = "ScaleUpInitiated"
	// ReasonWaitingForWorkloads marks workloads that have not reached their target state yet.
	ReasonWaitingForWorkloads = "WaitingForWorkloads"
	// ReasonWorkloadsReady marks workloads that reached their target replica count and availability.
	ReasonWorkloadsReady = "WorkloadsReady"
	// ReasonScaleUpFinalized marks the removal of the temporary replica recovery labels.
	ReasonScaleUpFinalized = "ScaleUpFinalized"
	// ReasonMaintenanceModeActivated marks a failed best-effort activation attempt in an event.
	ReasonMaintenanceModeActivated = "MaintenanceModeActivated"
	// ReasonMaintenanceModeDeactivated marks the successfully removed maintenance notice.
	ReasonMaintenanceModeDeactivated = "MaintenanceModeDeactivated"
)

// workflowConditionTypes are the conditions every running restore carries, in the order a reader of
// the status should see them.
var workflowConditionTypes = []string{
	k8sv1.ConditionSuccessful,
	k8sv1.ConditionPrepared,
	k8sv1.ConditionProviderRestoreSuccessful,
	k8sv1.ConditionWorkloadsRecovered,
}

// missingWorkflowConditions returns the workflow conditions the restore does not carry yet, as
// Unknown. A condition that is already present is never returned, so a milestone that a stage has
// already resolved cannot fall back to Unknown.
func missingWorkflowConditions(restore *k8sv1.Restore) []metav1.Condition {
	var missing []metav1.Condition

	for _, conditionType := range workflowConditionTypes {
		if meta.FindStatusCondition(restore.Status.Conditions, conditionType) != nil {
			continue
		}

		missing = append(missing, metav1.Condition{
			Type:    conditionType,
			Status:  metav1.ConditionUnknown,
			Reason:  ReasonPending,
			Message: "The restore workflow has not reached this milestone yet.",
		})
	}

	return missing
}

// reachedMilestone is a reached milestone of the restore workflow.
func reachedMilestone(conditionType string, reason string, message string) metav1.Condition {
	return metav1.Condition{
		Type:    conditionType,
		Status:  metav1.ConditionTrue,
		Reason:  reason,
		Message: message,
	}
}

// observeProviderRestoreState maps the state of the owned provider restore to the status and reason
// of the tri-state ProviderRestoreSuccessful condition.
//
// Unknown means that the provider has not decided yet, so a state this operator does not know is
// never reported as success or failure. Translating provider vocabulary into a state is the provider
// package's job; this function only decides what the condition says about it.
func observeProviderRestoreState(state velero.RestoreState) (metav1.ConditionStatus, string) {
	switch state {
	case velero.RestorePending:
		return metav1.ConditionUnknown, ReasonProviderRestorePending
	case velero.RestoreRunning:
		return metav1.ConditionUnknown, ReasonProviderRestoreRunning
	case velero.RestoreSucceeded:
		return metav1.ConditionTrue, ReasonProviderRestoreCompleted
	case velero.RestoreFailed:
		return metav1.ConditionFalse, ReasonProviderRestoreFailed
	default:
		return metav1.ConditionUnknown, ReasonProviderRestoreStateUnknown
	}
}

// findSuccessfulCondition returns the Successful condition actually present in the status, or nil.
func findSuccessfulCondition(restore *k8sv1.Restore) *metav1.Condition {
	return meta.FindStatusCondition(restore.Status.Conditions, k8sv1.ConditionSuccessful)
}

// determineLegacySuccessfulCondition derives a Successful condition from the deprecated scalar status of a
// Restore that predates conditions. It returns nil when the scalar carries no interpretable
// outcome, which is the case for a new restore and for a deleting one: deletion is communicated
// by metadata.deletionTimestamp, not by status.
func determineLegacySuccessfulCondition(restore *k8sv1.Restore) *metav1.Condition {
	var status metav1.ConditionStatus
	switch restore.Status.Status { // NOSONAR -- legacy restore status compatibility
	case k8sv1.RestoreStatusCompleted: // NOSONAR -- legacy restore status compatibility
		status = metav1.ConditionTrue
	case k8sv1.RestoreStatusFailed: // NOSONAR -- legacy restore status compatibility
		status = metav1.ConditionFalse
	case k8sv1.RestoreStatusInProgress: // NOSONAR -- legacy restore status compatibility
		status = metav1.ConditionUnknown
	default:
		return nil
	}

	return &metav1.Condition{
		Type:    k8sv1.ConditionSuccessful,
		Status:  status,
		Reason:  ReasonMigratedFromLegacyStatus,
		Message: fmt.Sprintf("Derived from the deprecated status %q of a restore created before conditions existed.", restore.Status.Status),
	}
}

// effectiveSuccessfulCondition returns the Successful condition to base workflow decisions on.
// A written condition always wins; the deprecated scalar status is consulted only for restores
// that carry no Successful condition yet.
func effectiveSuccessfulCondition(restore *k8sv1.Restore) *metav1.Condition {
	if condition := findSuccessfulCondition(restore); condition != nil {
		return condition
	}

	return determineLegacySuccessfulCondition(restore)
}

// isTerminal reports whether the restore reached a terminal outcome and must not be worked on
// again. Both success and terminal failure are terminal; an unknown outcome is not.
func isTerminal(restore *k8sv1.Restore) bool {
	condition := effectiveSuccessfulCondition(restore)

	return condition != nil && condition.Status != metav1.ConditionUnknown
}

func isTerminalLegacyStatus(status string) bool {
	return status == k8sv1.RestoreStatusCompleted || status == k8sv1.RestoreStatusFailed // NOSONAR -- legacy restore status compatibility
}

// legacyStatusFor maps conditions to thr deprecated status field.
func legacyStatusFor(restore *k8sv1.Restore) string {
	if restore.DeletionTimestamp != nil && !restore.DeletionTimestamp.IsZero() {
		return k8sv1.RestoreStatusDeleting // NOSONAR -- legacy restore status compatibility
	}

	switch condition := findSuccessfulCondition(restore); {
	case condition == nil && (len(restore.Status.Conditions) == 0 || isTerminalLegacyStatus(restore.Status.Status)):
		return restore.Status.Status
	case condition == nil:
		return k8sv1.RestoreStatusInProgress // NOSONAR -- legacy restore status compatibility
	case condition.Status == metav1.ConditionTrue:
		return k8sv1.RestoreStatusCompleted // NOSONAR -- legacy restore status compatibility
	case condition.Status == metav1.ConditionFalse:
		return k8sv1.RestoreStatusFailed // NOSONAR -- legacy restore status compatibility
	default:
		return k8sv1.RestoreStatusInProgress // NOSONAR -- legacy restore status compatibility
	}
}

// restoreOutcome derives how a terminal restore ended from its effective Successful condition.
func restoreOutcome(restore *k8sv1.Restore) string {
	condition := effectiveSuccessfulCondition(restore)
	switch {
	case condition == nil:
		return "unknown"
	case condition.Status == metav1.ConditionTrue:
		return "succeeded"
	case condition.Status == metav1.ConditionFalse:
		return "failed"
	default:
		return "unknown"
	}
}
