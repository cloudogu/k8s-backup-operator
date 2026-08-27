package backup

import (
	"context"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/k8s-backup-operator/internal/metrics"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type statusUpdate func(status *backupv1.BackupStatus)

type conditionsUpdater struct {
	client client.Client
}

type backupConditionTransition struct {
	conditionType string
	from          metav1.ConditionStatus
	to            metav1.ConditionStatus
}

func newConditionsUpdater(k8sClient client.Client) *conditionsUpdater {
	return &conditionsUpdater{client: k8sClient}
}

// updateStatus applies a status mutation, synchronizes the legacy scalar phase, and records status
// transitions only after the resulting patch was persisted successfully.
func (u *conditionsUpdater) updateStatus(
	ctx context.Context,
	backup *backupv1.Backup,
	updateFn statusUpdate,
) error {
	backupBeforePatch := backup.DeepCopy()
	updateFn(&backup.Status)
	backup.Status.Status = legacyBackupStatusFor(backup)
	legacyStatusChanged := backupBeforePatch.Status.Status != backup.Status.Status

	transitions := determineBackupConditionTransitions(
		backupBeforePatch.Status.Conditions,
		backup.Status.Conditions,
	)

	if err := u.client.Status().Patch(ctx, backup, client.MergeFrom(backupBeforePatch)); err != nil {
		return err
	}

	for _, transition := range transitions {
		metrics.UpdateBackupConditionTransitionMetric(
			backup.Namespace,
			backup.Name,
			transition.conditionType,
			string(transition.from),
			string(transition.to),
		)
	}

	if legacyStatusChanged {
		metrics.UpdateBackupStatusMetrics(
			backup.Namespace,
			backup.Name,
			backup.Status.Status,
		)
	}

	return nil
}

func determineBackupConditionTransitions(
	before []metav1.Condition,
	after []metav1.Condition,
) []backupConditionTransition {
	previousStatuses := make(map[string]metav1.ConditionStatus, len(before))
	for _, beforeCondition := range before {
		previousStatuses[beforeCondition.Type] = beforeCondition.Status
	}

	var transitions []backupConditionTransition
	for _, afterCondition := range after {
		previousStatus, existed := previousStatuses[afterCondition.Type]
		if !existed || previousStatus == afterCondition.Status {
			continue
		}

		transitions = append(transitions, backupConditionTransition{
			conditionType: afterCondition.Type,
			from:          previousStatus,
			to:            afterCondition.Status,
		})
	}

	return transitions
}

// legacyBackupStatusFor derives the deprecated scalar phase consumed by clients that have not yet
// migrated to conditions.
func legacyBackupStatusFor(backup *backupv1.Backup) string {
	if meta.IsStatusConditionTrue(backup.Status.Conditions, backupv1.ConditionDeleting) {
		return backupv1.BackupStatusDeleting //nolint:staticcheck // legacy restore status compatibility
	}
	if meta.IsStatusConditionTrue(backup.Status.Conditions, backupv1.ConditionCanceled) {
		return backupv1.BackupStatusFailed //nolint:staticcheck // legacy restore status compatibility
	}
	if succeeded := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionSucceeded); succeeded != nil {
		switch succeeded.Status {
		case metav1.ConditionTrue:
			return backupv1.BackupStatusCompleted //nolint:staticcheck // legacy restore status compatibility
		case metav1.ConditionFalse:
			return backupv1.BackupStatusFailed //nolint:staticcheck // legacy restore status compatibility
		default:
			return backupv1.BackupStatusInProgress //nolint:staticcheck // legacy restore status compatibility
		}
	}
	if len(backup.Status.Conditions) > 0 {
		return backupv1.BackupStatusInProgress
	} //nolint:staticcheck // legacy restore status compatibility
	return backup.Status.Status
}
