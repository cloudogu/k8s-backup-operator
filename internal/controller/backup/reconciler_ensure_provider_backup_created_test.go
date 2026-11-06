package backup

import (
	"context"
	"errors"
	"testing"
	"time"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/k8s-backup-operator/internal/controller/backup/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func newBackupStateForTest(backupName string) state.BackupState {
	return state.BackupState{
		Name:     backupName,
		Coverage: backupv1.BackupCoverageFull,
		Units: []backupv1.UnitIdentifier{
			{Type: backupv1.UnitTypeDoguV2},
			{Type: backupv1.UnitTypeDogu, Name: "ldap"},
			{Type: backupv1.UnitTypeGlobalConfig},
			{Type: backupv1.UnitTypeMisc},
		},
		InstalledDogusV2: []backupv1.DoguVersion{{Name: "cas", Version: "7.0.0-1"}},
		InstalledDogusV3: []backupv1.DoguVersion{{Name: "ldap", Version: "2.6.8-1"}},
		Fqdn:             "ces.example.com",
		Domain:           "example.com",
	}
}

func newReconcilerForProviderBackupCreatedTest(t *testing.T, counter *callCounter, collector backupStateCollector, objects ...client.Object) (*defaultReconciler, client.Client) {
	fakeClient := newFakeClientBuilderWithCounter(t, counter).
		WithObjects(objects...).
		WithStatusSubresource(&backupv1.Backup{}).
		Build()
	reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default", "velero")
	reconciler.stateCollector = collector
	return reconciler, fakeClient
}

func TestReconcilerEnsureProviderBackupCreated(t *testing.T) {
	t.Run("If no unit exists, create all units, set succeeded to unknown, set start time and retry", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		counter := &callCounter{}
		collector := newMockBackupStateCollector(t)
		collector.EXPECT().Collect(mock.Anything, backup).Return(newBackupStateForTest(backup.Name), nil)
		reconciler, fakeClient := newReconcilerForProviderBackupCreatedTest(t, counter, collector, backup)

		require.True(t, backup.Status.StartTimestamp.IsZero())

		nextAction, err := reconciler.ensureProviderBackupCreated(context.Background(), backup)

		require.NoError(t, err)
		assert.Equal(t, Retry, nextAction)

		succeeded := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionSucceeded)
		require.NotNil(t, succeeded)
		assert.Equal(t, metav1.ConditionUnknown, succeeded.Status)
		assert.Equal(t, reasonProviderBackupResourceDoesNotExist, succeeded.Reason)
		assert.False(t, backup.Status.StartTimestamp.IsZero())

		assert.Equal(t, 4, counter.veleroBackupCreateCount)
		assert.Equal(t, 1, counter.subResourcePatchCount)

		units := &velerov1.BackupList{}
		require.NoError(t, fakeClient.List(context.Background(), units, client.InNamespace("ns")))
		var names []string
		for _, unit := range units.Items {
			names = append(names, unit.Name)
			assert.Equal(t, backup.Name, unit.Labels[backupv1.BackupLabelKey])
			assert.Equal(t, "default", unit.Spec.StorageLocation)
		}
		assert.ElementsMatch(t, []string{"backup-doguv2", "backup-dogu-ldap", "backup-global-config", "backup-misc"}, names)
	})

	t.Run("Set the start time only once", func(t *testing.T) {
		// The start time is set by using the function time.Now(). If we used time.Now() here as base time
		// we would not be able to detect if the start time was modified. The time difference would be too small.
		baseTime := metav1.NewTime(time.Now().Add(time.Minute))
		backup := newBackupForTest("ns", "backup")
		backup.Status.StartTimestamp = baseTime
		counter := &callCounter{}
		collector := newMockBackupStateCollector(t)
		collector.EXPECT().Collect(mock.Anything, backup).Return(newBackupStateForTest(backup.Name), nil)
		reconciler, _ := newReconcilerForProviderBackupCreatedTest(t, counter, collector, backup)

		nextAction, err := reconciler.ensureProviderBackupCreated(context.Background(), backup)

		require.NoError(t, err)
		assert.Equal(t, Retry, nextAction)
		assert.WithinDuration(t, baseTime.Time, backup.Status.StartTimestamp.Time, time.Second)
	})

	t.Run("If all units exist, proceed to the next step", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		counter := &callCounter{}
		collector := newMockBackupStateCollector(t)
		collector.EXPECT().Collect(mock.Anything, backup).Return(newBackupStateForTest(backup.Name), nil)
		objects := []client.Object{backup}
		for _, name := range []string{"backup-doguv2", "backup-dogu-ldap", "backup-global-config", "backup-misc"} {
			objects = append(objects, newUnitVeleroBackupForReconcilerTest("ns", name, backup.Name, velerov1.BackupPhaseInProgress))
		}
		reconciler, _ := newReconcilerForProviderBackupCreatedTest(t, counter, collector, objects...)

		nextAction, err := reconciler.ensureProviderBackupCreated(context.Background(), backup)

		require.NoError(t, err)
		assert.Equal(t, Next, nextAction)
		assert.Equal(t, 0, counter.veleroBackupCreateCount)
		assert.Equal(t, 0, counter.subResourcePatchCount)
	})

	t.Run("If some units exist, create only the missing units", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		counter := &callCounter{}
		collector := newMockBackupStateCollector(t)
		collector.EXPECT().Collect(mock.Anything, backup).Return(newBackupStateForTest(backup.Name), nil)
		reconciler, _ := newReconcilerForProviderBackupCreatedTest(t, counter, collector,
			backup,
			newUnitVeleroBackupForReconcilerTest("ns", "backup-doguv2", backup.Name, velerov1.BackupPhaseInProgress),
			newUnitVeleroBackupForReconcilerTest("ns", "backup-dogu-ldap", backup.Name, velerov1.BackupPhaseNew),
		)

		nextAction, err := reconciler.ensureProviderBackupCreated(context.Background(), backup)

		require.NoError(t, err)
		assert.Equal(t, Retry, nextAction)
		assert.Equal(t, 2, counter.veleroBackupCreateCount)
	})

	t.Run("Ignore units of other backups", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		counter := &callCounter{}
		collector := newMockBackupStateCollector(t)
		collector.EXPECT().Collect(mock.Anything, backup).Return(state.BackupState{
			Name:  backup.Name,
			Units: []backupv1.UnitIdentifier{{Type: backupv1.UnitTypeMisc}},
		}, nil)
		reconciler, _ := newReconcilerForProviderBackupCreatedTest(t, counter, collector,
			backup,
			newUnitVeleroBackupForReconcilerTest("ns", "other-misc", "other", velerov1.BackupPhaseInProgress),
		)

		nextAction, err := reconciler.ensureProviderBackupCreated(context.Background(), backup)

		require.NoError(t, err)
		assert.Equal(t, Retry, nextAction)
		assert.Equal(t, 1, counter.veleroBackupCreateCount)
	})

	t.Run("Treat an already existing unit as created, because the cache may be outdated", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		counter := &callCounter{
			createCallError: apierrors.NewAlreadyExists(schema.GroupResource{Group: "velero.io", Resource: "backups"}, "backup-misc"),
		}
		collector := newMockBackupStateCollector(t)
		collector.EXPECT().Collect(mock.Anything, backup).Return(state.BackupState{
			Name:  backup.Name,
			Units: []backupv1.UnitIdentifier{{Type: backupv1.UnitTypeMisc}},
		}, nil)
		reconciler, _ := newReconcilerForProviderBackupCreatedTest(t, counter, collector, backup)

		nextAction, err := reconciler.ensureProviderBackupCreated(context.Background(), backup)

		require.NoError(t, err)
		assert.Equal(t, Next, nextAction)
		assert.Equal(t, 0, counter.subResourcePatchCount)
	})

	t.Run("If collecting the backup state fails, abort", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		counter := &callCounter{}
		collector := newMockBackupStateCollector(t)
		collector.EXPECT().Collect(mock.Anything, backup).Return(state.BackupState{}, errors.New("collect error"))
		reconciler, _ := newReconcilerForProviderBackupCreatedTest(t, counter, collector, backup)

		nextAction, err := reconciler.ensureProviderBackupCreated(context.Background(), backup)

		assert.ErrorContains(t, err, "collect error")
		assert.Equal(t, Abort, nextAction)
		assert.Equal(t, 0, counter.veleroBackupCreateCount)
	})

	t.Run("If listing the units fails, abort", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		counter := &callCounter{veleroBackupListCallError: errors.New("list error")}
		collector := newMockBackupStateCollector(t)
		collector.EXPECT().Collect(mock.Anything, backup).Return(newBackupStateForTest(backup.Name), nil)
		reconciler, _ := newReconcilerForProviderBackupCreatedTest(t, counter, collector, backup)

		nextAction, err := reconciler.ensureProviderBackupCreated(context.Background(), backup)

		assert.ErrorContains(t, err, "list error")
		assert.Equal(t, Abort, nextAction)
		assert.Equal(t, 0, counter.veleroBackupCreateCount)
	})

	t.Run("If creating a unit fails, abort", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		counter := &callCounter{createCallError: errors.New("create error")}
		collector := newMockBackupStateCollector(t)
		collector.EXPECT().Collect(mock.Anything, backup).Return(newBackupStateForTest(backup.Name), nil)
		reconciler, _ := newReconcilerForProviderBackupCreatedTest(t, counter, collector, backup)

		nextAction, err := reconciler.ensureProviderBackupCreated(context.Background(), backup)

		assert.ErrorContains(t, err, "create error")
		assert.Equal(t, Abort, nextAction)
		assert.Equal(t, 0, counter.subResourcePatchCount)
		assert.True(t, backup.Status.StartTimestamp.IsZero())
	})

	t.Run("If creating a later unit fails, set the start time for the created units and abort", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		counter := &callCounter{}
		collector := newMockBackupStateCollector(t)
		collector.EXPECT().Collect(mock.Anything, backup).Return(newBackupStateForTest(backup.Name), nil)
		reconciler, _ := newReconcilerForProviderBackupCreatedTest(t, counter, collector, backup)
		createCalls := 0
		reconciler.client = interceptCreate(reconciler.client, func() error {
			createCalls++
			if createCalls == 2 {
				return errors.New("create error")
			}
			return nil
		})

		nextAction, err := reconciler.ensureProviderBackupCreated(context.Background(), backup)

		assert.ErrorContains(t, err, "create error")
		assert.Equal(t, Abort, nextAction)
		assert.Equal(t, 1, counter.veleroBackupCreateCount)
		assert.Equal(t, 1, counter.subResourcePatchCount)
		assert.False(t, backup.Status.StartTimestamp.IsZero())
	})

	t.Run("If patching the status fails, abort", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		counter := &callCounter{subResourcePatchCallError: errors.New("patch error")}
		collector := newMockBackupStateCollector(t)
		collector.EXPECT().Collect(mock.Anything, backup).Return(newBackupStateForTest(backup.Name), nil)
		reconciler, _ := newReconcilerForProviderBackupCreatedTest(t, counter, collector, backup)

		nextAction, err := reconciler.ensureProviderBackupCreated(context.Background(), backup)

		assert.ErrorContains(t, err, "patch error")
		assert.Equal(t, Abort, nextAction)
	})
}

// createInterceptingClient fails a Create call if the given function returns an error.
type createInterceptingClient struct {
	client.Client
	beforeCreate func() error
}

func (c createInterceptingClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	if err := c.beforeCreate(); err != nil {
		return err
	}
	return c.Client.Create(ctx, obj, opts...)
}

func interceptCreate(k8sClient client.Client, beforeCreate func() error) client.Client {
	return createInterceptingClient{Client: k8sClient, beforeCreate: beforeCreate}
}
