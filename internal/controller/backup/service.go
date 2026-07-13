package backup

import (
	"context"
	"maps"
	"time"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var defaultLabels = map[string]string{
	"app":                      "ces",
	"k8s.cloudogu.com/part-of": "backup",
}

type Backup struct {
	Name        string
	Labels      map[string]string
	Annotations map[string]string
	Finalizers  []string
	Conditions  []Condition
	StartTime   *time.Time
}

type Condition struct {
	Type               string
	Status             bool
	LastTransitionTime time.Duration
	Reason             string
	Message            string
}

type Blueprint struct {
	DisplayName       string
	DogusAsJsonString string
}

type Clock interface {
	Now() time.Time
}

type ServiceImpl struct {
	client client.Client
	clock  Clock
}

func NewService(client client.Client, clock Clock) *ServiceImpl {
	return &ServiceImpl{
		client: client,
		clock:  clock,
	}
}

func (srv *ServiceImpl) configureBackup(ctx context.Context, backup *backupv1.Backup) {
	if backup.Labels == nil {
		backup.Labels = make(map[string]string)
	}

	maps.Copy(backup.Labels, defaultLabels)
	controllerutil.AddFinalizer(backup, backupv1.BackupFinalizer)
}

func (srv *ServiceImpl) addBlueprintAnnotation(context context.Context, backup *backupv1.Backup) {

}

func (srv *ServiceImpl) createBackup(context context.Context, backup Backup) error {
	newBackup := Backup{
		Name:        backup.Name,
		Labels:      make(map[string]string),
		Annotations: make(map[string]string),
		Finalizers:  append([]string(nil), backup.Finalizers...),
		StartTime:   backup.StartTime,
	}
	maps.Copy(newBackup.Labels, backup.Labels)
	maps.Copy(newBackup.Annotations, backup.Annotations)

	maps.Copy(newBackup.Labels, defaultLabels)

	newBackup.Finalizers = append(newBackup.Finalizers, backupv1.BackupFinalizer)

	if newBackup.StartTime == nil {
		newBackup.StartTime = new(srv.clock.Now())
	}

	/*
		blueprint, err := srv.blueprintGateway.find(context)
		if err != nil {
			return fmt.Errorf("find blueprint: %w", err)
		}

		if blueprint != nil {
			blueprintInfos := map[string]string{
				annotations.BlueprintIdAnnotation: blueprint.DisplayName,
				annotations.DogusAnnotation:       blueprint.DogusAsJsonString,
			}
			maps.Copy(newBackup.Annotations, blueprintInfos)
		}

	*/

	/*
		err = srv.backupRepository.save(context, newBackup)
		if err != nil {
			return fmt.Errorf("save backup: %w", err)
		}

	*/

	/*
		err = srv.maintenanceGateway.ActivateMaintenance(
			context,
			backup2.maintenanceModeTitle,
			backup2.maintenanceModeText,
		)
		if err != nil {
			return fmt.Errorf("activate maintenance mode: %w", err)
		}

	*/

	/*
		err = srv.veleroBackupRepository.save(context, newBackup)
		if err != nil {
			return fmt.Errorf("create provider backup: %w", err)
		}

	*/

	return nil
}

func (srv *ServiceImpl) cancelBackup(context context.Context, backup Backup) error {
	//TODO implement me
	panic("implement me")
}

func (srv *ServiceImpl) deleteBackup(context context.Context, backup Backup) error {
	//TODO implement me
	panic("implement me")
}
