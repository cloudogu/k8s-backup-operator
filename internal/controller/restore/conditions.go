package restore

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	k8sv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
)

const (
	// ReasonPending marks a restore that has been observed but not yet started.
	ReasonPending = "Pending"
	// ReasonWaitingForActiveRestore marks a restore that must wait because another restore
	// currently holds the namespace-wide restore arbitration.
	ReasonWaitingForActiveRestore = "WaitingForActiveRestore"
	// ReasonPreparing marks a running (destructive) preparation, i.e. maintenance mode,
	// scale-down and cleanup.
	ReasonPreparing = "Preparing"
	// ReasonPreparationFailed marks a terminally failed preparation.
	ReasonPreparationFailed = "PreparationFailed"
	// ReasonVeleroRestorePending marks an owned Velero restore that exists but has not started.
	ReasonVeleroRestorePending = "VeleroRestorePending"
	// ReasonVeleroRestoreRunning marks an owned Velero restore that is executing.
	ReasonVeleroRestoreRunning = "VeleroRestoreRunning"
	// ReasonVeleroRestoreCompleted marks an owned Velero restore that finished successfully.
	ReasonVeleroRestoreCompleted = "VeleroRestoreCompleted"
	// ReasonVeleroRestoreFailed marks an owned Velero restore that failed terminally,
	// including validation and partial failures.
	ReasonVeleroRestoreFailed = "VeleroRestoreFailed"
	// ReasonVeleroRestorePhaseUnknown marks an owned Velero restore whose phase this operator does
	// not know, for example after a Velero upgrade added one. Such a phase is never success.
	ReasonVeleroRestorePhaseUnknown = "VeleroRestorePhaseUnknown"
	// ReasonVeleroRestoreConflict marks an existing Velero restore of the expected name that
	// this operator may not adopt because its identity metadata does not prove our ownership.
	ReasonVeleroRestoreConflict = "VeleroRestoreConflict"
	// ReasonRecoveringWorkloads marks a running workload recovery, i.e. scale-up and
	// maintenance mode deactivation.
	ReasonRecoveringWorkloads = "RecoveringWorkloads"
	// ReasonWorkloadRecoveryFailed marks a failed workload recovery after provider success.
	ReasonWorkloadRecoveryFailed = "WorkloadRecoveryFailed"
	// ReasonRecoveryNotAttemptedAfterProviderFailure marks workloads that were deliberately not
	// scaled up because the provider restore failed terminally.
	ReasonRecoveryNotAttemptedAfterProviderFailure = "RecoveryNotAttemptedAfterProviderFailure"
	// ReasonSynchronizingBackups marks the running read-only backup catalog convergence check.
	ReasonSynchronizingBackups = "SynchronizingBackups"
	// ReasonBackupSynchronizationFailed marks a backup catalog that did not converge.
	ReasonBackupSynchronizationFailed = "BackupSynchronizationFailed"
	// ReasonRestoreCompleted marks the successfully finished restore workflow.
	ReasonRestoreCompleted = "RestoreCompleted"
	// ReasonMigratedFromLegacyStatus marks a condition that was not written by this workflow but
	// derived from the deprecated scalar status of a Restore created by an older operator. The
	// original cause of a legacy success or failure is not recoverable, so it is not claimed.
	ReasonMigratedFromLegacyStatus = "MigratedFromLegacyStatus"
)

// successfulCondition returns the Successful condition actually present in the status, or nil.
func successfulCondition(restore *k8sv1.Restore) *metav1.Condition {
	return meta.FindStatusCondition(restore.Status.Conditions, k8sv1.ConditionSuccessful)
}

// legacySuccessfulCondition derives a Successful condition from the deprecated scalar status of a
// Restore that predates conditions. It returns nil when the scalar carries no interpretable
// outcome, which is the case for a new restore and for a deleting one: deletion is communicated
// by metadata.deletionTimestamp, not by status.
//
// This exists only for the compatibility window. It must never turn an existing terminal restore
// into new work.
func legacySuccessfulCondition(restore *k8sv1.Restore) *metav1.Condition {
	var status metav1.ConditionStatus
	switch restore.Status.Status {
	case k8sv1.RestoreStatusCompleted:
		status = metav1.ConditionTrue
	case k8sv1.RestoreStatusFailed:
		status = metav1.ConditionFalse
	case k8sv1.RestoreStatusInProgress:
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
//
// The derived condition is meant to be persisted once, so that a restore created by an older
// operator ends up with real conditions and this interpretation can be deleted after the
// compatibility window. Only Successful is seeded; milestone conditions are never derived,
// because this operator did not observe those stages. The seeding write itself belongs to the
// status updater and the reconciler stages, not here.
func effectiveSuccessfulCondition(restore *k8sv1.Restore) *metav1.Condition {
	if condition := successfulCondition(restore); condition != nil {
		return condition
	}

	return legacySuccessfulCondition(restore)
}

// isTerminal reports whether the restore reached a terminal outcome and must not be worked on
// again. Both success and terminal failure are terminal; an unknown outcome is not.
func isTerminal(restore *k8sv1.Restore) bool {
	condition := effectiveSuccessfulCondition(restore)

	return condition != nil && condition.Status != metav1.ConditionUnknown
}

// legacyStatusFor maps conditions to thr deprecated status field.
func legacyStatusFor(restore *k8sv1.Restore) string {
	if restore.DeletionTimestamp != nil && !restore.DeletionTimestamp.IsZero() {
		return k8sv1.RestoreStatusDeleting
	}

	switch condition := successfulCondition(restore); {
	case condition == nil && len(restore.Status.Conditions) == 0:
		return restore.Status.Status
	case condition == nil:
		return k8sv1.RestoreStatusInProgress
	case condition.Status == metav1.ConditionTrue:
		return k8sv1.RestoreStatusCompleted
	case condition.Status == metav1.ConditionFalse:
		return k8sv1.RestoreStatusFailed
	default:
		return k8sv1.RestoreStatusInProgress
	}
}
