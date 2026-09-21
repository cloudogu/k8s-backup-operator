package restore

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	k8sv1 "github.com/cloudogu/k8s-backup-lib/api/v1"

	"github.com/cloudogu/k8s-backup-operator/internal/logging"
	"github.com/cloudogu/k8s-backup-operator/internal/provider/velero"
)

// ensureSourceBackupUsable checks that the backup this restore wants to replay is actually available.
// The stage runs with the blocking lease held -> fail on every backup that is not finished.
func (r *restoreReconciler) ensureSourceBackupUsable(ctx context.Context, restore *k8sv1.Restore) (*k8sv1.Restore, stageOutcome) {
	prepared, err := r.isAlreadyPrepared(ctx, restore)
	if err != nil {
		return restore, retryOnError(err)
	}
	if prepared {
		return restore, next()
	}

	check, err := velero.CheckSourceBackup(ctx, r.k8sClient, restore.Namespace, restore.Spec.BackupName)
	if err != nil {
		return restore, retryOnError(fmt.Errorf("failed to check the backup to be restored of restore %s: %w", restore.Name, err))
	}

	if check.Usable {
		return restore, next()
	}

	return r.failOnUnusableSourceBackup(ctx, restore, check)
}

// failOnUnusableSourceBackup reports the backup that cannot be restored as a terminal failure,
// before any preparation ran.
func (r *restoreReconciler) failOnUnusableSourceBackup(ctx context.Context, restore *k8sv1.Restore, check velero.SourceBackupCheck) (*k8sv1.Restore, stageOutcome) {
	r.recorder.Eventf(restore, nil, corev1.EventTypeWarning, check.Reason, actionCheckSourceBackup, check.Message)

	updated, err := newConditionUpdater(r.k8sClient).setConditions(ctx, restore,
		metav1.Condition{
			Type:    k8sv1.ConditionPrepared,
			Status:  metav1.ConditionFalse,
			Reason:  check.Reason,
			Message: check.Message,
		},
		metav1.Condition{
			Type:    k8sv1.ConditionSucceeded,
			Status:  metav1.ConditionFalse,
			Reason:  check.Reason,
			Message: fmt.Sprintf("The restore was not started: %s", check.Message),
		},
	)
	if err != nil {
		return restore, retryOnError(fmt.Errorf("failed to report the unusable backup of restore %s: %w", restore.Name, err))
	}

	logging.Info(ctx, "the backup to be restored cannot be restored", "backup", restore.Spec.BackupName, "reason", check.Reason)

	return updated, abort()
}
