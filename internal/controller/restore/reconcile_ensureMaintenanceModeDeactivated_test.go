package restore

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"

	k8sv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestMaintenanceModeDeactivationPersistsItsProgressAndRequeues(t *testing.T) {
	restore := restoreWithWorkloadRecoveryReason(ReasonScaleUpFinalized)

	maintenanceMock := newMockMaintenanceModeSwitch(t)
	maintenanceMock.EXPECT().Deactivate(testCtx, false).Return(nil).Once()
	writes := &clientWrites{}
	testClient := newTestClientWithParent(t, writes.interceptor(), restore)
	reconciler := NewRestoreReconciler(testClient, nil, testNamespace, nil, nil, requeueAfterTest)
	reconciler.maintenanceModeSwitch = maintenanceMock

	updated, outcome := reconciler.ensureMaintenanceModeDeactivated(testCtx, restore)

	assert.Equal(t, retryAfter(defaultRequeueDelay), outcome)
	assert.Equal(t, 1, writes.total())
	assertPersistedCondition(t, testClient, k8sv1.ConditionWorkloadsRecovered, metav1.ConditionUnknown, ReasonMaintenanceModeDeactivated)
	condition := meta.FindStatusCondition(updated.Status.Conditions, k8sv1.ConditionWorkloadsRecovered)
	require.NotNil(t, condition)
	assert.Equal(t, metav1.ConditionUnknown, condition.Status)
	assert.Equal(t, ReasonMaintenanceModeDeactivated, condition.Reason)
}

func TestMaintenanceModeDeactivationIsEnsuredAgainBeforeProceeding(t *testing.T) {
	restore := restoreWithWorkloadRecoveryReason(ReasonMaintenanceModeDeactivated)
	recorderMock := newMockEventRecorder(t)
	recorderMock.EXPECT().Event(restore, corev1.EventTypeNormal, ReasonMaintenanceModeDeactivated, "Maintenance mode deactivated").Once()

	maintenanceMock := newMockMaintenanceModeSwitch(t)
	maintenanceMock.EXPECT().Deactivate(testCtx, false).Return(nil).Once()
	writes := &clientWrites{}
	reconciler := NewRestoreReconciler(
		newTestClientWithParent(t, writes.interceptor(), restore), recorderMock, testNamespace, nil, nil, requeueAfterTest,
	)
	reconciler.maintenanceModeSwitch = maintenanceMock

	updated, outcome := reconciler.ensureMaintenanceModeDeactivated(testCtx, restore)

	assert.Equal(t, next(), outcome)
	assert.Same(t, restore, updated)
	assert.Equal(t, 0, writes.total())
}

func TestFailedMaintenanceModeDeactivationUsesBackoffWithoutChangingProgress(t *testing.T) {
	restore := restoreWithWorkloadRecoveryReason(ReasonScaleUpFinalized)
	recorderMock := newMockEventRecorder(t)
	recorderMock.EXPECT().Event(restore, corev1.EventTypeWarning, ReasonMaintenanceModeDeactivated, "Failed to deactivate maintenance mode after restore").Once()

	maintenanceMock := newMockMaintenanceModeSwitch(t)
	maintenanceMock.EXPECT().Deactivate(testCtx, false).Return(assert.AnError).Once()
	writes := &clientWrites{}
	testClient := newTestClientWithParent(t, writes.interceptor(), restore)
	reconciler := NewRestoreReconciler(testClient, recorderMock, testNamespace, nil, nil, requeueAfterTest)
	reconciler.maintenanceModeSwitch = maintenanceMock

	updated, outcome := reconciler.ensureMaintenanceModeDeactivated(testCtx, restore)

	require.Error(t, outcome.err)
	assert.ErrorIs(t, outcome.err, assert.AnError)
	assert.ErrorContains(t, outcome.err, "failed to deactivate maintenance mode after restore test-restore")
	assert.Same(t, restore, updated)
	assert.Equal(t, 0, writes.total())
	assertPersistedCondition(t, testClient, k8sv1.ConditionWorkloadsRecovered, metav1.ConditionUnknown, ReasonScaleUpFinalized)
}

func TestUnpersistableMaintenanceModeDeactivationIsRetried(t *testing.T) {
	restore := restoreWithWorkloadRecoveryReason(ReasonScaleUpFinalized)
	recorderMock := newMockEventRecorder(t)
	recorderMock.EXPECT().Event(restore, corev1.EventTypeWarning, ReasonMaintenanceModeDeactivated, "Failed to persist maintenance mode deactivation for restore").Once()

	maintenanceMock := newMockMaintenanceModeSwitch(t)
	maintenanceMock.EXPECT().Deactivate(testCtx, false).Return(nil).Once()
	reconciler := NewRestoreReconciler(
		newTestClientWithParent(t, failingStatusUpdate(assert.AnError), restore),
		recorderMock, testNamespace, nil, nil, requeueAfterTest,
	)
	reconciler.maintenanceModeSwitch = maintenanceMock

	_, outcome := reconciler.ensureMaintenanceModeDeactivated(testCtx, restore)

	require.Error(t, outcome.err)
	assert.ErrorIs(t, outcome.err, assert.AnError)
	assert.ErrorContains(t, outcome.err, "failed to persist maintenance mode deactivation for restore test-restore")
}
