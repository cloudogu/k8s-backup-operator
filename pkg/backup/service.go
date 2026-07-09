package backup

import (
	"context"
	"fmt"
	"maps"
	"time"

	backupV1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/k8s-backup-operator/pkg/annotations"
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

type Service interface {
	createBackup(context context.Context, backup Backup) error
	cancelBackup(context context.Context, backup Backup) error
	deleteBackup(context context.Context, backup Backup) error
}

type ServiceImpl struct {
	backupRepository backupRepository
	configGateway    configGateway
	blueprintGateway blueprintGateway
	providerService  Service
}

func NewService(
	backupRepository backupRepository,
	providerService Service,
	configGateway configGateway,
	blueprintGateway blueprintGateway) *ServiceImpl {
	return &ServiceImpl{
		backupRepository: backupRepository,
		providerService:  providerService,
		configGateway:    configGateway,
		blueprintGateway: blueprintGateway,
	}
}

func (srv *ServiceImpl) createBackup(context context.Context, backup Backup) error {
	newBackup := Backup{
		Name:        backup.Name,
		Labels:      make(map[string]string),
		Annotations: make(map[string]string),
		Finalizers:  append([]string(nil), backup.Finalizers...),
	}
	maps.Copy(newBackup.Labels, backup.Labels)
	maps.Copy(newBackup.Annotations, backup.Annotations)

	maps.Copy(newBackup.Labels, defaultLabels)

	newBackup.Finalizers = append(newBackup.Finalizers, backupV1.BackupFinalizer)

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

	err = srv.backupRepository.save(context, newBackup)
	if err != nil {
		return fmt.Errorf("save backup: %w", err)
	}

	err = srv.providerService.createBackup(context, newBackup)
	if err != nil {
		return fmt.Errorf("create provider backup: %w", err)
	}
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
