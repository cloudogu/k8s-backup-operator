package restore

import (
	"context"
	"testing"

	k8sv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/k8s-backup-operator/internal/controller/restore/state"
	"github.com/cloudogu/k8s-backup-operator/internal/provider/velero"
	"github.com/cloudogu/k8s-registry-lib/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

// ownedChild returns the provider restore the given Restore creates for the given unit.
func ownedChild(parent *k8sv1.Restore, unit k8sv1.UnitIdentifier) *velerov1.Restore {
	return velero.BuildRestore(parent, providerRestoreRequest(parent, unit))
}

// ownedRunningChild returns the provider restore of the given unit of the given Restore, executing.
func ownedRunningChild(parent *k8sv1.Restore, unit k8sv1.UnitIdentifier) *velerov1.Restore {
	child := ownedChild(parent, unit)
	child.Status.Phase = velerov1.RestorePhaseInProgress

	return child
}

// foreignChild returns a provider restore occupying the child name of the given unit of the given Restore, without
// an owner reference, so it must never be adopted.
func foreignChild(parent *k8sv1.Restore, unit k8sv1.UnitIdentifier) *velerov1.Restore {
	child := ownedRunningChild(parent, unit)
	child.OwnerReferences = nil
	child.Labels = nil

	return child
}

var (
	testLegacyUnit       = k8sv1.UnitIdentifier{Type: state.UnitTypeLegacy}
	testGlobalConfigUnit = k8sv1.UnitIdentifier{Type: k8sv1.UnitTypeGlobalConfig}
	testLdapUnit         = k8sv1.UnitIdentifier{Type: k8sv1.UnitTypeDogu, Name: "ldap"}
)

// completedUnitBackup is the provider backup of the given unit. The associated cloudogu-backup contains global-config
// and ldap units.
func completedUnitBackup(unit k8sv1.UnitIdentifier) *velerov1.Backup {
	return &velerov1.Backup{
		Name: velero.UnitObjectName(testBackup, unit), Namespace: testNamespace,
		Labels: map[string]string{
			k8sv1.BackupLabelKey:     k8sv1.GetTruncatedName(testBackup),
			k8sv1.BackupTypeLabelKey: string(unit.Type),
			k8sv1.DoguNameLabelKey:   unit.Name,
		},
		Annotations: map[string]string{
			k8sv1.UnitsAnnotationKey: `[{"type":"global-config"},{"type":"dogu","name":"ldap"}]`,
		},
		Status: velerov1.BackupStatus{Phase: velerov1.BackupPhaseCompleted},
	}
}

func TestNameConflictsFailsTheRestoreBeforePreparation(t *testing.T) {
	restore := withInitializedConditions(withMetadata(newParentRestore()))
	conflicting := foreignChild(restore, testLdapUnit)

	recorderMock := newMockEventRecorder(t)
	recorderMock.EXPECT().Eventf(matchesCloudoguRestoreNamed(testRestore), matchesVeleroRestoreNamed(conflicting.Name), corev1.EventTypeWarning, k8sv1.ErrorOnCreateEventReason, actionCreateProviderRestore,
		"velero restore [test-restore-dogu-ldap] conflicts with the expected child: it is not controlled by restore [test-restore] and must not be adopted").Return()
	reconciler := &restoreReconciler{
		recorder: recorderMock,
		k8sClient: newTestClientWithParent(t, interceptor.Funcs{}, restore,
			completedUnitBackup(testGlobalConfigUnit), completedUnitBackup(testLdapUnit), conflicting),
	}

	updated, outcome := reconciler.ensureProviderChildState(testCtx, restore)

	assert.Equal(t, abort(), outcome, "a conflict is terminal")
	successfulCondition := findSuccessfulCondition(updated)
	require.NotNil(t, successfulCondition)
	assert.Equal(t, metav1.ConditionFalse, successfulCondition.Status)
	assert.Equal(t, ReasonProviderRestoreConflict, successfulCondition.Reason)
	providerCondition := meta.FindStatusCondition(updated.Status.Conditions, k8sv1.ConditionProviderSucceeded)
	require.NotNil(t, providerCondition)
	assert.Equal(t, metav1.ConditionFalse, providerCondition.Status)
	assert.Equal(t, ReasonProviderRestoreConflict, providerCondition.Reason)
}

func TestNameConflictsAreNotCheckedOnceThePreparationHappened(t *testing.T) {
	restore := withInitializedConditions(withMetadata(newParentRestore()))
	reconciler := &restoreReconciler{
		k8sClient: newTestClientWithParent(t, interceptor.Funcs{}, restore,
			completedUnitBackup(testGlobalConfigUnit), completedUnitBackup(testLdapUnit),
			ownedRunningChild(restore, testGlobalConfigUnit), foreignChild(restore, testLdapUnit)),
	}

	_, outcome := reconciler.ensureProviderChildState(testCtx, restore)

	assert.Equal(t, next(), outcome, "the check only protects the preparation")
}

func TestForeignRestoreOfAnExcludedUnitChildIsNoConflict(t *testing.T) {
	restore := withInitializedConditions(withMetadata(newParentRestore()))
	restore.Spec.ExcludedDogus = []string{testLdapUnit.Name}

	reconciler := &restoreReconciler{
		k8sClient: newTestClientWithParent(t, interceptor.Funcs{}, restore,
			completedUnitBackup(testGlobalConfigUnit), completedUnitBackup(testLdapUnit), foreignChild(restore, testLdapUnit)),
	}

	_, outcome := reconciler.ensureProviderChildState(testCtx, restore)

	assert.Equal(t, next(), outcome, "only the children of the units to restore can conflict")
}

func TestBackupIsMissingCannotConflict(t *testing.T) {
	restore := withInitializedConditions(withMetadata(newParentRestore()))
	reconciler := &restoreReconciler{k8sClient: newTestClientWithParent(t, interceptor.Funcs{}, restore, foreignChild(restore, testLegacyUnit))}

	_, outcome := reconciler.ensureProviderChildState(testCtx, restore)

	assert.Equal(t, next(), outcome, "the source backup check reports the missing backup")
}

func TestBackupListErrorIsRetried(t *testing.T) {
	restore := withInitializedConditions(withMetadata(newParentRestore()))
	failingBackupList := interceptor.Funcs{
		List: func(ctx context.Context, wrapped client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
			if _, isBackupList := list.(*velerov1.BackupList); isBackupList {
				return assert.AnError
			}

			return wrapped.List(ctx, list, opts...)
		},
	}
	reconciler := &restoreReconciler{k8sClient: newTestClientWithParent(t, failingBackupList, restore)}

	_, outcome := reconciler.ensureProviderChildState(testCtx, restore)

	require.ErrorIs(t, outcome.err, assert.AnError)
	assert.ErrorContains(t, outcome.err, "failed to collect the backup of restore test-restore")
}

// An existing owned child must send the workflow past preparation even without a set condition
func TestAnExistingOwnedProviderChildSkipsThePreparationAndIsObservedInstead(t *testing.T) {
	restore := withInitializedConditions(withMetadata(newParentRestore()))
	recorderMock := newMockEventRecorder(t)

	maintenanceMock := newMockMaintenanceModeSwitch(t)
	maintenanceMock.EXPECT().GetStatus(testCtx).Return(repository.MaintenanceModeDescription{}, true, nil)

	factory := func(fakeClient client.WithWatch) reconcileFunction {
		reconciler := NewRestoreReconciler(fakeClient, recorderMock, testNamespace, nil, nil, requeueAfterTest, testBackupStorage, testProviderDeployment)
		reconciler.maintenanceModeSwitch = maintenanceMock
		return reconciler.Reconcile
	}
	// with owned child
	fixture := newMultiReconcileFixture(t, interceptor.Funcs{}, factory, restore, ownedRunningChild(restore, testLegacyUnit))

	results, errs := fixture.reconcileTimes(testCtx, newRestoreRequest(testRestore), 2)

	for index := range results {
		require.NoError(t, errs[index])
		assert.Equal(t, ctrl.Result{RequeueAfter: providerObservationRecoveryDelay}, results[index],
			"an undecided provider restore must not be waited for")
	}
	assert.Equal(t, []recordedClientAction{statusUpdateOf(restore)}, fixture.clientActions.snapshot(),
		"the observed state must be written once and the child must never be written")
	assertPersistedCondition(t, fixture.client, k8sv1.ConditionProviderSucceeded, metav1.ConditionUnknown, ReasonProviderRestoreRunning)
}

func TestAConflictingProviderChildFailsTheRestoreBeforePreparation(t *testing.T) {
	restore := withInitializedConditions(withMetadata(newParentRestore()))
	conflicting := foreignChild(restore, testLegacyUnit)

	recorderMock := newMockEventRecorder(t)
	recorderMock.EXPECT().Eventf(matchesCloudoguRestoreNamed(testRestore), matchesVeleroRestoreNamed(testRestore), corev1.EventTypeWarning, k8sv1.ErrorOnCreateEventReason, actionCreateProviderRestore,
		"velero restore [test-restore] conflicts with the expected child: it is not controlled by restore [test-restore] and must not be adopted").Return()

	factory := func(fakeClient client.WithWatch) reconcileFunction {
		return NewRestoreReconciler(fakeClient, recorderMock, testNamespace, nil, nil, requeueAfterTest, testBackupStorage, testProviderDeployment).Reconcile
	}
	fixture := newMultiReconcileFixture(t, interceptor.Funcs{}, factory, restore, conflicting, usableSourceBackup(),
		readyStorageLocation(), readyProviderDeployment())
	request := newRestoreRequest(testRestore)

	results, errs := fixture.reconcileTimes(testCtx, request, 2)

	for index := range results {
		require.NoError(t, errs[index])
		assert.Equal(t, ctrl.Result{}, results[index], "a conflict is terminal, so it must not be requeued")
	}
	assert.Equal(t, []recordedClientAction{statusUpdateOf(restore), deleteOf(newRestoreLease(restore))}, fixture.clientActions.snapshot(),
		"the conflict must be reported exactly once and the child must not be touched")

	stored := &k8sv1.Restore{}
	require.NoError(t, fixture.client.Get(testCtx, request.NamespacedName, stored))
	assert.Equal(t, k8sv1.RestoreStatusFailed, stored.Status.Status)
	assert.Equal(t, metav1.ConditionFalse, findSuccessfulCondition(stored).Status)
	assert.Equal(t, ReasonProviderRestoreConflict, findSuccessfulCondition(stored).Reason)
	providerCondition := meta.FindStatusCondition(stored.Status.Conditions, k8sv1.ConditionProviderSucceeded)
	require.NotNil(t, providerCondition)
	assert.Equal(t, metav1.ConditionFalse, providerCondition.Status)
	assert.Equal(t, ReasonProviderRestoreConflict, providerCondition.Reason)
}

func TestAnUnreportableProviderChildConflictIsRetried(t *testing.T) {
	restore := withInitializedConditions(withMetadata(newParentRestore()))

	recorderMock := newMockEventRecorder(t)
	recorderMock.EXPECT().Eventf(matchesCloudoguRestoreNamed(testRestore), matchesVeleroRestoreNamed(testRestore), corev1.EventTypeWarning, k8sv1.ErrorOnCreateEventReason, actionCreateProviderRestore, mock.Anything).Return()

	factory := func(fakeClient client.WithWatch) reconcileFunction {
		return NewRestoreReconciler(fakeClient, recorderMock, testNamespace, nil, nil, requeueAfterTest, testBackupStorage, testProviderDeployment).Reconcile
	}
	fixture := newMultiReconcileFixture(t, failingStatusUpdate(assert.AnError), factory, restore, foreignChild(restore, testLegacyUnit), usableSourceBackup(),
		readyStorageLocation(), readyProviderDeployment())

	_, errs := fixture.reconcileTimes(testCtx, newRestoreRequest(testRestore), 1)

	require.ErrorIs(t, errs[0], assert.AnError)
	assert.ErrorContains(t, errs[0], "failed to report the provider restore conflict of restore test-restore")
}

func TestAnUnreadableProviderChildIsRetried(t *testing.T) {
	restore := withInitializedConditions(withMetadata(newParentRestore()))

	failingChildGet := interceptor.Funcs{
		Get: func(ctx context.Context, wrapped client.WithWatch, key client.ObjectKey, object client.Object, opts ...client.GetOption) error {
			if _, isChild := object.(*velerov1.Restore); isChild {
				return assert.AnError
			}

			return wrapped.Get(ctx, key, object, opts...)
		},
	}

	factory := func(fakeClient client.WithWatch) reconcileFunction {
		return NewRestoreReconciler(fakeClient, nil, testNamespace, nil, nil, requeueAfterTest, testBackupStorage, testProviderDeployment).Reconcile
	}
	fixture := newMultiReconcileFixture(t, failingChildGet, factory, restore, usableSourceBackup(),
		readyStorageLocation(), readyProviderDeployment())

	_, errs := fixture.reconcileTimes(testCtx, newRestoreRequest(testRestore), 1)

	require.ErrorIs(t, errs[0], assert.AnError)
	assert.ErrorContains(t, errs[0], "failed to read the provider restore of restore test-restore")
}
