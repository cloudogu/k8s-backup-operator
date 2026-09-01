package restore

import (
	"testing"

	"k8s.io/apimachinery/pkg/api/meta"

	k8sv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func restoreWithWorkloadRecoveryReason(reason string) *k8sv1.Restore {
	restore := recoverableRestore()
	applyConditions(restore, []metav1.Condition{{
		Type:    k8sv1.ConditionWorkloadsRecovered,
		Status:  metav1.ConditionUnknown,
		Reason:  reason,
		Message: "Workload recovery progress.",
	}})

	return restore
}

func TestWorkloadsNotReadyPersistWaitingReasonAndRequeue(t *testing.T) {
	restore := restoreWithWorkloadRecoveryReason(ReasonScaleUpInitiated)

	scaleMock := newMockScaleManager(t)
	scaleMock.EXPECT().AreWorkloadsReady(testCtx).Return(false, nil).Once()
	writes := &clientWrites{}
	testClient := newTestClientWithParent(t, writes.interceptor(), restore)
	reconciler := NewRestoreReconciler(testClient, nil, testNamespace, nil, scaleMock, requeueAfterTest, testBackupStorage)

	updated, outcome := reconciler.ensureWorkloadsReady(testCtx, restore)

	assert.Equal(t, retry(), outcome)
	assert.Equal(t, 1, writes.total())
	assertPersistedCondition(t, testClient, k8sv1.ConditionWorkloadsRecovered, metav1.ConditionUnknown, ReasonWaitingForWorkloads)
	condition := meta.FindStatusCondition(updated.Status.Conditions, k8sv1.ConditionWorkloadsRecovered)
	require.NotNil(t, condition)
	assert.Equal(t, ReasonWaitingForWorkloads, condition.Reason)
}

func TestWorkloadsStillNotReadyRequeueWithoutAnotherStatusWrite(t *testing.T) {
	restore := restoreWithWorkloadRecoveryReason(ReasonWaitingForWorkloads)

	scaleMock := newMockScaleManager(t)
	scaleMock.EXPECT().AreWorkloadsReady(testCtx).Return(false, nil).Once()
	writes := &clientWrites{}
	reconciler := NewRestoreReconciler(
		newTestClientWithParent(t, writes.interceptor(), restore), nil, testNamespace, nil, scaleMock, requeueAfterTest, testBackupStorage,
	)

	updated, outcome := reconciler.ensureWorkloadsReady(testCtx, restore)

	assert.Equal(t, retry(), outcome)
	assert.Same(t, restore, updated)
	assert.Equal(t, 0, writes.total())
}

func TestReadyWorkloadsPersistReadyReasonAndRequeue(t *testing.T) {
	restore := restoreWithWorkloadRecoveryReason(ReasonWaitingForWorkloads)

	scaleMock := newMockScaleManager(t)
	scaleMock.EXPECT().AreWorkloadsReady(testCtx).Return(true, nil).Once()
	writes := &clientWrites{}
	testClient := newTestClientWithParent(t, writes.interceptor(), restore)
	reconciler := NewRestoreReconciler(testClient, nil, testNamespace, nil, scaleMock, requeueAfterTest, testBackupStorage)

	_, outcome := reconciler.ensureWorkloadsReady(testCtx, restore)

	assert.Equal(t, retry(), outcome)
	assert.Equal(t, 1, writes.total())
	assertPersistedCondition(t, testClient, k8sv1.ConditionWorkloadsRecovered, metav1.ConditionUnknown, ReasonWorkloadsReady)
}

func TestPersistedReadyReasonIsRecheckedBeforeProceeding(t *testing.T) {
	restore := restoreWithWorkloadRecoveryReason(ReasonWorkloadsReady)

	scaleMock := newMockScaleManager(t)
	scaleMock.EXPECT().AreWorkloadsReady(testCtx).Return(true, nil).Once()
	writes := &clientWrites{}
	reconciler := NewRestoreReconciler(
		newTestClientWithParent(t, writes.interceptor(), restore), nil, testNamespace, nil, scaleMock, requeueAfterTest, testBackupStorage,
	)

	updated, outcome := reconciler.ensureWorkloadsReady(testCtx, restore)

	assert.Equal(t, next(), outcome)
	assert.Same(t, restore, updated)
	assert.Equal(t, 0, writes.total())
}

func TestWorkloadReadinessDoesNotResetFinalizedRecoveryProgress(t *testing.T) {
	restore := restoreWithWorkloadRecoveryReason(ReasonScaleUpFinalized)

	scaleMock := newMockScaleManager(t)
	scaleMock.EXPECT().AreWorkloadsReady(testCtx).Return(true, nil).Once()
	writes := &clientWrites{}
	reconciler := NewRestoreReconciler(
		newTestClientWithParent(t, writes.interceptor(), restore), nil, testNamespace, nil, scaleMock, requeueAfterTest, testBackupStorage,
	)

	_, outcome := reconciler.ensureWorkloadsReady(testCtx, restore)

	assert.Equal(t, next(), outcome)
	assert.Equal(t, 0, writes.total())
}

func TestWorkloadReadinessRegressionReturnsToWaiting(t *testing.T) {
	restore := restoreWithWorkloadRecoveryReason(ReasonWorkloadsReady)

	scaleMock := newMockScaleManager(t)
	scaleMock.EXPECT().AreWorkloadsReady(testCtx).Return(false, nil).Once()
	testClient := newTestClientWithParent(t, interceptor.Funcs{}, restore)
	reconciler := NewRestoreReconciler(testClient, nil, testNamespace, nil, scaleMock, requeueAfterTest, testBackupStorage)

	_, outcome := reconciler.ensureWorkloadsReady(testCtx, restore)

	assert.Equal(t, retry(), outcome)
	assertPersistedCondition(t, testClient, k8sv1.ConditionWorkloadsRecovered, metav1.ConditionUnknown, ReasonWaitingForWorkloads)
}

func TestWorkloadReadinessObservationErrorUsesBackoffWithoutChangingStatus(t *testing.T) {
	restore := restoreWithWorkloadRecoveryReason(ReasonWaitingForWorkloads)

	scaleMock := newMockScaleManager(t)
	scaleMock.EXPECT().AreWorkloadsReady(testCtx).Return(false, assert.AnError).Once()
	writes := &clientWrites{}
	reconciler := NewRestoreReconciler(
		newTestClientWithParent(t, writes.interceptor(), restore), nil, testNamespace, nil, scaleMock, requeueAfterTest, testBackupStorage,
	)

	updated, outcome := reconciler.ensureWorkloadsReady(testCtx, restore)

	require.Error(t, outcome.err)
	assert.ErrorIs(t, outcome.err, assert.AnError)
	assert.ErrorContains(t, outcome.err, "failed to observe workloads after scale-up")
	assert.Same(t, restore, updated)
	assert.Equal(t, 0, writes.total())
}

func TestUnpersistableWorkloadReadinessIsRetried(t *testing.T) {
	restore := restoreWithWorkloadRecoveryReason(ReasonWaitingForWorkloads)

	scaleMock := newMockScaleManager(t)
	scaleMock.EXPECT().AreWorkloadsReady(testCtx).Return(true, nil).Once()
	reconciler := NewRestoreReconciler(
		newTestClientWithParent(t, failingStatusUpdate(assert.AnError), restore),
		nil, testNamespace, nil, scaleMock, requeueAfterTest, testBackupStorage,
	)

	_, outcome := reconciler.ensureWorkloadsReady(testCtx, restore)

	require.Error(t, outcome.err)
	assert.ErrorIs(t, outcome.err, assert.AnError)
	assert.ErrorContains(t, outcome.err, "failed to persist workload readiness for restore test-restore")
}
