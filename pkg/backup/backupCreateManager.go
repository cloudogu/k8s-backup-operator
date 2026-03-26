package backup

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	v1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	annotationsPkg "github.com/cloudogu/k8s-backup-operator/pkg/annotations"
	"github.com/cloudogu/k8s-backup-operator/pkg/metrics"
	"github.com/cloudogu/k8s-backup-operator/pkg/provider"
	blueprintv3 "github.com/cloudogu/k8s-blueprint-lib/v3/client"
	"github.com/cloudogu/k8s-registry-lib/repository"
	"github.com/cloudogu/retry-lib/retry"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	maintenanceModeTitle = "Service temporary unavailable"
	maintenanceModeText  = "Backup in progress"
)

type backupCreateManager struct {
	k8sClient             k8sClient
	clientSet             ecosystemInterface
	blueprintClient       blueprintv3.BlueprintInterface
	namespace             string
	recorder              eventRecorder
	maintenanceModeSwitch MaintenanceModeSwitch
}

// newBackupCreateManager creates a new instance of backupCreateManager.
func newBackupCreateManager(k8sClient k8sClient, clientSet ecosystemInterface, blueprintClient blueprintv3.BlueprintInterface, namespace string, recorder eventRecorder) *backupCreateManager {
	maintenanceModeSwitch := repository.NewMaintenanceModeAdapter("k8s-backup-operator", k8sClient, namespace)
	return &backupCreateManager{k8sClient: k8sClient, clientSet: clientSet, blueprintClient: blueprintClient, namespace: namespace, recorder: recorder, maintenanceModeSwitch: maintenanceModeSwitch}
}

func (bcm *backupCreateManager) create(ctx context.Context, backup *v1.Backup) error {
	logger := log.FromContext(ctx)
	metrics.InitBackupStatusMetrics(bcm.namespace, backup.Name)
	bcm.recorder.Event(backup, corev1.EventTypeNormal, v1.CreateEventReason, "Start backup process")
	backupClient := bcm.clientSet.EcosystemV1Alpha1().Backups(bcm.namespace)

	backup, err := backupClient.UpdateStatusInProgress(ctx, backup)
	if err != nil {
		return fmt.Errorf("failed to set status [%s] in backup resource: %w", v1.BackupStatusInProgress, err)
	}
	metrics.UpdateBackupStatusMetrics(bcm.namespace, backup.Name, v1.BackupStatusInProgress)

	backup.Status.StartTimestamp = metav1.Now()
	backup, err = backupClient.UpdateStatus(ctx, backup, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update start time in status of backup resource: %w", err)
	}

	defer func(backup *v1.Backup) {
		errDefer := bcm.updateCompletionTimestamp(ctx, backup)
		if errDefer != nil {
			logger.Error(fmt.Errorf("failed to update completion time in status of backup resource: %w", err), "backup error")
		}
	}(backup)

	backup, err = backupClient.AddFinalizer(ctx, backup, v1.BackupFinalizer)
	if err != nil {
		return fmt.Errorf("failed to set finalizer %s to backup resource: %w", v1.BackupFinalizer, err)
	}

	backup, err = backupClient.AddLabels(ctx, backup)
	if err != nil {
		return fmt.Errorf("failed to add labels to backup resource: %w", err)
	}

	backup, err = bcm.addAnnotations(ctx, backup)
	if err != nil {
		return fmt.Errorf("failed to add annotations to backup resource: %w", err)
	}

	err = bcm.maintenanceModeSwitch.Activate(ctx, repository.MaintenanceModeDescription{
		Title: maintenanceModeTitle,
		Text:  maintenanceModeText,
	}, false)
	if err != nil {
		return fmt.Errorf("failed to active maintenance mode: %w", err)
	}

	defer func() {
		errDefer := bcm.maintenanceModeSwitch.Deactivate(ctx, false)
		if errDefer != nil {
			logger.Error(fmt.Errorf("failed to deactivate maintenance mode: [%w]", errDefer), "backup error")
		}
	}()

	err = bcm.triggerBackup(ctx, backup)
	if err != nil {
		err = fmt.Errorf("failed to trigger backup provider: %w", err)
		_, updateStatusErr := backupClient.UpdateStatusFailed(ctx, backup)
		if updateStatusErr != nil {
			err = errors.Join(err, fmt.Errorf("failed to update backups status to 'Failed': %w", updateStatusErr))
		}
		metrics.UpdateBackupStatusMetrics(bcm.namespace, backup.Name, v1.BackupStatusFailed)

		return err
	}

	_, err = backupClient.UpdateStatusCompleted(ctx, backup)
	if err != nil {
		return fmt.Errorf("failed to set status [%s] in backup resource: %w", v1.BackupStatusCompleted, err)
	}
	metrics.UpdateBackupStatusMetrics(bcm.namespace, backup.Name, v1.BackupStatusCompleted)

	return nil
}

func (bcm *backupCreateManager) updateCompletionTimestamp(ctx context.Context, backup *v1.Backup) error {
	backupClient := bcm.clientSet.EcosystemV1Alpha1().Backups(bcm.namespace)
	return retry.OnConflict(func() error {
		backup, err := backupClient.Get(ctx, backup.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		backup.Status.CompletionTimestamp = metav1.Now()
		_, err = backupClient.UpdateStatus(ctx, backup, metav1.UpdateOptions{})
		return err
	})
}

func (bcm *backupCreateManager) triggerBackup(ctx context.Context, backup *v1.Backup) error {
	backupProvider, err := provider.Get(ctx, backup, backup.Spec.Provider, backup.Namespace, bcm.recorder, bcm.k8sClient)
	if err != nil {
		return fmt.Errorf("failed to get backup provider: %w", err)
	}

	return backupProvider.CreateBackup(ctx, backup)
}

// addAnnotation adds annotations to the backup resource.
// * blueprintIdAnnotation: the id of the blueprint
// * dogusAnnotation: the dogus of the blueprint
func (bcm *backupCreateManager) addAnnotations(ctx context.Context, backup *v1.Backup) (*v1.Backup, error) {
	annotations := backup.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}

	// add blueprint id
	blueprintList, err := bcm.blueprintClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get blueprint: %w", err)
	}
	if len(blueprintList.Items) == 0 {
		return nil, fmt.Errorf("no blueprint found")
	}

	blueprint := blueprintList.Items[0]
	annotations[annotationsPkg.BlueprintIdAnnotation] = blueprint.Spec.DisplayName

	// add dogus
	dogus, err := json.Marshal(blueprint.Spec.Blueprint.Dogus)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal dogus: %w", err)
	}
	annotations[annotationsPkg.DogusAnnotation] = string(dogus)

	// update the resource to persist the annotations
	backup.SetAnnotations(annotations)
	err = bcm.k8sClient.Update(ctx, backup)
	if err != nil {
		return nil, fmt.Errorf("failed to update annotations on backup resource: %w", err)
	}
	return backup, nil
}
