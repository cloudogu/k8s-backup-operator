package backup

import (
	"context"
	"errors"
	"fmt"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/go-logr/logr"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type defaultReconciler struct {
	client client.Client
}

func newReconciler(client client.Client) *defaultReconciler {
	return &defaultReconciler{
		client: client,
	}
}

func (c *defaultReconciler) checkVeleroBackupStorageLocation(ctx context.Context, backup *backupv1.Backup, namespace string, logger logr.Logger) (ctrl.Result, error) {
	veleroBackupStorageLocation := velerov1.BackupStorageLocation{}
	namespacedName := types.NamespacedName{Namespace: namespace, Name: veleroBackupStorageName}
	err := c.client.Get(ctx, namespacedName, &veleroBackupStorageLocation)

	if err != nil {
		condition := metav1.Condition{
			Type:    backupv1.ConditionPrepared,
			Status:  metav1.ConditionFalse,
			Reason:  veleroBackupStorageNotAvailable,
			Message: fmt.Sprintf("velero backup storage location 'name=%s' is not available.", veleroBackupStorageName),
		}
		logger.Error(err, fmt.Sprintf("Failed to check velero backup storage location 'name=%s'", veleroBackupStorageName))

		patchErr := c.patchCondition(ctx, backup, condition)
		return ctrl.Result{RequeueAfter: defaultRequeueAfterTime}, errors.Join(err, patchErr)
	}

	if veleroBackupStorageLocation.Status.Phase != velerov1.BackupStorageLocationPhaseAvailable {
		condition := metav1.Condition{
			Type:    backupv1.ConditionPrepared,
			Status:  metav1.ConditionFalse,
			Reason:  veleroBackupStorageNotAvailable,
			Message: fmt.Sprintf("Velero backup storage location 'name=%s' is not available.", veleroBackupStorageName),
		}
		logger.Error(err, fmt.Sprintf("Velero backup storage location 'name=%s' is not available.", veleroBackupStorageName))

		patchErr := c.patchCondition(ctx, backup, condition)
		return ctrl.Result{RequeueAfter: defaultRequeueAfterTime}, patchErr
	}

	condition := metav1.Condition{
		Type:    backupv1.ConditionPrepared,
		Status:  metav1.ConditionTrue,
		Reason:  veleroBackupStorageAvailable,
		Message: fmt.Sprintf("velero backup storage location 'name=%s' is available.", veleroBackupStorageName),
	}
	patchErr := c.patchCondition(ctx, backup, condition)
	return ctrl.Result{}, patchErr
}

func (c *defaultReconciler) patchCondition(ctx context.Context, backup *backupv1.Backup, condition metav1.Condition) error {
	patchBase := backup.DeepCopy()

	meta.SetStatusCondition(&backup.Status.Conditions, condition)

	if err := c.client.Status().Patch(ctx, backup, client.MergeFrom(patchBase)); err != nil {
		return fmt.Errorf("failed to patch condition: %w", err)
	}

	return nil
}
