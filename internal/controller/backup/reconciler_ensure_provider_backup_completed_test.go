package backup

import (
	"context"
	"testing"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/k8s-backup-operator/internal/controller/backup/state"
	"github.com/stretchr/testify/assert"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestReconcilerEnsureProviderBackupCompleted(t *testing.T) {
	t.Run("fail to list provider backups", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		veleroBackup := newVeleroBackupForReconcilerTest("ns", "backup", velerov1.BackupPhaseInProgress)
		fakeClient := newFakeClientBuilder(t).
			WithObjects(backup, veleroBackup).
			WithStatusSubresource(backup).
			WithInterceptorFuncs(interceptor.Funcs{List: func(ctx context.Context, client client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
				if _, isVeleroBackups := list.(*velerov1.BackupList); isVeleroBackups {
					return assert.AnError
				}
				return client.List(ctx, list, opts...)
			}}).
			Build()
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default", "velero")
		stateCollector := newMockBackupStateCollector(t)
		reconciler.stateCollector = stateCollector

		nextAction, err := reconciler.ensureProviderBackupCompleted(t.Context(), backup)

		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "listing unit backups")
		assert.Equal(t, Abort, nextAction)
	})

	t.Run("fail to collect backup state", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		veleroBackup := newVeleroBackupForReconcilerTest("ns", "backup", velerov1.BackupPhaseInProgress)
		fakeClient := newFakeClientBuilder(t).
			WithObjects(backup, veleroBackup).
			WithStatusSubresource(backup).
			Build()
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default", "velero")
		stateCollector := newMockBackupStateCollector(t)
		stateCollector.EXPECT().Collect(t.Context(), backup).Return(state.BackupState{}, assert.AnError)
		reconciler.stateCollector = stateCollector

		nextAction, err := reconciler.ensureProviderBackupCompleted(t.Context(), backup)

		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "collecting backup state")
		assert.Equal(t, Abort, nextAction)
	})

	t.Run("fail to unmarshal v2 dogus", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		veleroBackup := newVeleroBackupForReconcilerTest("ns", "backup", velerov1.BackupPhaseInProgress)
		veleroBackup.Labels[backupv1.BackupLabelKey] = "backup"
		veleroBackup.Labels[backupv1.BackupTypeLabelKey] = string(backupv1.UnitTypeDoguV2)
		veleroBackup.Annotations[backupv1.InstalledDogusV2AnnotationKey] = "invalid json"
		fakeClient := newFakeClientBuilder(t).
			WithObjects(backup, veleroBackup).
			WithStatusSubresource(backup).
			Build()
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default", "velero")
		stateCollector := newMockBackupStateCollector(t)
		stateCollector.EXPECT().Collect(t.Context(), backup).
			Return(state.BackupState{
				Units: []backupv1.UnitIdentifier{{Type: backupv1.UnitTypeDoguV2}},
			}, nil)
		reconciler.stateCollector = stateCollector

		nextAction, err := reconciler.ensureProviderBackupCompleted(t.Context(), backup)

		assert.ErrorContains(t, err, "determining unit completion: unmarshal installed dogus v2 from unit")
		assert.Equal(t, Abort, nextAction)
	})

	// TODO more tests

}
