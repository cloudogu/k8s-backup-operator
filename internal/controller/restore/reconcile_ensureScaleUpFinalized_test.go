package restore

import (
	"testing"

	"k8s.io/apimachinery/pkg/api/meta"

	k8sv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestScaleUpFinalizationPersistsItsProgressAndRequeues(t *testing.T) {
	restore := restoreWithWorkloadRecoveryReason(ReasonWorkloadsReady)

	scaleMock := newMockScaleManager(t)
	scaleMock.EXPECT().FinalizeScaleUp(testCtx).Return(nil).Once()
	writes := &clientWrites{}
	testClient := newTestClientWithParent(t, writes.interceptor(), restore)
	reconciler := NewRestoreReconciler(testClient, nil, testNamespace, nil, scaleMock, requeueAfterTest, testBackupStorage)

	updated, outcome := reconciler.ensureScaleUpFinalized(testCtx, restore)

	assert.Equal(t, retryAfter(defaultRequeueDelay), outcome)
	assert.Equal(t, 1, writes.total())
	assertPersistedCondition(t, testClient, k8sv1.ConditionWorkloadsRecovered, metav1.ConditionUnknown, ReasonScaleUpFinalized)
	condition := meta.FindStatusCondition(updated.Status.Conditions, k8sv1.ConditionWorkloadsRecovered)
	require.NotNil(t, condition)
	assert.Equal(t, ReasonScaleUpFinalized, condition.Reason)
}

func TestScaleUpFinalizationIsEnsuredAgainBeforeProceeding(t *testing.T) {
	restore := restoreWithWorkloadRecoveryReason(ReasonScaleUpFinalized)

	scaleMock := newMockScaleManager(t)
	scaleMock.EXPECT().FinalizeScaleUp(testCtx).Return(nil).Once()
	writes := &clientWrites{}
	reconciler := NewRestoreReconciler(
		newTestClientWithParent(t, writes.interceptor(), restore), nil, testNamespace, nil, scaleMock, requeueAfterTest, testBackupStorage,
	)

	updated, outcome := reconciler.ensureScaleUpFinalized(testCtx, restore)

	assert.Equal(t, next(), outcome)
	assert.Same(t, restore, updated)
	assert.Equal(t, 0, writes.total())
}

func TestFailedScaleUpFinalizationReportsRecoveryFalseAndRetries(t *testing.T) {
	restore := restoreWithWorkloadRecoveryReason(ReasonWorkloadsReady)

	scaleMock := newMockScaleManager(t)
	scaleMock.EXPECT().FinalizeScaleUp(testCtx).Return(assert.AnError).Once()
	recorderMock := newMockEventRecorder(t)
	recorderMock.EXPECT().Event(
		matchesRestoreNamed(testRestore),
		corev1.EventTypeWarning,
		ReasonWorkloadRecoveryFailed,
		"failed to finalize workload scale-up after restore",
	).Return()
	recorderMock.EXPECT().Event(
		matchesRestoreNamed(testRestore),
		corev1.EventTypeWarning,
		k8sv1.ErrorOnCreateEventReason,
		"failed to finalize workload scale-up after restore: assert.AnError general error for testing",
	).Return()
	testClient := newTestClientWithParent(t, interceptor.Funcs{}, restore)
	reconciler := NewRestoreReconciler(testClient, recorderMock, testNamespace, nil, scaleMock, requeueAfterTest, testBackupStorage)

	_, outcome := reconciler.ensureScaleUpFinalized(testCtx, restore)

	require.Error(t, outcome.err)
	assert.ErrorIs(t, outcome.err, assert.AnError)
	assert.ErrorContains(t, outcome.err, "failed to finalize workload scale-up after restore")
	assertPersistedCondition(t, testClient, k8sv1.ConditionWorkloadsRecovered, metav1.ConditionFalse, ReasonWorkloadRecoveryFailed)
}

func TestUnpersistableScaleUpFinalizationIsRetried(t *testing.T) {
	restore := restoreWithWorkloadRecoveryReason(ReasonWorkloadsReady)

	scaleMock := newMockScaleManager(t)
	scaleMock.EXPECT().FinalizeScaleUp(testCtx).Return(nil).Once()
	reconciler := NewRestoreReconciler(
		newTestClientWithParent(t, failingStatusUpdate(assert.AnError), restore),
		nil, testNamespace, nil, scaleMock, requeueAfterTest, testBackupStorage,
	)

	_, outcome := reconciler.ensureScaleUpFinalized(testCtx, restore)

	require.Error(t, outcome.err)
	assert.ErrorIs(t, outcome.err, assert.AnError)
	assert.ErrorContains(t, outcome.err, "failed to persist finalized workload scale-up for restore test-restore")
}
