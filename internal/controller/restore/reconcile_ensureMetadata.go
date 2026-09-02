package restore

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/cloudogu/k8s-backup-lib/api/ecosystem"
	k8sv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
)

// restoreLabels are the labels every Restore this operator works on carries.
func restoreLabels() map[string]string {
	return map[string]string{
		ecosystem.AppLabelKey:    ecosystem.AppLabelValueCes,
		ecosystem.PartOfLabelKey: ecosystem.PartOfLabelValueBackup,
	}
}

// ensureRestoreMetadata converges the finalizer and the labels of the given Restore in a single
// write and reports whether it had to write at all, so that a repeated reconciliation of an already
// written Restore writes nothing.
func ensureRestoreMetadata(ctx context.Context, k8sClient k8sClient, restore *k8sv1.Restore) (bool, error) {
	changed := controllerutil.AddFinalizer(restore, k8sv1.RestoreFinalizer)
	if applyLabels(restore) {
		changed = true
	}

	if !changed {
		return false, nil
	}

	if err := k8sClient.Update(ctx, restore); err != nil {
		return false, fmt.Errorf("failed to write finalizer %s and labels %v of restore: %w", k8sv1.RestoreFinalizer, restoreLabels(), err)
	}

	return true, nil
}

// removeFinalizer removes the given finalizer and persists the Restore only if the finalizer set
// actually changed.
func removeFinalizer(ctx context.Context, k8sClient k8sClient, restore *k8sv1.Restore, finalizer string) error {
	if !controllerutil.RemoveFinalizer(restore, finalizer) {
		return nil
	}

	if err := k8sClient.Update(ctx, restore); err != nil {
		return fmt.Errorf("failed to remove finalizer %s from restore: %w", finalizer, err)
	}

	return nil
}

// applyLabels sets the restore labels on the given Restore without persisting it and reports whether
// at least one of them was missing or had a different value.
func applyLabels(restore *k8sv1.Restore) bool {
	desired := restoreLabels()

	changed := false
	for key, value := range desired {
		if restore.Labels[key] != value {
			changed = true
		}
	}
	if !changed {
		return false
	}

	if restore.Labels == nil {
		restore.Labels = make(map[string]string, len(desired))
	}
	for key, value := range desired {
		restore.Labels[key] = value
	}

	return true
}
