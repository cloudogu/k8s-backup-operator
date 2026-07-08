package backup

import (
	"context"
	"fmt"
	"maps"
	"time"

	backup "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/k8s-backup-operator/pkg/blueprint"
)

var defaultLabels = map[string]string{
	"app":                      "ces",
	"k8s.cloudogu.com/part-of": "backup",
}

type Backup struct {
	Name       string
	Labels     map[string]string
	Conditions []Condition
}

type Condition struct {
	Type               string
	Status             bool
	LastTransitionTime time.Duration
	Reason             string
	Message            string
}

type Service interface {
	createBackup(context context.Context, backup Backup) error
	cancelBackup(context context.Context, backup Backup) error
	deleteBackup(context context.Context, backup Backup) error
}

type ServiceImpl struct {
	backupRepository         backupRepository
	providerBackupRepository providerBackupRepository
	configGateway            configGateway
}

func NewService(
	backupRepository backupRepository,
	providerBackupRepository providerBackupRepository,
	configGateway configGateway) *ServiceImpl {
	return &ServiceImpl{
		backupRepository:         backupRepository,
		providerBackupRepository: providerBackupRepository,
		configGateway:            configGateway,
	}
}

func (srv *ServiceImpl) createBackup(context context.Context, backup Backup) error {
	newBackup := Backup{
		Name:   backup.Name,
		Labels: make(map[string]string),
	}
	maps.Copy(newBackup.Labels, backup.Labels)
	maps.Copy(newBackup.Labels, defaultLabels)

	err := srv.backupRepository.save(newBackup)
	if err != nil {
		return fmt.Errorf("save backup: %w", err)
	}

	err = srv.providerBackupRepository.save(newBackup)
	if err != nil {
		return fmt.Errorf("save provider backup: %w", err)
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

func (srv *ServiceImpl) initBackupCr(backupCr *backup.Backup, blueprintWithDogus *blueprint.BlueprintWithDogus) {

}
