package restore

import (
	"context"
	"fmt"

	"github.com/cloudogu/k8s-backup-operator/pkg/provider"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/k8s-backup-operator/internal/provider/velero"
)

type defaultDeleteManager struct {
	k8sClient k8sClient
	namespace string
	recorder  eventRecorder
}

func newDeleteManager(k8sClient k8sClient, namespace string, recorder eventRecorder) *defaultDeleteManager {
	return &defaultDeleteManager{k8sClient: k8sClient, namespace: namespace, recorder: recorder}
}

func (dm *defaultDeleteManager) delete(ctx context.Context, restore *v1.Restore) error {
	logger := log.FromContext(ctx)
	conditions := newConditionUpdater(dm.k8sClient)

	child, err := velero.GetRestore(ctx, dm.k8sClient, restore)
	if err != nil {
		return fmt.Errorf("failed to get provider restore of [%s]: %w", restore.Name, err)
	}
	successfulCondition := effectiveSuccessfulCondition(restore)
	deleteChild := child != nil &&
		(velero.IsOwnedRestore(restore, child) ||
			(successfulCondition != nil && successfulCondition.Reason == ReasonMigratedFromLegacyStatus))

	// A deleting restore has no condition of its own; the status phase is derived from the
	// deletion timestamp, so this write carries no conditions and only persists that status phase.
	restore, err = conditions.setConditions(ctx, restore)
	if err != nil {
		return fmt.Errorf("failed to update status [%s] on restore [%s]: %w", v1.RestoreStatusDeleting, restore.Name, err) // NOSONAR -- legacy restore status compatibility
	}

	// The provider is still resolved for its readiness gate and its provider-selection event, but the
	// provider child is deleted directly: it is owned by this controller, not by the provider gateway.
	_, err = provider.Get(ctx, restore, restore.Spec.Provider, restore.Namespace, dm.recorder, dm.k8sClient)
	if err != nil {
		return fmt.Errorf("failed to get provider [%s]: %w", restore.Spec.Provider, err)
	}

	switch {
	case deleteChild:
		err = velero.DeleteRestore(ctx, dm.k8sClient, child)
		if err != nil {
			return fmt.Errorf("failed to delete restore: %w", err)
		}
	case child != nil:
		message := fmt.Sprintf("Leaving provider restore [%s] untouched because it is not owned by this restore. Remove it manually if it is not needed.", child.Name)
		logger.Info(message)
		dm.recorder.Event(restore, corev1.EventTypeWarning, v1.DeleteEventReason, message)
	}

	err = removeFinalizer(ctx, dm.k8sClient, restore, v1.RestoreFinalizer)
	if err != nil {
		return fmt.Errorf("failed to delete finalizer [%s]: %w", v1.RestoreFinalizer, err)
	}

	return nil
}
