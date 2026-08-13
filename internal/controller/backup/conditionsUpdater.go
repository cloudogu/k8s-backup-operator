package backup

import (
	"context"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/k8s-backup-operator/pkg/metrics"
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

// updateStatus applies a status mutation and records condition status transitions only after the
// resulting patch was persisted successfully.
func (u *conditionsUpdater) updateStatus(
	ctx context.Context,
	backup *backupv1.Backup,
	updateFn statusUpdate,
) error {
	backupBeforePatch := backup.DeepCopy()
	updateFn(&backup.Status)
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

	return nil
}

func determineBackupConditionTransitions(
	before []metav1.Condition,
	after []metav1.Condition,
) []backupConditionTransition {
	previousStatuses := make(map[string]metav1.ConditionStatus, len(before))
	for _, condition := range before {
		previousStatuses[condition.Type] = condition.Status
	}

	var transitions []backupConditionTransition
	for _, condition := range after {
		previousStatus, existed := previousStatuses[condition.Type]
		if !existed || previousStatus == condition.Status {
			continue
		}

		transitions = append(transitions, backupConditionTransition{
			conditionType: condition.Type,
			from:          previousStatus,
			to:            condition.Status,
		})
	}

	return transitions
}
