package restore

import (
	"context"
	"testing"

	"github.com/cloudogu/k8s-registry-lib/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	k8sv1 "github.com/cloudogu/k8s-backup-lib/api/v1"

	"github.com/cloudogu/k8s-backup-operator/internal/provider/velero"
)

// CleanUp,Scale & MaintenanceMock will fail on any calls
func newSourceBackupGateFactory(t *testing.T, recorder eventRecorder) reconcileFactory {
	t.Helper()

	return func(fakeClient client.WithWatch) reconcileFunction {
		reconciler := NewRestoreReconciler(fakeClient, recorder, testNamespace, newMockCleanupManager(t), newMockScaleManager(t),
			requeueAfterTest, testBackupStorage, testProviderDeployment)
		reconciler.maintenanceModeSwitch = newMockMaintenanceModeSwitch(t)

		return reconciler.Reconcile
	}
}

func TestAMissingSourceBackupFailsTheRestoreBeforeMaintenanceAndPreparation(t *testing.T) {
	restore := withInitializedConditions(withMetadata(newParentRestore()))

	recorderMock := newMockEventRecorder(t)
	recorderMock.EXPECT().Event(matchesRestoreNamed(testRestore), corev1.EventTypeWarning,
		velero.ReasonVeleroSourceBackupNotUsable,
		"The velero backup 'name=test-backup' to be restored was not found.").Once()

	// no backup in client
	fixture := newMultiReconcileFixture(t, interceptor.Funcs{}, newSourceBackupGateFactory(t, recorderMock),
		restore, readyStorageLocation(), readyProviderDeployment())

	results, errs := fixture.reconcileTimes(testCtx, newRestoreRequest(testRestore), 2)

	for index := range results {
		require.NoError(t, errs[index], "a backup that cannot be restored is a decided outcome, not an error")
		assert.Equal(t, ctrl.Result{}, results[index], "a terminal restore must not be requeued")
	}
	assert.Equal(t, []recordedClientAction{statusUpdateOf(restore), deleteOf(newRestoreLease(restore))}, fixture.clientActions.snapshot(),
		"the missing backup must be reported exactly once and the ecosystem must stay untouched")
	assertPreparedCondition(t, fixture.client, metav1.ConditionFalse, velero.ReasonVeleroSourceBackupNotUsable)
	assertSuccessfulCondition(t, fixture.client, testRestore, metav1.ConditionFalse, velero.ReasonVeleroSourceBackupNotUsable)
}

func TestASourceBackupThatIsNotRestorableFailsTheRestore(t *testing.T) {
	phases := []velerov1.BackupPhase{
		velerov1.BackupPhaseFailed,
		velerov1.BackupPhaseFailedValidation,
		velerov1.BackupPhaseDeleting,
		// the workflow holds the blocking lease, so an unfinished backup is abandoned or hand-made
		velerov1.BackupPhaseInProgress,
		// a phase a later velero version may add is not restorable either
		"SomePhaseALaterVeleroAdded",
	}

	for _, phase := range phases {
		t.Run(string(phase), func(t *testing.T) {
			restore := withInitializedConditions(withMetadata(newParentRestore()))

			recorderMock := newMockEventRecorder(t)
			recorderMock.EXPECT().Event(matchesRestoreNamed(testRestore), corev1.EventTypeWarning, velero.ReasonVeleroSourceBackupNotUsable, mock.Anything).Once()

			fixture := newMultiReconcileFixture(t, interceptor.Funcs{}, newSourceBackupGateFactory(t, recorderMock),
				restore, readyStorageLocation(), readyProviderDeployment(), sourceBackup(phase))

			results, errs := fixture.reconcileTimes(testCtx, newRestoreRequest(testRestore), 2)

			for index := range results {
				require.NoError(t, errs[index], "a backup that cannot be restored is a decided outcome, not an error")
				assert.Equal(t, ctrl.Result{}, results[index], "a terminal restore must not be requeued")
			}
			assert.Equal(t, []recordedClientAction{statusUpdateOf(restore), deleteOf(newRestoreLease(restore))}, fixture.clientActions.snapshot(),
				"the backup must be reported exactly once and the ecosystem must stay untouched")
			assertPreparedCondition(t, fixture.client, metav1.ConditionFalse, velero.ReasonVeleroSourceBackupNotUsable)
			assertSuccessfulCondition(t, fixture.client, testRestore, metav1.ConditionFalse, velero.ReasonVeleroSourceBackupNotUsable)
		})
	}
}

func TestARestorableSourceBackupLetsThePreparationRun(t *testing.T) {
	restore := withInitializedConditions(withMetadata(newParentRestore()))

	recorderMock := newMockEventRecorder(t)
	recorderMock.EXPECT().Event(matchesRestoreNamed(testRestore), corev1.EventTypeNormal, mock.Anything, mock.Anything).Times(3)

	scaleMock := newMockScaleManager(t)
	scaleMock.EXPECT().ScaleDown(testCtx).Return(nil).Once()
	cleanupMock := newMockCleanupManager(t)
	cleanupMock.EXPECT().Cleanup(testCtx).Return(nil).Once()
	maintenanceMock := newMockMaintenanceModeSwitch(t)
	maintenanceMock.EXPECT().GetStatus(testCtx).Return(repository.MaintenanceModeDescription{}, true, nil)

	factory := func(fakeClient client.WithWatch) reconcileFunction {
		reconciler := NewRestoreReconciler(fakeClient, recorderMock, testNamespace, cleanupMock, scaleMock,
			requeueAfterTest, testBackupStorage, testProviderDeployment)
		reconciler.maintenanceModeSwitch = maintenanceMock

		return reconciler.Reconcile
	}
	fixture := newMultiReconcileFixture(t, interceptor.Funcs{}, factory,
		restore, readyStorageLocation(), readyProviderDeployment(), sourceBackup(velerov1.BackupPhasePartiallyFailed))

	_, errs := fixture.reconcileTimes(testCtx, newRestoreRequest(testRestore), 1)

	require.NoError(t, errs[0])
	assertPreparedCondition(t, fixture.client, metav1.ConditionTrue, ReasonPreparationCompleted)
}

func TestAnAlreadyPreparedRestoreIsNotFailedByAMissingSourceBackup(t *testing.T) {
	restore := withPreparation(withInitializedConditions(withMetadata(newParentRestore())))

	recorderMock := newMockEventRecorder(t)
	recorderMock.EXPECT().Event(matchesRestoreNamed(testRestore), corev1.EventTypeNormal, mock.Anything, mock.Anything).Maybe()

	maintenanceMock := newMockMaintenanceModeSwitch(t)
	maintenanceMock.EXPECT().GetStatus(testCtx).Return(repository.MaintenanceModeDescription{}, true, nil)

	factory := func(fakeClient client.WithWatch) reconcileFunction {
		reconciler := NewRestoreReconciler(fakeClient, recorderMock, testNamespace, newMockCleanupManager(t), newMockScaleManager(t),
			requeueAfterTest, testBackupStorage, testProviderDeployment)
		reconciler.maintenanceModeSwitch = maintenanceMock

		return reconciler.Reconcile
	}
	// no backup in client
	fixture := newMultiReconcileFixture(t, interceptor.Funcs{}, factory, restore, readyStorageLocation(), readyProviderDeployment())

	_, errs := fixture.reconcileTimes(testCtx, newRestoreRequest(testRestore), 1)

	require.NoError(t, errs[0])
	assertPreparedCondition(t, fixture.client, metav1.ConditionTrue, ReasonPreparationCompleted)

	stored := &k8sv1.Restore{}
	require.NoError(t, fixture.client.Get(testCtx, client.ObjectKey{Namespace: testNamespace, Name: testRestore}, stored))
	assert.False(t, isTerminal(stored), "a prepared restore must not be failed by the source backup gate")
}

func TestAFailedSourceBackupReadIsRetriedWithoutTouchingTheEcosystem(t *testing.T) {
	restore := withInitializedConditions(withMetadata(newParentRestore()))

	failingBackupRead := interceptor.Funcs{
		Get: func(ctx context.Context, wrapped client.WithWatch, key client.ObjectKey, object client.Object, opts ...client.GetOption) error {
			if _, isBackup := object.(*velerov1.Backup); isBackup {
				return assert.AnError
			}

			return wrapped.Get(ctx, key, object, opts...)
		},
	}
	fixture := newMultiReconcileFixture(t, failingBackupRead, newSourceBackupGateFactory(t, newMockEventRecorder(t)),
		restore, readyStorageLocation(), readyProviderDeployment(), usableSourceBackup())

	results, errs := fixture.reconcileTimes(testCtx, newRestoreRequest(testRestore), 1)

	require.ErrorIs(t, errs[0], assert.AnError, "an unreadable backup is transient and must be retried")
	assert.Equal(t, ctrl.Result{}, results[0])
	assert.Empty(t, fixture.clientActions.snapshot(), "a failed read must not change anything")
}
