package restore

import (
	"context"
	"errors"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	k8sv1 "github.com/cloudogu/k8s-backup-lib/api/v1"

	"github.com/cloudogu/k8s-backup-operator/internal/controller/restore/state"
	"github.com/cloudogu/k8s-backup-operator/internal/logging"
)

// ensureSourceBackupUsable fails the restore before preparation if its backup cannot be restored as requested.
func (r *restoreReconciler) ensureSourceBackupUsable(ctx context.Context, restore *k8sv1.Restore) (*k8sv1.Restore, stageOutcome) {
	prepared, err := r.isAlreadyPrepared(ctx, restore)
	if err != nil {
		return restore, retryOnError(err)
	}
	if prepared {
		return restore, next()
	}

	restoreState, err := state.NewCollector(r.k8sClient).Collect(ctx, restore)
	if malformedErr, ok := errors.AsType[*state.MalformedBackupError](err); ok {
		return r.failOnUnusableSourceBackup(ctx, restore, ReasonUnusableUnits, fmt.Sprintf("The backup cannot be read: %v.", malformedErr))
	}
	if err != nil {
		return restore, retryOnError(fmt.Errorf("failed to check the backup to be restored of restore %s: %w", restore.Name, err))
	}

	reason, message := checkSourceBackup(restore.Spec, restoreState)
	if reason == "" {
		return restore, next()
	}

	return r.failOnUnusableSourceBackup(ctx, restore, reason, message)
}

// checkSourceBackup returns why the backup cannot be restored as requested; an empty reason if it can.
func checkSourceBackup(spec k8sv1.RestoreSpec, restoreState state.RestoreState) (reason string, message string) {
	if restoreState.Kind == state.KindNotFound {
		return ReasonSourceBackupNotFound, fmt.Sprintf("The velero backup 'name=%s' to be restored was not found.", spec.BackupName)
	}
	if invalid := restoreState.InvalidExclusions(spec); len(invalid) > 0 {
		return ReasonInvalidExclusions, fmt.Sprintf("The exclusions do not match the backup: %s.", strings.Join(invalid, "; "))
	}

	unitsToRestore := restoreState.UnitsToRestore(spec)
	var unusable []string
	for _, unit := range unitsToRestore {
		if !unit.IsUsable() {
			unusable = append(unusable, describeUnusableUnit(unit))
		}
	}
	if len(unusable) > 0 {
		return ReasonUnusableUnits, fmt.Sprintf("The backups of these units cannot be restored: %s.", strings.Join(unusable, "; "))
	}
	if len(unitsToRestore) == 0 {
		return ReasonUnusableUnits, "No unit is left to restore after applying the exclusions and options."
	}
	return "", ""
}

// describeUnusableUnit adds the provider backup phase or "missing" and how to skip the unit.
func describeUnusableUnit(unit state.UnitBackup) string {
	name := string(unit.Type)
	if unit.Name != "" {
		name += " " + unit.Name
	}

	observed := "missing"
	if unit.ProviderBackup != nil {
		phase := string(unit.ProviderBackup.Phase)
		if phase == "" {
			phase = "unknown"
		}
		observed = "phase " + phase
	}

	if skipOption := unit.SkipOption(); skipOption != "" {
		return fmt.Sprintf("%s (%s, skip with %s)", name, observed, skipOption)
	}
	return fmt.Sprintf("%s (%s)", name, observed)
}

// failOnUnusableSourceBackup reports the backup that cannot be restored as a terminal failure,
// before any preparation ran.
func (r *restoreReconciler) failOnUnusableSourceBackup(ctx context.Context, restore *k8sv1.Restore, reason string, message string) (*k8sv1.Restore, stageOutcome) {
	r.recorder.Eventf(restore, nil, corev1.EventTypeWarning, reason, actionCheckSourceBackup, message)

	updated, err := newConditionUpdater(r.k8sClient).setConditions(ctx, restore,
		metav1.Condition{
			Type:    k8sv1.ConditionPrepared,
			Status:  metav1.ConditionFalse,
			Reason:  reason,
			Message: message,
		},
		metav1.Condition{
			Type:    k8sv1.ConditionSucceeded,
			Status:  metav1.ConditionFalse,
			Reason:  reason,
			Message: fmt.Sprintf("The restore was not started: %s", message),
		},
	)
	if err != nil {
		return restore, retryOnError(fmt.Errorf("failed to report the unusable backup of restore %s: %w", restore.Name, err))
	}

	logging.Info(ctx, "the backup to be restored cannot be restored", "backup", restore.Spec.BackupName, "reason", reason)

	return updated, abort()
}
