package restore

import (
	"context"
	"testing"

	k8sv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/k8s-backup-operator/internal/provider/velero"
	"github.com/cloudogu/k8s-registry-lib/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func assertPreparedCondition(t *testing.T, testClient client.Client, status metav1.ConditionStatus, reason string) {
	t.Helper()

	stored := &k8sv1.Restore{}
	require.NoError(t, testClient.Get(testCtx, client.ObjectKey{Namespace: testNamespace, Name: testRestore}, stored))

	condition := meta.FindStatusCondition(stored.Status.Conditions, k8sv1.ConditionPrepared)
	require.NotNil(t, condition, "no Prepared condition was persisted")
	assert.Equal(t, status, condition.Status)
	assert.Equal(t, reason, condition.Reason)
}

func TestPreparationScalesDownCleansUpAndPersistsItsMilestoneWithoutStartingTheRestore(t *testing.T) {
	restore := withInitializedConditions(withMetadata(newParentRestore()))
	var stageOrder []string
	recorderMock := newMockEventRecorder(t)
	recorderMock.EXPECT().Event(matchesRestoreNamed(testRestore), corev1.EventTypeNormal, ReasonPreparing, "Preparation in progress - scale down and cleanup").Once()
	recorderMock.EXPECT().Event(matchesRestoreNamed(testRestore), corev1.EventTypeNormal, ReasonCleanupCompleted, "Cleanup before restore completed").Once()
	recorderMock.EXPECT().Event(matchesRestoreNamed(testRestore), corev1.EventTypeNormal, ReasonPreparationCompleted, "Preparation completed").Once()

	providerMock := newMockRestoreProvider(t)
	providerMock.EXPECT().CheckReady(testCtx).Run(func(context.Context) {
		stageOrder = append(stageOrder, "provider-ready")
	}).Return(nil).Once()
	installProvider(t, providerMock)
	scaleMock := newMockScaleManager(t)
	scaleMock.EXPECT().ScaleDown(testCtx).Run(func(context.Context) {
		stageOrder = append(stageOrder, "scale-down")
	}).Return(nil).Once()
	cleanupMock := newMockCleanupManager(t)
	cleanupMock.EXPECT().Cleanup(testCtx).Run(func(context.Context) {
		stageOrder = append(stageOrder, "cleanup")
	}).Return(nil).Once()
	maintenanceMock := newMockMaintenanceModeSwitch(t)
	maintenanceMock.EXPECT().GetStatus(testCtx).Run(func(context.Context) {
		stageOrder = append(stageOrder, "maintenance")
	}).Return(repository.MaintenanceModeDescription{}, true, nil).Once()

	factory := func(fakeClient client.WithWatch) reconcileFunction {
		reconciler := NewRestoreReconciler(fakeClient, recorderMock, testNamespace, cleanupMock, scaleMock, requeueAfterTest)
		reconciler.maintenanceModeSwitch = maintenanceMock

		return reconciler.Reconcile
	}
	fixture := newMultiReconcileFixture(t, interceptor.Funcs{}, factory, restore)

	results, errs := fixture.reconcileTimes(testCtx, newRestoreRequest(testRestore), 1)

	require.NoError(t, errs[0])
	assert.Equal(t, ctrl.Result{RequeueAfter: defaultRequeueDelay}, results[0], "the preparation must end the reconciliation")
	assert.Equal(t, []recordedClientAction{statusUpdateOf(restore)}, fixture.clientActions.snapshot(),
		"the preparation must persist its milestone in one status write")
	assert.Equal(t, []string{"provider-ready", "maintenance", "scale-down", "cleanup"}, stageOrder)
	assertPreparedCondition(t, fixture.client, metav1.ConditionTrue, ReasonPreparationCompleted)
}

func TestAPreparedRestoreSkipsThePreparationAndStartsTheProviderRestore(t *testing.T) {
	restore := withPreparation(withInitializedConditions(withMetadata(newParentRestore())))

	recorderMock := newMockEventRecorder(t)
	recorderMock.EXPECT().Event(matchesRestoreNamed(testRestore), corev1.EventTypeNormal, ReasonProviderRestoreRunning, "Start provider restore process").Once()

	maintenanceMock := newMockMaintenanceModeSwitch(t)
	maintenanceMock.EXPECT().GetStatus(testCtx).Return(repository.MaintenanceModeDescription{}, true, nil)

	// The cleanup, scale and maintenance mocks carry no expectations, so any preparation step would fail.
	factory := func(fakeClient client.WithWatch) reconcileFunction {
		reconciler := NewRestoreReconciler(fakeClient, recorderMock, testNamespace, newMockCleanupManager(t), newMockScaleManager(t), requeueAfterTest)
		reconciler.maintenanceModeSwitch = maintenanceMock

		return reconciler.Reconcile
	}
	fixture := newMultiReconcileFixture(t, interceptor.Funcs{}, factory, restore)

	results, errs := fixture.reconcileTimes(testCtx, newRestoreRequest(testRestore), 1)

	require.NoError(t, errs[0])
	assert.Equal(t, ctrl.Result{RequeueAfter: defaultRequeueDelay}, results[0])
	assert.Equal(t, []recordedClientAction{createOf(velero.BuildRestore(restore))}, fixture.clientActions.snapshot(),
		"a prepared restore must not be written again, it must start the provider restore")
}

// A restore whose status was lost reads as unprepared although its owned child proves it prepared.
func TestPreparationIsSkippedForAnUnpreparedRestoreThatAlreadyHasAProviderChild(t *testing.T) {
	restore := withInitializedConditions(withMetadata(newParentRestore()))
	require.False(t, meta.IsStatusConditionTrue(restore.Status.Conditions, k8sv1.ConditionPrepared),
		"the restore under test must not carry the Prepared milestone")

	writes := &clientWrites{}
	recorderMock := newMockEventRecorder(t)
	reconciler := &restoreReconciler{
		namespace: testNamespace,
		recorder:  recorderMock,
		// with owned child in the client
		k8sClient:             newTestClient(t, writes.interceptor(), restore, ownedRunningChild(restore)),
		cleanup:               newMockCleanupManager(t),
		scaleManager:          newMockScaleManager(t),
		maintenanceModeSwitch: newMockMaintenanceModeSwitch(t),
	}

	updated, outcome := reconciler.ensurePreparation(testCtx, restore)

	assert.Equal(t, next(), outcome, "the workflow must continue without preparing again")
	assert.Equal(t, restore, updated)
	assert.Equal(t, 0, writes.total(), "an inferred preparation must not be written")
}

func TestPreparationContinuesWhenTheMaintenanceModeCannotBeActivated(t *testing.T) {
	restore := withInitializedConditions(withMetadata(newParentRestore()))
	recorderMock := newMockEventRecorder(t)
	recorderMock.EXPECT().Event(matchesRestoreNamed(testRestore), corev1.EventTypeNormal, ReasonPreparing, "Preparation in progress - scale down and cleanup").Once()
	recorderMock.EXPECT().Event(matchesRestoreNamed(testRestore), corev1.EventTypeNormal, ReasonCleanupCompleted, "Cleanup before restore completed").Once()
	recorderMock.EXPECT().Eventf(matchesRestoreNamed(testRestore), corev1.EventTypeNormal, ReasonMaintenanceModeActivated, "Could not activate maintenance mode; continuing restore.").Once()
	recorderMock.EXPECT().Event(matchesRestoreNamed(testRestore), corev1.EventTypeNormal, ReasonPreparationCompleted, "Preparation completed").Once()

	expectReadinessCheck(t, nil)
	maintenanceMock := newMockMaintenanceModeSwitch(t)
	maintenanceMock.EXPECT().GetStatus(testCtx).Return(repository.MaintenanceModeDescription{}, false, nil).Once()
	maintenanceMock.EXPECT().Activate(testCtx, repository.MaintenanceModeDescription{
		Title: maintenanceModeTitle,
		Text:  maintenanceModeText,
	}, false).Return(assert.AnError).Once()
	scaleMock := newMockScaleManager(t)
	scaleMock.EXPECT().ScaleDown(testCtx).Return(nil).Once()
	cleanupMock := newMockCleanupManager(t)
	cleanupMock.EXPECT().Cleanup(testCtx).Return(nil).Once()

	factory := func(fakeClient client.WithWatch) reconcileFunction {
		reconciler := NewRestoreReconciler(fakeClient, recorderMock, testNamespace, cleanupMock, scaleMock, requeueAfterTest)
		reconciler.maintenanceModeSwitch = maintenanceMock
		return reconciler.Reconcile
	}
	fixture := newMultiReconcileFixture(t, interceptor.Funcs{}, factory, restore)

	results, errs := fixture.reconcileTimes(testCtx, newRestoreRequest(testRestore), 1)

	require.NoError(t, errs[0])
	// expected requeue, because of condition write
	assert.Equal(t, ctrl.Result{RequeueAfter: defaultRequeueDelay}, results[0])
	assertPreparedCondition(t, fixture.client, metav1.ConditionTrue, ReasonPreparationCompleted)
}

func TestAFailedScaleDownReportsPreparedFalseAndRetriesWithoutCleaningUp(t *testing.T) {
	restore := withInitializedConditions(withMetadata(newParentRestore()))
	recorderMock := newMockEventRecorder(t)
	recorderMock.EXPECT().Event(matchesRestoreNamed(testRestore), corev1.EventTypeNormal, ReasonPreparing, "Preparation in progress - scale down and cleanup").Once()
	recorderMock.EXPECT().Event(matchesRestoreNamed(testRestore), corev1.EventTypeWarning, ReasonPreparationFailed, "The preparation of the ecosystem failed -> retrying").Once()

	expectReadinessCheck(t, nil)
	maintenanceMock := newMockMaintenanceModeSwitch(t)
	maintenanceMock.EXPECT().GetStatus(testCtx).Return(repository.MaintenanceModeDescription{}, true, nil)
	scaleMock := newMockScaleManager(t)
	scaleMock.EXPECT().ScaleDown(testCtx).Return(assert.AnError).Once()

	factory := func(fakeClient client.WithWatch) reconcileFunction {
		reconciler := NewRestoreReconciler(fakeClient, recorderMock, testNamespace, newMockCleanupManager(t), scaleMock, requeueAfterTest)
		reconciler.maintenanceModeSwitch = maintenanceMock
		return reconciler.Reconcile
	}
	fixture := newMultiReconcileFixture(t, interceptor.Funcs{}, factory, restore)

	results, errs := fixture.reconcileTimes(testCtx, newRestoreRequest(testRestore), 1)

	require.ErrorIs(t, errs[0], assert.AnError)
	assert.ErrorContains(t, errs[0], "failed to scale down workloads before restore")
	assert.Equal(t, ctrl.Result{}, results[0], "an error carries the retry, so there must be no explicit requeue")
	assertPreparedCondition(t, fixture.client, metav1.ConditionFalse, ReasonPreparationFailed)

	stored := &k8sv1.Restore{}
	require.NoError(t, fixture.client.Get(testCtx, newRestoreRequest(testRestore).NamespacedName, stored))
	assert.False(t, isTerminal(stored), "a failed preparation must leave the restore resumable")
}

func TestAFailedCleanupReportsPreparedFalseAndRetries(t *testing.T) {
	restore := withInitializedConditions(withMetadata(newParentRestore()))
	recorderMock := newMockEventRecorder(t)
	recorderMock.EXPECT().Event(matchesRestoreNamed(testRestore), corev1.EventTypeNormal, ReasonPreparing, "Preparation in progress - scale down and cleanup").Once()
	recorderMock.EXPECT().Event(matchesRestoreNamed(testRestore), corev1.EventTypeWarning, ReasonPreparationFailed, "The preparation of the ecosystem failed -> retrying").Once()
	recorderMock.EXPECT().Event(matchesRestoreNamed(testRestore), corev1.EventTypeWarning, ReasonCleanupFailed, "Cleanup before restore failed").Once()

	expectReadinessCheck(t, nil)
	maintenanceMock := newMockMaintenanceModeSwitch(t)
	maintenanceMock.EXPECT().GetStatus(testCtx).Return(repository.MaintenanceModeDescription{}, true, nil)
	scaleMock := newMockScaleManager(t)
	scaleMock.EXPECT().ScaleDown(testCtx).Return(nil).Once()
	cleanupMock := newMockCleanupManager(t)
	cleanupMock.EXPECT().Cleanup(testCtx).Return(assert.AnError).Once()

	factory := func(fakeClient client.WithWatch) reconcileFunction {
		reconciler := NewRestoreReconciler(fakeClient, recorderMock, testNamespace, cleanupMock, scaleMock, requeueAfterTest)
		reconciler.maintenanceModeSwitch = maintenanceMock
		return reconciler.Reconcile
	}
	fixture := newMultiReconcileFixture(t, interceptor.Funcs{}, factory, restore)

	results, errs := fixture.reconcileTimes(testCtx, newRestoreRequest(testRestore), 1)

	require.ErrorIs(t, errs[0], assert.AnError)
	assert.ErrorContains(t, errs[0], "failed to cleanup before restore")
	assert.Equal(t, ctrl.Result{}, results[0])
	assertPreparedCondition(t, fixture.client, metav1.ConditionFalse, ReasonPreparationFailed)
}

func TestAnUnpersistablePreparationMilestoneIsRetriedWithoutStartingTheRestore(t *testing.T) {
	restore := withInitializedConditions(withMetadata(newParentRestore()))
	recorderMock := newMockEventRecorder(t)
	recorderMock.EXPECT().Event(matchesRestoreNamed(testRestore), corev1.EventTypeNormal, ReasonPreparing, "Preparation in progress - scale down and cleanup").Once()
	recorderMock.EXPECT().Event(matchesRestoreNamed(testRestore), corev1.EventTypeNormal, ReasonCleanupCompleted, "Cleanup before restore completed").Once()

	expectReadinessCheck(t, nil)
	maintenanceMock := newMockMaintenanceModeSwitch(t)
	maintenanceMock.EXPECT().GetStatus(testCtx).Return(repository.MaintenanceModeDescription{}, true, nil)
	scaleMock := newMockScaleManager(t)
	scaleMock.EXPECT().ScaleDown(testCtx).Return(nil).Once()
	cleanupMock := newMockCleanupManager(t)
	cleanupMock.EXPECT().Cleanup(testCtx).Return(nil).Once()

	factory := func(fakeClient client.WithWatch) reconcileFunction {
		reconciler := NewRestoreReconciler(fakeClient, recorderMock, testNamespace, cleanupMock, scaleMock, requeueAfterTest)
		reconciler.maintenanceModeSwitch = maintenanceMock
		return reconciler.Reconcile
	}
	fixture := newMultiReconcileFixture(t, failingStatusUpdate(assert.AnError), factory, restore)

	results, errs := fixture.reconcileTimes(testCtx, newRestoreRequest(testRestore), 1)

	require.ErrorIs(t, errs[0], assert.AnError)
	assert.ErrorContains(t, errs[0], "failed to persist the preparation of restore test-restore")
	assert.Equal(t, ctrl.Result{}, results[0])
}

func TestAnUnreadyProviderPreventsMaintenanceAndPreparationWithoutTouchingTheEcosystem(t *testing.T) {
	restore := withInitializedConditions(withMetadata(newParentRestore()))

	expectReadinessCheck(t, assert.AnError)
	recorderMock := newMockEventRecorder(t)

	factory := func(fakeClient client.WithWatch) reconcileFunction {
		reconciler := NewRestoreReconciler(fakeClient, recorderMock, testNamespace, newMockCleanupManager(t), newMockScaleManager(t), requeueAfterTest)
		reconciler.maintenanceModeSwitch = newMockMaintenanceModeSwitch(t)

		return reconciler.Reconcile
	}
	fixture := newMultiReconcileFixture(t, interceptor.Funcs{}, factory, restore)

	results, errs := fixture.reconcileTimes(testCtx, newRestoreRequest(testRestore), 1)

	require.ErrorIs(t, errs[0], assert.AnError)
	assert.ErrorContains(t, errs[0], "failed to get restore provider [velero]: provider velero is not ready")
	assert.Equal(t, ctrl.Result{}, results[0], "an error carries the retry, so there must be no explicit requeue")
	assert.Empty(t, fixture.clientActions.snapshot(), "an unready provider must leave the restore untouched")
}
