package restore

import (
	"context"
	"testing"

	"github.com/cloudogu/k8s-backup-operator/internal/provider/velero"
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

	"github.com/cloudogu/k8s-backup-operator/internal/controller/restore/state"
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
	recorderMock.EXPECT().Eventf(matchesCloudoguRestoreNamed(testRestore), nil, corev1.EventTypeWarning,
		ReasonSourceBackupNotFound, actionCheckSourceBackup,
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
	assertPreparedCondition(t, fixture.client, metav1.ConditionFalse, ReasonSourceBackupNotFound)
	assertSuccessfulCondition(t, fixture.client, testRestore, metav1.ConditionFalse, ReasonSourceBackupNotFound)
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
			recorderMock.EXPECT().Eventf(matchesCloudoguRestoreNamed(testRestore), nil, corev1.EventTypeWarning, ReasonUnusableUnits, mock.Anything, mock.Anything).Once()

			fixture := newMultiReconcileFixture(t, interceptor.Funcs{}, newSourceBackupGateFactory(t, recorderMock),
				restore, readyStorageLocation(), readyProviderDeployment(), sourceBackup(phase))

			results, errs := fixture.reconcileTimes(testCtx, newRestoreRequest(testRestore), 2)

			for index := range results {
				require.NoError(t, errs[index], "a backup that cannot be restored is a decided outcome, not an error")
				assert.Equal(t, ctrl.Result{}, results[index], "a terminal restore must not be requeued")
			}
			assert.Equal(t, []recordedClientAction{statusUpdateOf(restore), deleteOf(newRestoreLease(restore))}, fixture.clientActions.snapshot(),
				"the backup must be reported exactly once and the ecosystem must stay untouched")
			assertPreparedCondition(t, fixture.client, metav1.ConditionFalse, ReasonUnusableUnits)
			assertSuccessfulCondition(t, fixture.client, testRestore, metav1.ConditionFalse, ReasonUnusableUnits)
		})
	}
}

func TestAMalformedSourceBackupFailsTheRestore(t *testing.T) {
	restore := withInitializedConditions(withMetadata(newParentRestore()))
	malformed := &velerov1.Backup{
		Name: velero.UnitObjectName(testBackup, testGlobalConfigUnit), Namespace: testNamespace,
		Labels: map[string]string{
			k8sv1.BackupLabelKey:     k8sv1.GetTruncatedName(testBackup),
			k8sv1.BackupTypeLabelKey: string(testGlobalConfigUnit.Type),
		},
		Annotations: nil, // <- no Annotations
		Status:      velerov1.BackupStatus{Phase: velerov1.BackupPhaseCompleted},
	}

	recorderMock := newMockEventRecorder(t)
	recorderMock.EXPECT().Eventf(matchesCloudoguRestoreNamed(testRestore), nil, corev1.EventTypeWarning, ReasonUnusableUnits, actionCheckSourceBackup,
		"The backup cannot be read: collect units: provider backup test-backup-global-config has no annotation k8s.cloudogu.com/backup-units.").Once()

	fixture := newMultiReconcileFixture(t, interceptor.Funcs{}, newSourceBackupGateFactory(t, recorderMock),
		restore, readyStorageLocation(), readyProviderDeployment(), malformed)

	results, errs := fixture.reconcileTimes(testCtx, newRestoreRequest(testRestore), 2)

	for index := range results {
		require.NoError(t, errs[index], "a malformed backup is a decided outcome, not an error")
		assert.Equal(t, ctrl.Result{}, results[index], "a terminal restore must not be requeued")
	}
	assert.Equal(t, []recordedClientAction{statusUpdateOf(restore), deleteOf(newRestoreLease(restore))}, fixture.clientActions.snapshot(),
		"the malformed backup must be reported exactly once, the lease released and the ecosystem stay untouched")
	assertPreparedCondition(t, fixture.client, metav1.ConditionFalse, ReasonUnusableUnits)
	assertSuccessfulCondition(t, fixture.client, testRestore, metav1.ConditionFalse, ReasonUnusableUnits)
}

func TestARestorableSourceBackupLetsThePreparationRun(t *testing.T) {
	restore := withInitializedConditions(withMetadata(newParentRestore()))

	recorderMock := newMockEventRecorder(t)
	recorderMock.EXPECT().Eventf(matchesCloudoguRestoreNamed(testRestore), nil, corev1.EventTypeNormal, mock.Anything, mock.Anything, mock.Anything).Times(3)

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
		restore, readyStorageLocation(), readyProviderDeployment(), sourceBackup(velerov1.BackupPhaseCompleted))

	_, errs := fixture.reconcileTimes(testCtx, newRestoreRequest(testRestore), 1)

	require.NoError(t, errs[0])
	assertPreparedCondition(t, fixture.client, metav1.ConditionTrue, ReasonPreparationCompleted)
}

func TestAnAlreadyPreparedRestoreIsNotFailedByAMissingSourceBackup(t *testing.T) {
	restore := withPreparation(withInitializedConditions(withMetadata(newParentRestore())))

	recorderMock := newMockEventRecorder(t)
	recorderMock.EXPECT().Eventf(matchesCloudoguRestoreNamed(testRestore), nil, corev1.EventTypeNormal, mock.Anything, mock.Anything, mock.Anything).Maybe()

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

func TestCheckSourceBackup(t *testing.T) {
	backupInPhase := func(phase velerov1.BackupPhase) *state.ProviderBackup {
		return &state.ProviderBackup{Phase: phase}
	}
	// includes components / redmine if provided
	unitsWith := func(components, redmine *state.ProviderBackup) state.RestoreState {
		return state.RestoreState{
			Kind: state.KindUnits,
			Units: []state.UnitBackup{
				{Type: k8sv1.UnitTypeGlobalConfig, ProviderBackup: backupInPhase(velerov1.BackupPhaseCompleted)},
				{Type: k8sv1.UnitTypeComponents, ProviderBackup: components},
				{Type: k8sv1.UnitTypeDoguV2, ProviderBackup: backupInPhase(velerov1.BackupPhaseInProgress)},
				{Type: k8sv1.UnitTypeDogu, Name: "redmine", ProviderBackup: redmine},
			},
			V2Dogus:    []string{"cockpit"},
			Components: []string{"k8s-loki"},
		}
	}
	completed := backupInPhase(velerov1.BackupPhaseCompleted)
	ignoreV2 := k8sv1.RestoreSpec{BackupName: testBackup, Options: &k8sv1.UnitOptions{IgnoreV2Dogus: true}}
	legacyInPhase := func(phase velerov1.BackupPhase) state.RestoreState {
		return state.RestoreState{Kind: state.KindLegacy, Units: []state.UnitBackup{{Type: state.UnitTypeLegacy, ProviderBackup: backupInPhase(phase)}}}
	}

	tests := []struct {
		name        string
		spec        k8sv1.RestoreSpec
		state       state.RestoreState
		wantReason  string
		wantMessage string
	}{
		{
			name:        "not found",
			spec:        k8sv1.RestoreSpec{BackupName: testBackup},
			state:       state.RestoreState{Kind: state.KindNotFound},
			wantReason:  ReasonSourceBackupNotFound,
			wantMessage: "The velero backup 'name=test-backup' to be restored was not found.",
		},
		{
			name:        "invalid exclusions are listed and reported before unusable units",
			spec:        k8sv1.RestoreSpec{BackupName: testBackup, ExcludedDogus: []string{"cockpit"}, ExcludedComponents: []string{"k8s-foo"}},
			state:       unitsWith(nil, completed), // components unusable, but invalid exclusions go first
			wantReason:  ReasonInvalidExclusions,
			wantMessage: "The exclusions do not match the backup: excludedDogus: cockpit is a V2 dogu; excludedComponents: k8s-foo is not in the backup.",
		},
		{
			name:        "missing and unsuccessful units name their phase and how to skip them",
			spec:        k8sv1.RestoreSpec{BackupName: testBackup},
			state:       unitsWith(nil, backupInPhase(velerov1.BackupPhasePartiallyFailed)),
			wantReason:  ReasonUnusableUnits,
			wantMessage: "The backups of these units cannot be restored: components (missing, skip with excludedComponents); doguv2 (phase InProgress, skip with options.ignoreV2Dogus); dogu redmine (phase PartiallyFailed, skip with excludedDogus).",
		},
		{
			name:        "unusable global-config names its option",
			spec:        k8sv1.RestoreSpec{BackupName: testBackup, Options: &k8sv1.UnitOptions{IgnoreV2Dogus: true}},
			state:       state.RestoreState{Kind: state.KindUnits, Units: []state.UnitBackup{{Type: k8sv1.UnitTypeGlobalConfig, ProviderBackup: backupInPhase(velerov1.BackupPhaseFailed)}}},
			wantReason:  ReasonUnusableUnits,
			wantMessage: "The backups of these units cannot be restored: global-config (phase Failed, skip with options.ignoreGlobalConfig).",
		},
		{
			name:        "unit without phase",
			spec:        k8sv1.RestoreSpec{BackupName: testBackup},
			state:       state.RestoreState{Kind: state.KindUnits, Units: []state.UnitBackup{{Type: k8sv1.UnitTypeGlobalConfig, ProviderBackup: backupInPhase("")}}},
			wantReason:  ReasonUnusableUnits,
			wantMessage: "The backups of these units cannot be restored: global-config (phase unknown, skip with options.ignoreGlobalConfig).",
		},
		{
			name:  "skipped unusable units do not fail the restore",
			spec:  k8sv1.RestoreSpec{BackupName: testBackup, ExcludedDogus: []string{"redmine"}, ExcludedComponents: []string{"k8s-loki"}, Options: &k8sv1.UnitOptions{IgnoreV2Dogus: true}},
			state: unitsWith(nil, nil),
		},
		{
			name:        "nothing left to restore",
			spec:        k8sv1.RestoreSpec{BackupName: testBackup, ExcludedDogus: []string{"redmine"}, ExcludedComponents: []string{"k8s-loki"}, Options: &k8sv1.UnitOptions{IgnoreGlobalConfig: true, IgnoreV2Dogus: true}},
			state:       unitsWith(completed, completed),
			wantReason:  ReasonUnusableUnits,
			wantMessage: "No unit is left to restore after applying the exclusions and options.",
		},
		{name: "usable unit backup", spec: ignoreV2, state: unitsWith(completed, completed)},
		{
			name:        "unusable legacy backup",
			spec:        k8sv1.RestoreSpec{BackupName: testBackup},
			state:       legacyInPhase(velerov1.BackupPhaseFailed),
			wantReason:  ReasonUnusableUnits,
			wantMessage: "The backups of these units cannot be restored: legacy (phase Failed).",
		},
		{
			name:        "partially failed legacy backup",
			spec:        k8sv1.RestoreSpec{BackupName: testBackup},
			state:       legacyInPhase(velerov1.BackupPhasePartiallyFailed),
			wantReason:  ReasonUnusableUnits,
			wantMessage: "The backups of these units cannot be restored: legacy (phase PartiallyFailed).",
		},
		{name: "usable legacy backup", spec: k8sv1.RestoreSpec{BackupName: testBackup}, state: legacyInPhase(velerov1.BackupPhaseCompleted)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reason, message := checkSourceBackup(tt.spec, tt.state)

			assert.Equal(t, tt.wantReason, reason)
			assert.Equal(t, tt.wantMessage, message)
		})
	}
}
