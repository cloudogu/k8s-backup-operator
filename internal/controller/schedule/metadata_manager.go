package schedule

import (
	"context"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/k8s-backup-operator/internal/logging"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	LabelApp    = "app"
	LabelPartOf = "k8s.cloudogu.com/part-of"

	LabelValueApp    = "ces"
	LabelValuePartOf = "backup"
)

type defaultMetadataManager struct {
	client client.Client
}

func (m defaultMetadataManager) ensure(ctx context.Context, schedule *backupv1.BackupSchedule) error {

	before := schedule.DeepCopy()
	changed := false

	if !controllerutil.ContainsFinalizer(schedule, backupv1.BackupScheduleFinalizer) {
		controllerutil.AddFinalizer(schedule, backupv1.BackupScheduleFinalizer)
		changed = true
	}

	if schedule.Labels == nil {
		schedule.Labels = map[string]string{}
	}

	changed = addLabelsIfNecessary(schedule, changed)
	if !changed {
		return nil
	}

	if err := m.client.Patch(ctx, schedule, client.MergeFrom(before)); err != nil {
		return err
	}

	logging.Info(ctx, "updated BackupSchedule metadata")
	return nil
}

func (m defaultMetadataManager) remove(ctx context.Context, schedule *backupv1.BackupSchedule) error {

	if !controllerutil.ContainsFinalizer(schedule, backupv1.BackupScheduleFinalizer) {
		return nil
	}

	before := schedule.DeepCopy()
	controllerutil.RemoveFinalizer(schedule, backupv1.BackupScheduleFinalizer)

	if err := m.client.Patch(ctx, schedule, client.MergeFrom(before)); err != nil {
		return err
	}

	logging.Info(ctx, "removed BackupSchedule finalizer")
	return nil
}

func addLabelsIfNecessary(schedule *backupv1.BackupSchedule, changed bool) bool {
	if schedule.Labels[LabelApp] != LabelValueApp {
		schedule.Labels[LabelApp] = LabelValueApp
		changed = true
	}

	if schedule.Labels[LabelPartOf] != LabelValuePartOf {
		schedule.Labels[LabelPartOf] = LabelValuePartOf
		changed = true
	}

	return changed
}
