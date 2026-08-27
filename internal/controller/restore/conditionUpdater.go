package restore

import (
	"context"
	"fmt"

	"github.com/cloudogu/k8s-backup-operator/internal/metrics"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"

	k8sv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
)

type restoreStatusClient interface {
	Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error
	Status() client.SubResourceWriter
}

type conditionUpdater struct {
	client restoreStatusClient
}

type conditionTransition struct {
	conditionType string
	from          metav1.ConditionStatus
	to            metav1.ConditionStatus
}

func newConditionUpdater(client restoreStatusClient) *conditionUpdater {
	return &conditionUpdater{client: client}
}

// setConditions applies the given conditions to the Restore status, writes the deprecated
// scalar status for consumers that have not migrated yet, and persists the result.
func (u *conditionUpdater) setConditions(ctx context.Context, restore *k8sv1.Restore, conditions ...metav1.Condition) (*k8sv1.Restore, error) {

	current := restore
	result := restore

	var persistedTransitions []conditionTransition
	var persistedLegacyStatus = ""

	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		persistedTransitions = nil

		desired := current.DeepCopy()
		transitions := applyConditions(desired, conditions)
		legacyStatusChanged := current.Status.Status != desired.Status.Status

		if apiequality.Semantic.DeepEqual(current.Status, desired.Status) {
			result = current

			return nil
		}

		updateErr := u.client.Status().Update(ctx, desired)
		if apierrors.IsConflict(updateErr) {
			refreshed := &k8sv1.Restore{}
			if getErr := u.client.Get(ctx, client.ObjectKeyFromObject(restore), refreshed); getErr != nil {
				return fmt.Errorf("failed to get restore %q after a conflicting status update: %w", restore.Name, getErr)
			}
			current = refreshed

			return updateErr
		}
		if updateErr != nil {
			return updateErr
		}

		// only count transition of successful update
		persistedTransitions = transitions
		if legacyStatusChanged {
			persistedLegacyStatus = desired.Status.Status
		}
		// The client updated desired in place, so it now carries the persisted resource version.
		result = desired

		return nil
	})
	if err != nil {
		return restore, fmt.Errorf("failed to update status of restore %q: %w", restore.Name, err)
	}

	updateRestoreMetrics(result, persistedTransitions, persistedLegacyStatus)

	return result, nil
}

func updateRestoreMetrics(restore *k8sv1.Restore, persistedTransitions []conditionTransition, persistedLegacyStatus string) {
	for _, transition := range persistedTransitions {
		metrics.UpdateRestoreConditionTransitionMetric(
			restore.Namespace,
			restore.Name,
			restore.Spec.BackupName,
			transition.conditionType,
			string(transition.from),
			string(transition.to),
		)
	}
	if persistedLegacyStatus != "" {
		metrics.UpdateRestoreStatusMetrics(
			restore.Namespace,
			restore.Name,
			restore.Spec.BackupName,
			persistedLegacyStatus,
		)
	}
}

// setConditionsFromLegacyStatus persists the Successful condition derived from the deprecated
// scalar status of a Restore created by an older operator, so that the interpretation has to happen
// only once.
func (u *conditionUpdater) setConditionsFromLegacyStatus(ctx context.Context, restore *k8sv1.Restore) (*k8sv1.Restore, error) {
	if findSuccessfulCondition(restore) != nil {
		return restore, nil
	}

	condition := determineLegacySuccessfulCondition(restore)
	if condition == nil {
		return restore, nil
	}

	return u.setConditions(ctx, restore, *condition)
}

// applyConditions merges the conditions into the status and keeps the deprecated scalar status in
// sync.
func applyConditions(restore *k8sv1.Restore, conditions []metav1.Condition) []conditionTransition {
	var transitions []conditionTransition
	for _, condition := range conditions {
		previous := meta.FindStatusCondition(
			restore.Status.Conditions,
			condition.Type,
		)
		var previousStatus metav1.ConditionStatus
		hadPrevious := previous != nil
		if hadPrevious {
			previousStatus = previous.Status
		}
		condition.ObservedGeneration = restore.Generation
		meta.SetStatusCondition(&restore.Status.Conditions, condition)

		if hadPrevious && previousStatus != condition.Status {
			transitions = append(transitions, conditionTransition{
				conditionType: condition.Type,
				from:          previousStatus,
				to:            condition.Status,
			})
		}
	}

	restore.Status.Status = legacyStatusFor(restore) // NOSONAR -- legacy restore status compatibility
	return transitions
}
