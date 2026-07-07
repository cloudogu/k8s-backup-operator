package backup

import (
	"context"
	"time"

	backup "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/k8s-backup-operator/pkg/blueprint"
)

const (
	appLabelKey            = "app"
	appLabelValueCes       = "ces"
	partOfLabelKey         = "k8s.cloudogu.com/part-of"
	partOfLabelValueBackup = "backup"
)

type Backup struct {
	Name       string
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
	// TODO: rename markBackupAsNotFinishedInTime
	markBackupAsNotFinishedInTime(context context.Context, backup Backup) error
	deleteBackup(context context.Context, backup Backup) error
}

type ServiceImpl struct {
}

func (srv *ServiceImpl) initBackupCr(backupCr *backup.Backup, blueprintWithDogus *blueprint.BlueprintWithDogus) {
	if backupCr.Labels == nil {
		backupCr.Labels = make(map[string]string)
	}
	backupCr.Labels[appLabelKey] = appLabelValueCes
	backupCr.Labels[partOfLabelKey] = partOfLabelValueBackup

}
