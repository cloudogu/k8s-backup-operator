package backup

import (
	"context"
	"errors"
	"fmt"
	"strings"
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
	"k8s.io/client-go/tools/events"
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
	t.Run("If no unit exists, create all units, set succeeded to unknown, set start time and proceed to the next step", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		counter := &callCounter{}
		collector := newMockBackupStateCollector(t)
		collector.EXPECT().Collect(mock.Anything, backup).Return(newBackupStateForTest(backup.Name), nil)
		reconciler, fakeClient := newReconcilerForProviderBackupCreatedTest(t, counter, collector, backup)

		require.True(t, backup.Status.StartTimestamp.IsZero())

		nextAction, err := reconciler.ensureProviderBackupCreated(context.Background(), backup)

		require.NoError(t, err)
		assert.Equal(t, Next, nextAction)

		succeeded := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionSucceeded)
		require.NotNil(t, succeeded)
		assert.Equal(t, metav1.ConditionUnknown, succeeded.Status)
		assert.Equal(t, reasonProviderBackupResourceDoesNotExist, succeeded.Reason)
		assert.Equal(t, "Provider backups of all 4 units exist.", succeeded.Message)
		assert.False(t, backup.Status.StartTimestamp.IsZero())

		assert.Equal(t, 4, counter.veleroBackupCreateCount)
		assert.Equal(t, 1, counter.subResourcePatchCount)

		assert.Equal(t, backupv1.BackupCoverageFull, backup.Status.BackupCoverage)
		require.NotNil(t, backup.Status.GlobalConfig)
		assert.Equal(t, &backupv1.ProviderRef{Name: "backup-global-config"}, backup.Status.GlobalConfig.ProviderRef)
		assert.Equal(t, "ces.example.com", backup.Status.GlobalConfig.FQDN)
		require.NotNil(t, backup.Status.V2Dogus)
		assert.Equal(t, &backupv1.ProviderRef{Name: "backup-doguv2"}, backup.Status.V2Dogus.ProviderRef)
		require.Len(t, backup.Status.Dogus, 1)
		assert.Equal(t, &backupv1.ProviderRef{Name: "backup-dogu-ldap"}, backup.Status.Dogus[0].ProviderRef)
		require.NotNil(t, backup.Status.Misc)
		assert.Equal(t, &backupv1.ProviderRef{Name: "backup-misc"}, backup.Status.Misc.ProviderRef)

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
		assert.Equal(t, Next, nextAction)
		assert.WithinDuration(t, baseTime.Time, backup.Status.StartTimestamp.Time, time.Second)
	})

	t.Run("If all units exist, complete the unit status and proceed to the next step", func(t *testing.T) {
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
		// The status of a previous run may not have been written, e.g. because the patch failed.
		assert.Equal(t, 1, counter.subResourcePatchCount)
		require.NotNil(t, backup.Status.Misc)
		assert.Equal(t, &backupv1.ProviderRef{Name: "backup-misc"}, backup.Status.Misc.ProviderRef)
		assert.Nil(t, meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionSucceeded))
		assert.True(t, backup.Status.StartTimestamp.IsZero())
	})

	t.Run("If all units exist and the status is complete, proceed without patching the status", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		counter := &callCounter{}
		collector := newMockBackupStateCollector(t)
		collector.EXPECT().Collect(mock.Anything, backup).Return(newBackupStateForTest(backup.Name), nil)
		reconciler, _ := newReconcilerForProviderBackupCreatedTest(t, counter, collector, backup)

		_, err := reconciler.ensureProviderBackupCreated(context.Background(), backup)
		require.NoError(t, err)
		require.Equal(t, 1, counter.subResourcePatchCount)

		nextAction, err := reconciler.ensureProviderBackupCreated(context.Background(), backup)

		require.NoError(t, err)
		assert.Equal(t, Next, nextAction)
		assert.Equal(t, 1, counter.subResourcePatchCount)
	})

	t.Run("Keep the state of units which are already finished", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		backup.Status.Dogus = []backupv1.DoguUnitStatus{{
			Name: "ldap", Version: "2.6.8-1",
			State:       backupv1.UnitStateSucceeded,
			Reason:      "ProviderBackupSucceeded",
			Message:     "done",
			ProviderRef: &backupv1.ProviderRef{Name: "backup-dogu-ldap"},
		}}
		expectedDogus := backup.DeepCopy().Status.Dogus
		counter := &callCounter{}
		collector := newMockBackupStateCollector(t)
		collector.EXPECT().Collect(mock.Anything, backup).Return(newBackupStateForTest(backup.Name), nil)
		reconciler, _ := newReconcilerForProviderBackupCreatedTest(t, counter, collector,
			backup,
			newUnitVeleroBackupForReconcilerTest("ns", "backup-dogu-ldap", backup.Name, velerov1.BackupPhaseCompleted),
		)

		nextAction, err := reconciler.ensureProviderBackupCreated(context.Background(), backup)

		require.NoError(t, err)
		assert.Equal(t, Next, nextAction)
		assert.Equal(t, 3, counter.veleroBackupCreateCount)
		assert.Equal(t, expectedDogus, backup.Status.Dogus)
		require.NotNil(t, backup.Status.Misc)
		assert.Empty(t, backup.Status.Misc.State)
		assert.Equal(t, &backupv1.ProviderRef{Name: "backup-misc"}, backup.Status.Misc.ProviderRef)
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
		assert.Equal(t, Next, nextAction)
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
		assert.Equal(t, Next, nextAction)
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
		assert.Equal(t, 1, counter.subResourcePatchCount)
		require.NotNil(t, backup.Status.Misc)
		assert.Equal(t, &backupv1.ProviderRef{Name: "backup-misc"}, backup.Status.Misc.ProviderRef)
		assert.True(t, backup.Status.StartTimestamp.IsZero())
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

	t.Run("If no unit can be created, record the failures and proceed to the next step", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		counter := &callCounter{createCallError: errors.New("create error")}
		collector := newMockBackupStateCollector(t)
		collector.EXPECT().Collect(mock.Anything, backup).Return(newBackupStateForTest(backup.Name), nil)
		reconciler, _ := newReconcilerForProviderBackupCreatedTest(t, counter, collector, backup)

		nextAction, err := reconciler.ensureProviderBackupCreated(context.Background(), backup)

		require.NoError(t, err)
		assert.Equal(t, Next, nextAction)
		// The metadata of the units and the failure are written, but no unit gets a provider reference.
		assert.Equal(t, 1, counter.subResourcePatchCount)
		succeeded := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionSucceeded)
		require.NotNil(t, succeeded)
		assert.Equal(t, metav1.ConditionUnknown, succeeded.Status)
		assert.Equal(t, reasonProviderBackupCreationFailed, succeeded.Reason)
		assert.Equal(t, "Failed to create the provider backups of units backup-doguv2 (create error), backup-dogu-ldap (create error), backup-global-config (create error), backup-misc (create error). Provider backups of 0 of 4 units exist.", succeeded.Message)
		assertEventRecorded(t, reconciler, "Warning ProviderBackupCreationFailed Failed to create the provider backup of unit backup-doguv2: create error")
		assertEventRecorded(t, reconciler, "Warning ProviderBackupCreationFailed Failed to create the provider backup of unit backup-misc: create error")
		assert.True(t, backup.Status.StartTimestamp.IsZero())
		require.NotNil(t, backup.Status.Misc)
		assert.Nil(t, backup.Status.Misc.ProviderRef)
		require.Len(t, backup.Status.Dogus, 1)
		assert.Nil(t, backup.Status.Dogus[0].ProviderRef)
	})

	t.Run("If creating a unit fails, create the remaining units, set the start time and proceed to the next step", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		counter := &callCounter{}
		collector := newMockBackupStateCollector(t)
		collector.EXPECT().Collect(mock.Anything, backup).Return(newBackupStateForTest(backup.Name), nil)
		reconciler, _ := newReconcilerForProviderBackupCreatedTest(t, counter, collector, backup)
		createCalls := 0
		reconciler.client = interceptCreate(reconciler.client, func() error {
			createCalls++
			if createCalls == 1 {
				return errors.New("create error")
			}
			return nil
		})

		nextAction, err := reconciler.ensureProviderBackupCreated(context.Background(), backup)

		require.NoError(t, err)
		assert.Equal(t, Next, nextAction)
		assert.Equal(t, 3, counter.veleroBackupCreateCount)
		assert.Equal(t, 1, counter.subResourcePatchCount)
		assert.False(t, backup.Status.StartTimestamp.IsZero())
		succeeded := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionSucceeded)
		require.NotNil(t, succeeded)
		assert.Equal(t, reasonProviderBackupCreationFailed, succeeded.Reason)
		assert.Equal(t, "Failed to create the provider backups of units backup-doguv2 (create error). Provider backups of 3 of 4 units exist.", succeeded.Message)
		require.NotNil(t, backup.Status.V2Dogus)
		assert.Nil(t, backup.Status.V2Dogus.ProviderRef)
		require.NotNil(t, backup.Status.Misc)
		assert.Equal(t, &backupv1.ProviderRef{Name: "backup-misc"}, backup.Status.Misc.ProviderRef)
	})

	t.Run("Record the failures of all units which could not be created", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		counter := &callCounter{}
		collector := newMockBackupStateCollector(t)
		collector.EXPECT().Collect(mock.Anything, backup).Return(newBackupStateForTest(backup.Name), nil)
		reconciler, _ := newReconcilerForProviderBackupCreatedTest(t, counter, collector, backup)
		createCalls := 0
		reconciler.client = interceptCreate(reconciler.client, func() error {
			createCalls++
			if createCalls%2 == 0 {
				return fmt.Errorf("create error %d", createCalls)
			}
			return nil
		})

		nextAction, err := reconciler.ensureProviderBackupCreated(context.Background(), backup)

		require.NoError(t, err)
		assert.Equal(t, Next, nextAction)
		assert.Equal(t, 2, counter.veleroBackupCreateCount)
		assertEventRecorded(t, reconciler, "Warning ProviderBackupCreationFailed Failed to create the provider backup of unit backup-dogu-ldap: create error 2")
		assertEventRecorded(t, reconciler, "Warning ProviderBackupCreationFailed Failed to create the provider backup of unit backup-misc: create error 4")
		succeeded := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionSucceeded)
		require.NotNil(t, succeeded)
		assert.Equal(t, "Failed to create the provider backups of units backup-dogu-ldap (create error 2), backup-misc (create error 4). Provider backups of 2 of 4 units exist.", succeeded.Message)
	})

	t.Run("Replace the creation failure of a previous run once all units exist", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		backup.Status.Conditions = []metav1.Condition{{
			Type:    backupv1.ConditionSucceeded,
			Status:  metav1.ConditionUnknown,
			Reason:  reasonProviderBackupCreationFailed,
			Message: "Failed to create the provider backups of units backup-misc (timeout). Provider backups of 3 of 4 units exist.",
		}}
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
		succeeded := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionSucceeded)
		require.NotNil(t, succeeded)
		assert.Equal(t, reasonProviderBackupResourceDoesNotExist, succeeded.Reason)
		assert.Equal(t, "Provider backups of all 4 units exist.", succeeded.Message)
	})

	t.Run("Create a unit which failed in a previous run in the next run", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		counter := &callCounter{}
		collector := newMockBackupStateCollector(t)
		collector.EXPECT().Collect(mock.Anything, backup).Return(newBackupStateForTest(backup.Name), nil)
		reconciler, _ := newReconcilerForProviderBackupCreatedTest(t, counter, collector, backup)
		failFirstCreate := true
		reconciler.client = interceptCreate(reconciler.client, func() error {
			if failFirstCreate {
				failFirstCreate = false
				return errors.New("create error")
			}
			return nil
		})

		_, err := reconciler.ensureProviderBackupCreated(context.Background(), backup)
		require.NoError(t, err)
		require.Equal(t, 3, counter.veleroBackupCreateCount)

		nextAction, err := reconciler.ensureProviderBackupCreated(context.Background(), backup)

		require.NoError(t, err)
		assert.Equal(t, Next, nextAction)
		assert.Equal(t, 4, counter.veleroBackupCreateCount)
		require.NotNil(t, backup.Status.V2Dogus)
		assert.Equal(t, &backupv1.ProviderRef{Name: "backup-doguv2"}, backup.Status.V2Dogus.ProviderRef)
		succeeded := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionSucceeded)
		require.NotNil(t, succeeded)
		assert.Equal(t, reasonProviderBackupResourceDoesNotExist, succeeded.Reason)
		assert.Equal(t, "Provider backups of all 4 units exist.", succeeded.Message)
	})

	t.Run("Do not patch the status again if the same creation failure occurs", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		counter := &callCounter{createCallError: errors.New("create error")}
		collector := newMockBackupStateCollector(t)
		collector.EXPECT().Collect(mock.Anything, backup).Return(newBackupStateForTest(backup.Name), nil)
		reconciler, _ := newReconcilerForProviderBackupCreatedTest(t, counter, collector, backup)

		_, err := reconciler.ensureProviderBackupCreated(context.Background(), backup)
		require.NoError(t, err)
		require.Equal(t, 1, counter.subResourcePatchCount)

		nextAction, err := reconciler.ensureProviderBackupCreated(context.Background(), backup)

		require.NoError(t, err)
		assert.Equal(t, Next, nextAction)
		assert.Equal(t, 1, counter.subResourcePatchCount)
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

func assertEventRecorded(t *testing.T, reconciler *defaultReconciler, expected string) {
	t.Helper()
	recorder, ok := reconciler.recorder.(*events.FakeRecorder)
	require.True(t, ok)
	for {
		select {
		case event := <-recorder.Events:
			if strings.HasPrefix(event, expected) {
				return
			}
		default:
			assert.Failf(t, "event not recorded", "expected an event starting with %q", expected)
			return
		}
	}
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
