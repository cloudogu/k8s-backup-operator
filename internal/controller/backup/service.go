package backup

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"time"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	blueprintv3 "github.com/cloudogu/k8s-blueprint-lib/v3/api/v3"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var defaultLabels = map[string]string{
	"app":                      "ces",
	"k8s.cloudogu.com/part-of": "backup",
}

const (
	blueprintIdAnnotation    = "backup.cloudogu.com/blueprintId"
	blueprintDogusAnnotation = "backup.cloudogu.com/dogus"
)

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

func (srv *ServiceImpl) addBlueprintAnnotation(context context.Context, backup *backupv1.Backup, displayName string, dogus []blueprintv3.Dogu) error {
	if backup.Annotations == nil {
		backup.Annotations = make(map[string]string)
	}

	dogusAsJson, err := json.Marshal(dogus)
	if err != nil {
		return fmt.Errorf("marshal blueprint dogus to json: %w", err)
	}

	annotations := map[string]string{
		blueprintIdAnnotation:    displayName,
		blueprintDogusAnnotation: string(dogusAsJson),
	}

	maps.Copy(backup.Annotations, annotations)

	return nil
}

func (srv *ServiceImpl) reconcileBackup(context context.Context, backup *backupv1.Backup) error {
	//TODO implement me
	panic("implement me")
}

func (srv *ServiceImpl) cancelBackup(context context.Context, backup *backupv1.Backup) error {
	//TODO implement me
	panic("implement me")
}

func (srv *ServiceImpl) deleteBackup(context context.Context, backup *backupv1.Backup) error {
	//TODO implement me
	panic("implement me")
}
