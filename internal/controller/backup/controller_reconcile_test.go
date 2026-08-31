package backup

import (
	"context"
	"testing"
	"time"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	blueprintv3 "github.com/cloudogu/k8s-blueprint-lib/v3/api/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	coordinationv1 "k8s.io/api/coordination/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

const requeueAfterTest = time.Duration(5) * time.Second

func TestControllerReconcileStageOrder(t *testing.T) {
	createStages := []string{
		"ensureVeleroStatusSynced",
		"ensureBackupSetup",
		"ensureBackupIsCanceledAfterTimeWindowExpired",
		"ensureBackupIsPrepared",
		"ensureActiveBackupLease",
		"ensureMaintenanceActivated",
		"ensureProviderBackupCreated",
		"ensureProviderBackupCompleted",
		"ensureMaintenanceDeactivated",
		"ensureBackupLeaseReleased",
		"ensureBackupRunCompleted",
	}
	finalizeStages := []string{
		"ensureMaintenanceDeactivated",
		"ensureBackupLeaseReleased",
		"ensureBackupRunCompleted",
	}

	ignoreStages := []string{
		"ensureOrphanedBackupDeleted",
	}

	deleteStages := []string{
		"ensureMaintenanceDeactivated",
		"ensureBackupLeaseReleased",
		"ensureProviderBackupDeleted",
	}

	tests := []struct {
		name     string
		backup   *backupv1.Backup
		expected []string
	}{
		{
			name:     "run a fresh backup through the create stages",
			backup:   newBackupForTest("ns", "backup"),
			expected: createStages,
		},
		{
			name:     "run a backup whose provider result is in through the create stages",
			backup:   withCondition(newBackupForTest("ns", "backup"), backupv1.ConditionProviderSucceeded, metav1.ConditionUnknown),
			expected: createStages,
		},
		{
			name:     "ignore a backup that already completed",
			backup:   withCondition(newBackupForTest("ns", "backup"), backupv1.ConditionSucceeded, metav1.ConditionTrue),
			expected: ignoreStages,
		},
		{
			name:     "ignore a backup that already failed",
			backup:   withCondition(newBackupForTest("ns", "backup"), backupv1.ConditionSucceeded, metav1.ConditionFalse),
			expected: ignoreStages,
		},
		{
			name:     "finalize a backup whose provider backup succeeded",
			backup:   withCondition(newBackupForTest("ns", "backup"), backupv1.ConditionProviderSucceeded, metav1.ConditionTrue),
			expected: finalizeStages,
		},
		{
			name:     "finalize a backup whose provider backup failed",
			backup:   withCondition(newBackupForTest("ns", "backup"), backupv1.ConditionProviderSucceeded, metav1.ConditionFalse),
			expected: finalizeStages,
		},
		{
			name:     "finalize a canceled backup",
			backup:   withCondition(newBackupForTest("ns", "backup"), backupv1.ConditionCanceled, metav1.ConditionTrue),
			expected: finalizeStages,
		},
		{
			name:     "delete a backup",
			backup:   withDeletionTimestamp(newBackupForTest("ns", "backup")),
			expected: deleteStages,
		},
		{
			// Deletion wins over every other state.
			name:     "delete a backup that already completed",
			backup:   withDeletionTimestamp(withCondition(newBackupForTest("ns", "backup"), backupv1.ConditionSucceeded, metav1.ConditionTrue)),
			expected: deleteStages,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fakeClient := newFakeClientBuilder(t).WithObjects(test.backup).Build()
			reconcilerMock := newMockReconciler(t)
			var executed []string
			recordStages(reconcilerMock, &executed)
			controller := NewController(fakeClient, reconcilerMock, requeueAfterTest)

			result, err := controller.Reconcile(context.Background(), newReconcilerRequest("ns", "backup"))

			assert.NoError(t, err)
			assert.Equal(t, ctrl.Result{}, result)
			assert.Equal(t, test.expected, executed)
		})
	}
}

func TestControllerReconcile(t *testing.T) {
	t.Run("If there is no backup do nothing", func(t *testing.T) {
		fakeClient := newFakeClientBuilder(t).Build()
		// We set the service to nil to check if the controller calls any method of the reconciler.
		controller := NewController(fakeClient, nil, requeueAfterTest)

		result, err := controller.Reconcile(context.Background(), newReconcilerRequest("ns", "backup"))

		assert.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, result)
	})

	t.Run("synchronize a provider backup before the normal backup process", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		backup.Spec.SyncedFromProvider = true
		fakeClient := newFakeClientBuilder(t).WithObjects(backup).Build()
		reconcilerMock := newMockReconciler(t)
		reconcilerMock.EXPECT().
			ensureVeleroStatusSynced(context.Background(), mock.Anything).
			Return(Abort, nil)
		allowRemainingStages(reconcilerMock)
		controller := NewController(fakeClient, reconcilerMock, requeueAfterTest)

		result, err := controller.Reconcile(context.Background(), newReconcilerRequest("ns", "backup"))

		assert.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, result)
	})

	t.Run("a failure while synchronizing the provider backup status reports the error", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		backup.Spec.SyncedFromProvider = true
		fakeClient := newFakeClientBuilder(t).WithObjects(backup).Build()
		reconcilerMock := newMockReconciler(t)
		reconcilerMock.EXPECT().
			ensureVeleroStatusSynced(context.Background(), mock.Anything).
			Return(Retry, assert.AnError)
		allowRemainingStages(reconcilerMock)
		controller := NewController(fakeClient, reconcilerMock, requeueAfterTest)

		result, err := controller.Reconcile(context.Background(), newReconcilerRequest("ns", "backup"))

		assert.ErrorIs(t, err, assert.AnError)
		assert.Equal(t, ctrl.Result{}, result)
	})

	t.Run("a failing stage stops the pipeline and reports the error", func(t *testing.T) {
		reconcilerMock, controller := newTestFixtureForControllerTest(t)
		reconcilerMock.EXPECT().
			ensureBackupIsPrepared(context.Background(), mock.Anything).
			Return(Retry, assert.AnError)
		allowRemainingStages(reconcilerMock)

		result, err := controller.Reconcile(context.Background(), newReconcilerRequest("ns", "backup"))

		assert.Equal(t, assert.AnError, err)
		assert.Equal(t, ctrl.Result{}, result)
	})

	t.Run("an aborting stage stops the pipeline without requeueing", func(t *testing.T) {
		reconcilerMock, controller := newTestFixtureForControllerTest(t)
		reconcilerMock.EXPECT().
			ensureBackupIsPrepared(context.Background(), mock.Anything).
			Return(Abort, assert.AnError)
		allowRemainingStages(reconcilerMock)

		result, err := controller.Reconcile(context.Background(), newReconcilerRequest("ns", "backup"))

		assert.Equal(t, assert.AnError, err)
		assert.Equal(t, ctrl.Result{}, result)
	})

	t.Run("a retrying stage without an error still requeues", func(t *testing.T) {
		reconcilerMock, controller := newTestFixtureForControllerTest(t)
		reconcilerMock.EXPECT().
			ensureMaintenanceActivated(context.Background(), mock.Anything).
			Return(Retry, nil)
		allowRemainingStages(reconcilerMock)

		result, err := controller.Reconcile(context.Background(), newReconcilerRequest("ns", "backup"))

		assert.NoError(t, err)
		assert.Equal(t, ctrl.Result{RequeueAfter: requeueAfterTest}, result)
	})
}

func TestRequiredOperation(t *testing.T) {
	tests := []struct {
		name     string
		backup   *backupv1.Backup
		expected operation
	}{
		{"fresh backup", newBackupForTest("ns", "backup"), operationCreate},
		{"running provider backup", withCondition(newBackupForTest("ns", "backup"), backupv1.ConditionProviderSucceeded, metav1.ConditionUnknown), operationCreate},
		{"provider backup succeeded", withCondition(newBackupForTest("ns", "backup"), backupv1.ConditionProviderSucceeded, metav1.ConditionTrue), operationFinalize},
		{"provider backup failed", withCondition(newBackupForTest("ns", "backup"), backupv1.ConditionProviderSucceeded, metav1.ConditionFalse), operationFinalize},
		{"canceled backup", withCondition(newBackupForTest("ns", "backup"), backupv1.ConditionCanceled, metav1.ConditionTrue), operationFinalize},
		{"completed backup", withCondition(newBackupForTest("ns", "backup"), backupv1.ConditionSucceeded, metav1.ConditionTrue), operationIgnore},
		{"failed backup", withCondition(newBackupForTest("ns", "backup"), backupv1.ConditionSucceeded, metav1.ConditionFalse), operationIgnore},
		{"deleting backup", withDeletionTimestamp(newBackupForTest("ns", "backup")), operationDelete},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, requiredOperation(test.backup))
		})
	}
}

// recordStages lets every stage record that it ran and continue, so that a test can assert the
// exact sequence the controller executed.
func recordStages(reconcilerMock *mockReconciler, executed *[]string) {
	record := func(name string) func(context.Context, *backupv1.Backup) (action, error) {
		return func(context.Context, *backupv1.Backup) (action, error) {
			*executed = append(*executed, name)
			return Next, nil
		}
	}

	expecter := reconcilerMock.EXPECT()
	expecter.ensureBackupLeaseReleased(mock.Anything, mock.Anything).RunAndReturn(record("ensureBackupLeaseReleased")).Maybe()
	expecter.ensureProviderBackupDeleted(mock.Anything, mock.Anything).RunAndReturn(record("ensureProviderBackupDeleted")).Maybe()
	expecter.ensureOrphanedBackupDeleted(mock.Anything, mock.Anything).RunAndReturn(record("ensureOrphanedBackupDeleted")).Maybe()
	expecter.ensureVeleroStatusSynced(mock.Anything, mock.Anything).RunAndReturn(record("ensureVeleroStatusSynced")).Maybe()
	expecter.ensureBackupSetup(mock.Anything, mock.Anything).RunAndReturn(record("ensureBackupSetup")).Maybe()
	expecter.ensureBackupIsCanceledAfterTimeWindowExpired(mock.Anything, mock.Anything).RunAndReturn(record("ensureBackupIsCanceledAfterTimeWindowExpired")).Maybe()
	expecter.ensureBackupIsPrepared(mock.Anything, mock.Anything).RunAndReturn(record("ensureBackupIsPrepared")).Maybe()
	expecter.ensureActiveBackupLease(mock.Anything, mock.Anything).RunAndReturn(record("ensureActiveBackupLease")).Maybe()
	expecter.ensureMaintenanceActivated(mock.Anything, mock.Anything).RunAndReturn(record("ensureMaintenanceActivated")).Maybe()
	expecter.ensureProviderBackupCreated(mock.Anything, mock.Anything).RunAndReturn(record("ensureProviderBackupCreated")).Maybe()
	expecter.ensureProviderBackupCompleted(mock.Anything, mock.Anything).RunAndReturn(record("ensureProviderBackupCompleted")).Maybe()
	expecter.ensureMaintenanceDeactivated(mock.Anything, mock.Anything).RunAndReturn(record("ensureMaintenanceDeactivated")).Maybe()
	expecter.ensureBackupRunCompleted(mock.Anything, mock.Anything).RunAndReturn(record("ensureBackupRunCompleted")).Maybe()
}

func withCondition(backup *backupv1.Backup, conditionType string, status metav1.ConditionStatus) *backupv1.Backup {
	meta.SetStatusCondition(&backup.Status.Conditions, metav1.Condition{
		Type:   conditionType,
		Status: status,
		Reason: "aReason",
	})
	return backup
}

func withDeletionTimestamp(backup *backupv1.Backup) *backupv1.Backup {
	now := metav1.Now()
	backup.DeletionTimestamp = &now
	backup.Finalizers = []string{backupv1.BackupFinalizer}
	return backup
}

func newBackupForTest(namespace string, name string) *backupv1.Backup {
	return &backupv1.Backup{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Spec: backupv1.BackupSpec{
			Provider: "velero",
		},
	}
}

func newFakeClientBuilder(t *testing.T) *fake.ClientBuilder {
	scheme := runtime.NewScheme()
	require.NoError(t, backupv1.AddToScheme(scheme))
	require.NoError(t, blueprintv3.AddToScheme(scheme))
	require.NoError(t, velerov1.AddToScheme(scheme))
	require.NoError(t, coordinationv1.AddToScheme(scheme))
	require.NoError(t, corev1.AddToScheme(scheme))

	return fake.NewClientBuilder().WithScheme(scheme)
}

func newFakeClientBuilderWithCounter(t *testing.T, callCounter *callCounter) *fake.ClientBuilder {
	return newFakeClientBuilder(t).
		WithInterceptorFuncs(interceptor.Funcs{
			Get:              callCounter.getCall,
			SubResourcePatch: callCounter.subResourcePatchCall,
			Create:           callCounter.createCall,
		})
}

func newReconcilerRequest(namespace string, name string) ctrl.Request {
	return ctrl.Request{NamespacedName: types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}}
}

func newTestFixtureForControllerTest(t *testing.T) (*mockReconciler, *Controller) {
	backup := newBackupForTest("ns", "backup")
	fakeClient := newFakeClientBuilder(t).
		WithObjects(backup).
		Build()

	reconcilerMock := newMockReconciler(t)
	controller := NewController(fakeClient, reconcilerMock, requeueAfterTest)
	return reconcilerMock, controller
}

// allowRemainingStages lets every stage that a test did not pin down succeed. Register the pinned
// stage first: the first matching expectation counts.
func allowRemainingStages(reconcilerMock *mockReconciler) {
	var ignored []string
	recordStages(reconcilerMock, &ignored)
}

// A backup synchronized from the provider owns neither the lease nor the maintenance mode. Both of
// its conditions therefore reach their terminal state together, which routes it to operationIgnore
// and keeps it away from every maintenance stage.
func TestRequiredOperationForSynchronizedBackup(t *testing.T) {
	tests := []struct {
		name     string
		status   metav1.ConditionStatus
		expected operation
	}{
		{"running", metav1.ConditionUnknown, operationCreate},
		{"completed", metav1.ConditionTrue, operationIgnore},
		{"failed", metav1.ConditionFalse, operationIgnore},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			backup := newBackupForTest("ns", "backup")
			backup.Spec.SyncedFromProvider = true
			withCondition(backup, backupv1.ConditionProviderSucceeded, test.status)
			withCondition(backup, backupv1.ConditionSucceeded, test.status)

			assert.Equal(t, test.expected, requiredOperation(backup))
		})
	}
}

// The maintenance gateway mock has no expectations, so any call to it fails the test.
func TestControllerReconcileDoesNotStealTheMaintenanceMode(t *testing.T) {
	ctx := context.Background()

	// Backup A is running: it owns the lease and therefore the active maintenance mode.
	newRunningBackupWithLease := func() (*backupv1.Backup, client.Object) {
		runningBackup := newBackupForTest("ns", "running-backup")
		return runningBackup, newHeldBackupLeaseForTest(runningBackup)
	}

	t.Run("a completed backup does not touch the maintenance mode of a running backup", func(t *testing.T) {
		runningBackup, lease := newRunningBackupWithLease()
		completedBackup := newBackupWithSucceededStatusForReconcilerTest("ns", "completed-backup", metav1.ConditionTrue)
		fakeClient := newFakeClientBuilder(t).
			WithObjects(runningBackup, completedBackup, lease).
			WithStatusSubresource(completedBackup).
			Build()
		maintenanceGatewayMock := newMockMaintenanceGateway(t)
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), maintenanceGatewayMock, newRealClock(), "default")
		controller := NewController(fakeClient, reconciler, requeueAfterTest)

		result, err := controller.Reconcile(ctx, newReconcilerRequest("ns", "completed-backup"))
		require.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, result)

		assertBackupLeaseStillHeldBy(t, fakeClient, runningBackup)
	})

	t.Run("a canceled backup that never started does not touch the maintenance mode of a running backup", func(t *testing.T) {
		runningBackup, lease := newRunningBackupWithLease()
		canceledBackup := withCondition(newBackupForTest("ns", "canceled-backup"), backupv1.ConditionCanceled, metav1.ConditionTrue)
		fakeClient := newFakeClientBuilder(t).
			WithObjects(runningBackup, canceledBackup, lease).
			WithStatusSubresource(canceledBackup).
			Build()
		maintenanceGatewayMock := newMockMaintenanceGateway(t)
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), maintenanceGatewayMock, newRealClock(), "default")
		controller := NewController(fakeClient, reconciler, requeueAfterTest)

		_, err := controller.Reconcile(ctx, newReconcilerRequest("ns", "canceled-backup"))
		require.NoError(t, err)

		assertBackupLeaseStillHeldBy(t, fakeClient, runningBackup)

		// The finalization still has to close the canceled run, otherwise it stays in the finalize
		// branch forever and re-enters this stage on every operator restart.
		stored := &backupv1.Backup{}
		require.NoError(t, fakeClient.Get(ctx, client.ObjectKeyFromObject(canceledBackup), stored))
		succeeded := meta.FindStatusCondition(stored.Status.Conditions, backupv1.ConditionSucceeded)
		require.NotNil(t, succeeded)
		assert.Equal(t, metav1.ConditionFalse, succeeded.Status)
		assert.Equal(t, reasonBackupCanceled, succeeded.Reason)
	})

	t.Run("deleting a backup deactivates the maintenance mode it holds before releasing its lease", func(t *testing.T) {
		deletedBackup := withDeletionTimestamp(newBackupForTest("ns", "deleted-backup"))
		lease := newHeldBackupLeaseForTest(deletedBackup)
		fakeClient := newFakeClientBuilder(t).
			WithObjects(deletedBackup, lease).
			WithStatusSubresource(deletedBackup).
			Build()
		maintenanceGatewayMock := newMockMaintenanceGateway(t)
		maintenanceGatewayMock.EXPECT().isMaintenanceModeActive(ctx).Return(true, nil)
		maintenanceGatewayMock.EXPECT().deactivateMaintenanceMode(ctx).RunAndReturn(func(ctx context.Context) error {
			// The lease is what proves ownership of the maintenance mode, so it must still be held
			// while the maintenance mode is switched off.
			assertBackupLeaseStillHeldBy(t, fakeClient, deletedBackup)
			return nil
		})
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), maintenanceGatewayMock, newRealClock(), "default")
		controller := NewController(fakeClient, reconciler, requeueAfterTest)

		_, err := controller.Reconcile(ctx, newReconcilerRequest("ns", "deleted-backup"))
		require.NoError(t, err)

		assertBackupLeaseReleased(t, fakeClient, deletedBackup)

		// No provider backup exists, so the deletion path removes the finalizer and the Backup is gone.
		stored := &backupv1.Backup{}
		err = fakeClient.Get(ctx, client.ObjectKeyFromObject(deletedBackup), stored)
		assert.Error(t, err, "expected the backup to be gone after its finalizer was removed")
	})

	t.Run("deleting a backup that does not hold the lease leaves the maintenance mode untouched", func(t *testing.T) {
		runningBackup, lease := newRunningBackupWithLease()
		deletedBackup := withDeletionTimestamp(newBackupForTest("ns", "deleted-backup"))
		fakeClient := newFakeClientBuilder(t).
			WithObjects(runningBackup, deletedBackup, lease).
			WithStatusSubresource(deletedBackup).
			Build()
		maintenanceGatewayMock := newMockMaintenanceGateway(t)
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), maintenanceGatewayMock, newRealClock(), "default")
		controller := NewController(fakeClient, reconciler, requeueAfterTest)

		_, err := controller.Reconcile(ctx, newReconcilerRequest("ns", "deleted-backup"))
		require.NoError(t, err)

		assertBackupLeaseStillHeldBy(t, fakeClient, runningBackup)
		// backup is gone
		stored := &backupv1.Backup{}
		err = fakeClient.Get(ctx, client.ObjectKeyFromObject(deletedBackup), stored)
		assert.Error(t, err, "expected the backup to be gone after its finalizer was removed")
	})

	t.Run("the lease holder deactivates the maintenance mode it owns", func(t *testing.T) {
		finishedBackup := newBackupWithProviderSucceededStatusForReconcilerTest("ns", "finished-backup", metav1.ConditionTrue)
		lease := newHeldBackupLeaseForTest(finishedBackup)
		fakeClient := newFakeClientBuilder(t).
			WithObjects(finishedBackup, lease).
			WithStatusSubresource(finishedBackup).
			Build()
		maintenanceGatewayMock := newMockMaintenanceGateway(t)
		maintenanceGatewayMock.EXPECT().isMaintenanceModeActive(ctx).Return(true, nil)
		maintenanceGatewayMock.EXPECT().deactivateMaintenanceMode(ctx).Return(nil)
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), maintenanceGatewayMock, newRealClock(), "default")
		controller := NewController(fakeClient, reconciler, requeueAfterTest)

		_, err := controller.Reconcile(ctx, newReconcilerRequest("ns", "finished-backup"))
		require.NoError(t, err)

		assertBackupLeaseReleased(t, fakeClient, finishedBackup)

		stored := &backupv1.Backup{}
		require.NoError(t, fakeClient.Get(ctx, client.ObjectKeyFromObject(finishedBackup), stored))
		succeeded := meta.FindStatusCondition(stored.Status.Conditions, backupv1.ConditionSucceeded)
		require.NotNil(t, succeeded)
		assert.Equal(t, metav1.ConditionTrue, succeeded.Status)
	})
}
