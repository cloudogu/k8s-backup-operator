package velero

import (
	"context"
	"fmt"

	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// ReasonVeleroSourceBackupUsable reports a backup that can be restored.
	ReasonVeleroSourceBackupUsable = "VeleroSourceBackupUsable"
	// ReasonVeleroSourceBackupNotUsable reports a backup that is missing or in a phase that cannot be restored.
	ReasonVeleroSourceBackupNotUsable = "VeleroSourceBackupNotUsable"
)

// SourceBackupCheck reports whether the backup a restore wants to replay can be restored. Reason and
// Message are written to the conditions of the restore that waits for it, so they describe the
// backup rather than the resource that observed it.
type SourceBackupCheck struct {
	Usable  bool
	Reason  string
	Message string
}

// CheckSourceBackup reports whether the named velero backup can be restored. API server errors are
// returned as an error.
func CheckSourceBackup(ctx context.Context, k8sClient client.Client, namespace string, backupName string) (SourceBackupCheck, error) {
	backup := &velerov1.Backup{}
	err := k8sClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: backupName}, backup)
	if apierrors.IsNotFound(err) {
		return SourceBackupCheck{
			Reason:  ReasonVeleroSourceBackupNotUsable,
			Message: fmt.Sprintf("The velero backup 'name=%s' to be restored was not found.", backupName),
		}, nil
	}
	if err != nil {
		return SourceBackupCheck{}, fmt.Errorf("get velero backup 'name=%s': %w", backupName, err)
	}

	phase := backup.Status.Phase
	if phase != velerov1.BackupPhaseCompleted && phase != velerov1.BackupPhasePartiallyFailed {
		return SourceBackupCheck{
			Reason:  ReasonVeleroSourceBackupNotUsable,
			Message: fmt.Sprintf("The velero backup 'name=%s' cannot be restored (phase: %s).", backupName, phase),
		}, nil
	}

	return SourceBackupCheck{
		Usable:  true,
		Reason:  ReasonVeleroSourceBackupUsable,
		Message: fmt.Sprintf("The velero backup 'name=%s' can be restored (phase: %s).", backupName, phase),
	}, nil
}
