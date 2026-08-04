package schedule

import (
	"context"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	LabelApp    = "app"
	LabelPartOf = "k8s.cloudogu.com/part-of"

	LabelValueApp    = "ces"
	LabelValuePartOf = "backup"
)

type metadataManager struct {
	client client.Client
}

func (m metadataManager) Ensure(ctx context.Context, schedule *backupv1.BackupSchedule) error {
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

	return m.client.Patch(ctx, schedule, client.MergeFrom(before))
}

func (m metadataManager) Remove(ctx context.Context, schedule *backupv1.BackupSchedule) error {
	if !controllerutil.ContainsFinalizer(schedule, backupv1.BackupScheduleFinalizer) {
		return nil
	}

	before := schedule.DeepCopy()
	controllerutil.RemoveFinalizer(schedule, backupv1.BackupScheduleFinalizer)

	return m.client.Patch(ctx, schedule, client.MergeFrom(before))
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
