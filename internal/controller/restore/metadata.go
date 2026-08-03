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

// addFinalizer adds the given finalizer and persists the Restore only if the finalizer set actually
// changed, so a repeated reconciliation of an already converged Restore writes nothing.
func addFinalizer(ctx context.Context, k8sClient k8sClient, restore *k8sv1.Restore, finalizer string) error {
	if !controllerutil.AddFinalizer(restore, finalizer) {
		return nil
	}

	if err := k8sClient.Update(ctx, restore); err != nil {
		return fmt.Errorf("failed to add finalizer %s to restore: %w", finalizer, err)
	}

	return nil
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

// addLabels applies the restore labels and persists the Restore only if at least one of them was
// missing or had a different value.
func addLabels(ctx context.Context, k8sClient k8sClient, restore *k8sv1.Restore) error {
	desired := restoreLabels()

	changed := false
	for key, value := range desired {
		if restore.Labels[key] != value {
			changed = true
		}
	}
	if !changed {
		return nil
	}

	if restore.Labels == nil {
		restore.Labels = make(map[string]string, len(desired))
	}
	for key, value := range desired {
		restore.Labels[key] = value
	}

	if err := k8sClient.Update(ctx, restore); err != nil {
		return fmt.Errorf("failed to add labels %v to restore: %w", desired, err)
	}

	return nil
}
