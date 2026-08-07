package restore

import (
	"context"
	"testing"

	k8sv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/k8s-backup-operator/internal/provider/velero"
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

// ownedRunningChild returns the provider restore of the given Restore, executing.
func ownedRunningChild(parent *k8sv1.Restore) *velerov1.Restore {
	child := velero.BuildRestore(parent)
	child.Status.Phase = velerov1.RestorePhaseInProgress

	return child
}

// foreignChild returns a provider restore occupying the name the given Restore expects, without an
// owner reference, so it must never be adopted.
func foreignChild(parent *k8sv1.Restore) *velerov1.Restore {
	child := velero.BuildRestore(parent)
	child.OwnerReferences = nil
	child.Status.Phase = velerov1.RestorePhaseInProgress

	return child
}

// An existing owned child must send the workflow past preparation even without a set condition
func TestAnExistingOwnedProviderChildSkipsThePreparationAndIsObservedInstead(t *testing.T) {
	restore := withInitializedConditions(withMetadata(newParentRestore()))

	factory := func(fakeClient client.WithWatch) reconcileFunction {
		return NewRestoreReconciler(fakeClient, nil, testNamespace, nil, nil).Reconcile
	}
	// with owned child
	fixture := newMultiReconcileFixture(t, interceptor.Funcs{}, factory, restore, ownedRunningChild(restore))

	results, errs := fixture.reconcileTimes(testCtx, newRestoreRequest(testRestore), 2)

	for index := range results {
		require.NoError(t, errs[index])
		assert.Equal(t, ctrl.Result{RequeueAfter: providerObservationRecoveryDelay}, results[index],
			"an undecided provider restore must not be waited for")
	}
	assert.Equal(t, []recordedClientAction{statusUpdateOf(restore)}, fixture.clientActions.snapshot(),
		"the observed state must be written once and the child must never be written")
	assertPersistedCondition(t, fixture.client, k8sv1.ConditionProviderRestoreSuccessful, metav1.ConditionUnknown, ReasonProviderRestoreRunning)
}

func TestAConflictingProviderChildFailsTheRestoreBeforePreparation(t *testing.T) {
	restore := withInitializedConditions(withMetadata(newParentRestore()))
	conflicting := foreignChild(restore)

	recorderMock := newMockEventRecorder(t)
	recorderMock.EXPECT().Event(matchesRestoreNamed(testRestore), corev1.EventTypeWarning, k8sv1.ErrorOnCreateEventReason,
		"velero restore [test-restore] conflicts with the expected child: it is not controlled by restore [test-restore] and must not be adopted").Return()

	factory := func(fakeClient client.WithWatch) reconcileFunction {
		return NewRestoreReconciler(fakeClient, recorderMock, testNamespace, nil, nil).Reconcile
	}
	fixture := newMultiReconcileFixture(t, interceptor.Funcs{}, factory, restore, conflicting)
	request := newRestoreRequest(testRestore)

	results, errs := fixture.reconcileTimes(testCtx, request, 2)

	for index := range results {
		require.NoError(t, errs[index])
		assert.Equal(t, ctrl.Result{}, results[index], "a conflict is terminal, so it must not be requeued")
	}
	assert.Equal(t, []recordedClientAction{statusUpdateOf(restore)}, fixture.clientActions.snapshot(),
		"the conflict must be reported exactly once and the child must not be touched")

	stored := &k8sv1.Restore{}
	require.NoError(t, fixture.client.Get(testCtx, request.NamespacedName, stored))
	assert.Equal(t, k8sv1.RestoreStatusFailed, stored.Status.Status)
	assert.Equal(t, metav1.ConditionFalse, findSuccessfulCondition(stored).Status)
	assert.Equal(t, ReasonProviderRestoreConflict, findSuccessfulCondition(stored).Reason)
	providerCondition := meta.FindStatusCondition(stored.Status.Conditions, k8sv1.ConditionProviderRestoreSuccessful)
	require.NotNil(t, providerCondition)
	assert.Equal(t, metav1.ConditionFalse, providerCondition.Status)
	assert.Equal(t, ReasonProviderRestoreConflict, providerCondition.Reason)
}

func TestAnUnreportableProviderChildConflictIsRetried(t *testing.T) {
	restore := withInitializedConditions(withMetadata(newParentRestore()))

	recorderMock := newMockEventRecorder(t)
	recorderMock.EXPECT().Event(matchesRestoreNamed(testRestore), corev1.EventTypeWarning, k8sv1.ErrorOnCreateEventReason, mock.Anything).Return()

	factory := func(fakeClient client.WithWatch) reconcileFunction {
		return NewRestoreReconciler(fakeClient, recorderMock, testNamespace, nil, nil).Reconcile
	}
	fixture := newMultiReconcileFixture(t, failingStatusUpdate(assert.AnError), factory, restore, foreignChild(restore))

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
		return NewRestoreReconciler(fakeClient, nil, testNamespace, nil, nil).Reconcile
	}
	fixture := newMultiReconcileFixture(t, failingChildGet, factory, restore)

	_, errs := fixture.reconcileTimes(testCtx, newRestoreRequest(testRestore), 1)

	require.ErrorIs(t, errs[0], assert.AnError)
	assert.ErrorContains(t, errs[0], "failed to read the provider restore of restore test-restore")
}
