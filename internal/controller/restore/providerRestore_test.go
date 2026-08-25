package restore

import (
	"context"
	"testing"

	k8sv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/k8s-backup-operator/internal/provider/velero"
	"github.com/cloudogu/k8s-registry-lib/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

// startableRestore is a Restore that reached the stage which starts the provider restore.
func startableRestore() *k8sv1.Restore {
	return withPreparation(withInitializedConditions(withMetadata(newParentRestore())))
}

func TestProviderRestoreStageCreatesOwnedChildWithoutWaiting(t *testing.T) {
	restore := startableRestore()

	recorderMock := newMockEventRecorder(t)
	recorderMock.EXPECT().Event(matchesRestoreNamed(testRestore), corev1.EventTypeNormal, ReasonPreparationCompleted, "Preparation completed").Once()
	recorderMock.EXPECT().Event(matchesRestoreNamed(testRestore), corev1.EventTypeNormal, ReasonProviderRestoreRunning, "Start provider restore process").Once()

	maintenanceMock := newMockMaintenanceModeSwitch(t)
	maintenanceMock.EXPECT().GetStatus(testCtx).Return(repository.MaintenanceModeDescription{}, true, nil)

	factory := func(fakeClient client.WithWatch) reconcileFunction {
		reconciler := NewRestoreReconciler(fakeClient, recorderMock, testNamespace, nil, nil, requeueAfterTest)
		reconciler.maintenanceModeSwitch = maintenanceMock
		// no manager - reaching a further stage would panic
		return reconciler.Reconcile
	}
	fixture := newMultiReconcileFixture(t, interceptor.Funcs{}, factory, restore)

	results, errs := fixture.reconcileTimes(testCtx, newRestoreRequest(testRestore), 1)

	require.NoError(t, errs[0])
	assert.Equal(t, ctrl.Result{RequeueAfter: defaultRequeueDelay}, results[0])
	assert.Equal(t, []recordedClientAction{createOf(velero.BuildRestore(restore))}, fixture.clientActions.snapshot(),
		"starting the restore must create the child and write nothing else")

	child := &velerov1.Restore{}
	require.NoError(t, fixture.client.Get(testCtx, client.ObjectKeyFromObject(restore), child))
	assert.True(t, velero.IsOwnedRestore(restore, child), "the created child must be owned by the restore")
	assert.Equal(t, testBackup, child.Spec.BackupName)

	assertPersistedCondition(t, fixture.client, k8sv1.ConditionProviderSucceeded, metav1.ConditionUnknown, ReasonPending)
}

func TestARepeatedReconciliationNeverStartsASecondProviderRestore(t *testing.T) {
	restore := startableRestore()

	recorderMock := newMockEventRecorder(t)
	recorderMock.EXPECT().Event(matchesRestoreNamed(testRestore), corev1.EventTypeNormal, ReasonPreparationCompleted, "Preparation completed").Times(3)
	recorderMock.EXPECT().Event(matchesRestoreNamed(testRestore), corev1.EventTypeNormal, ReasonProviderRestoreRunning, "Start provider restore process").Once()

	maintenanceMock := newMockMaintenanceModeSwitch(t)
	maintenanceMock.EXPECT().GetStatus(testCtx).Return(repository.MaintenanceModeDescription{}, true, nil)

	factory := func(fakeClient client.WithWatch) reconcileFunction {
		reconciler := NewRestoreReconciler(fakeClient, recorderMock, testNamespace, nil, nil, requeueAfterTest)
		reconciler.maintenanceModeSwitch = maintenanceMock
		return reconciler.Reconcile
	}
	fixture := newMultiReconcileFixture(t, interceptor.Funcs{}, factory, restore)
	request := newRestoreRequest(testRestore)

	firstResults, firstErrs := fixture.reconcileTimes(testCtx, request, 1)
	fixture.restart(factory)
	laterResults, laterErrs := fixture.reconcileTimes(testCtx, request, 2)

	require.NoError(t, firstErrs[0])
	require.NoError(t, laterErrs[0])
	require.NoError(t, laterErrs[1])
	assert.Equal(t, ctrl.Result{RequeueAfter: defaultRequeueDelay}, firstResults[0])
	for index := range laterResults {
		assert.Equal(t, ctrl.Result{RequeueAfter: providerObservationRecoveryDelay}, laterResults[index],
			"the restarted operator must observe the adopted child instead of starting a new restore")
	}

	// The second reconciliation observes the pending child once; the third one changes nothing.
	assert.Equal(t, []recordedClientAction{createOf(velero.BuildRestore(restore)), statusUpdateOf(restore)},
		fixture.clientActions.snapshot(), "the child must be created exactly once")
	assertPersistedCondition(t, fixture.client, k8sv1.ConditionProviderSucceeded, metav1.ConditionUnknown, ReasonProviderRestorePending)
}

func TestAProviderRestoreThatCannotBeStartedIsReportedAndRetried(t *testing.T) {
	restore := startableRestore()

	recorderMock := newMockEventRecorder(t)
	recorderMock.EXPECT().Event(matchesRestoreNamed(testRestore), corev1.EventTypeNormal, ReasonPreparationCompleted, "Preparation completed").Once()
	recorderMock.EXPECT().Event(matchesRestoreNamed(testRestore), corev1.EventTypeNormal, ReasonProviderRestoreRunning, "Start provider restore process").Once()
	recorderMock.EXPECT().Event(matchesRestoreNamed(testRestore), corev1.EventTypeWarning, ReasonProviderRestoreFailed, mock.Anything).Once()

	maintenanceMock := newMockMaintenanceModeSwitch(t)
	maintenanceMock.EXPECT().GetStatus(testCtx).Return(repository.MaintenanceModeDescription{}, true, nil)

	factory := func(fakeClient client.WithWatch) reconcileFunction {
		reconciler := NewRestoreReconciler(fakeClient, recorderMock, testNamespace, nil, nil, requeueAfterTest)
		reconciler.maintenanceModeSwitch = maintenanceMock
		return reconciler.Reconcile
	}
	fixture := newMultiReconcileFixture(t, failingCreate(assert.AnError), factory, restore)

	results, errs := fixture.reconcileTimes(testCtx, newRestoreRequest(testRestore), 1)

	require.ErrorIs(t, errs[0], assert.AnError)
	assert.ErrorContains(t, errs[0], "failed to start the provider restore of restore test-restore")
	assert.Equal(t, ctrl.Result{}, results[0], "an error carries the retry, so there must be no explicit requeue")

	stored := &k8sv1.Restore{}
	require.NoError(t, fixture.client.Get(testCtx, newRestoreRequest(testRestore).NamespacedName, stored))
	assert.False(t, isTerminal(stored), "a child that could not be created must leave the restore resumable")
}

func TestAConflictingChildFoundWhenStartingTheRestoreIsTerminal(t *testing.T) {
	restore := startableRestore()

	recorderMock := newMockEventRecorder(t)
	recorderMock.EXPECT().Event(matchesRestoreNamed(testRestore), corev1.EventTypeNormal, ReasonProviderRestoreRunning, "Start provider restore process").Once()
	recorderMock.EXPECT().Event(matchesRestoreNamed(testRestore), corev1.EventTypeWarning, k8sv1.ErrorOnCreateEventReason, mock.Anything).Return()

	reconciler := &restoreReconciler{
		namespace: testNamespace,
		recorder:  recorderMock,
		// the foreign child appears after this stage read the child state
		k8sClient: newTestClient(t, appearingForeignChild(t, restore), restore),
	}

	updated, outcome := reconciler.ensureProviderRestore(testCtx, restore)

	assert.Equal(t, abort(), outcome, "a conflict must not be retried")
	require.NotNil(t, updated)
	condition := findSuccessfulCondition(updated)
	require.NotNil(t, condition)
	assert.Equal(t, metav1.ConditionFalse, condition.Status)
	assert.Equal(t, ReasonProviderRestoreConflict, condition.Reason)
	assertPersistedCondition(t, reconciler.k8sClient, k8sv1.ConditionProviderSucceeded, metav1.ConditionFalse, ReasonProviderRestoreConflict)
}

// appearingForeignChild hides the foreign child from the first read of the stage and reveals it to the
// second one, which is the race the stage's conflict branch exists for.
func appearingForeignChild(t *testing.T, parent *k8sv1.Restore) interceptor.Funcs {
	t.Helper()

	childReads := 0

	return interceptor.Funcs{
		Create: func(_ context.Context, _ client.WithWatch, object client.Object, _ ...client.CreateOption) error {
			t.Errorf("the stage must not create %T while a foreign child occupies the expected name", object)

			return nil
		},
		Get: func(ctx context.Context, wrapped client.WithWatch, key client.ObjectKey, object client.Object, opts ...client.GetOption) error {
			child, isChild := object.(*velerov1.Restore)
			if !isChild {
				return wrapped.Get(ctx, key, object, opts...)
			}

			if childReads == 0 {
				childReads++
				return apierrors.NewNotFound(schema.GroupResource{Group: velerov1.SchemeGroupVersion.Group, Resource: "restores"}, key.Name)
			}

			foreignChild(parent).DeepCopyInto(child)

			return nil
		},
	}
}
