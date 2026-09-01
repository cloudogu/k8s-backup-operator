package restore

import (
	"testing"

	k8sv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func completableRestore() *k8sv1.Restore {
	return restoreWithWorkloadRecoveryReason(ReasonMaintenanceModeDeactivated)
}

func TestRestoreCompletionPersistsTerminalConditionsWithoutExternalRecoveryActions(t *testing.T) {
	restore := completableRestore()

	recorderMock := newMockEventRecorder(t)
	recorderMock.EXPECT().Event(
		matchesRestoreNamed(testRestore), corev1.EventTypeNormal, ReasonRestoreCompleted, "Restore successful",
	).Return()
	writes := &clientWrites{}
	testClient := newTestClientWithParent(t, writes.interceptor(), restore)
	reconciler := NewRestoreReconciler(testClient, recorderMock, testNamespace, nil, nil, requeueAfterTest, testBackupStorage)

	updated, outcome := reconciler.ensureRestoreCompleted(testCtx, restore)

	assert.Equal(t, abort(), outcome)
	assert.Equal(t, 1, writes.total())
	assertPersistedCondition(t, testClient, k8sv1.ConditionWorkloadsRecovered, metav1.ConditionTrue, ReasonWorkloadRecoveryCompleted)
	assertPersistedCondition(t, testClient, k8sv1.ConditionSucceeded, metav1.ConditionTrue, ReasonRestoreCompleted)
	assert.Equal(t, k8sv1.RestoreStatusCompleted, updated.Status.Status)
}

func TestRestoreCompletionDoesNotCompleteBeforeMaintenanceModeDeactivation(t *testing.T) {
	restore := restoreWithWorkloadRecoveryReason(ReasonScaleUpFinalized)

	writes := &clientWrites{}
	reconciler := NewRestoreReconciler(
		newTestClientWithParent(t, writes.interceptor(), restore), nil, testNamespace, nil, nil, requeueAfterTest, testBackupStorage,
	)

	updated, outcome := reconciler.ensureRestoreCompleted(testCtx, restore)

	assert.Equal(t, retry(), outcome)
	assert.Same(t, restore, updated)
	assert.Equal(t, 0, writes.total())
}

func TestRepeatedRestoreCompletionIsANoOp(t *testing.T) {
	restore := completableRestore()
	applyConditions(restore, []metav1.Condition{
		reachedMilestone(k8sv1.ConditionWorkloadsRecovered, ReasonWorkloadRecoveryCompleted, "Recovered."),
		reachedMilestone(k8sv1.ConditionSucceeded, ReasonRestoreCompleted, "Completed."),
	})

	writes := &clientWrites{}
	reconciler := NewRestoreReconciler(
		newTestClientWithParent(t, writes.interceptor(), restore), nil, testNamespace, nil, nil, requeueAfterTest, testBackupStorage,
	)

	updated, outcome := reconciler.ensureRestoreCompleted(testCtx, restore)

	assert.Equal(t, abort(), outcome)
	assert.Same(t, restore, updated)
	assert.Equal(t, 0, writes.total())
}

func TestUnpersistableRestoreCompletionIsRetriedWithoutSuccessEvent(t *testing.T) {
	restore := completableRestore()

	reconciler := NewRestoreReconciler(
		newTestClientWithParent(t, failingStatusUpdate(assert.AnError), restore),
		nil, testNamespace, nil, nil, requeueAfterTest, testBackupStorage,
	)

	_, outcome := reconciler.ensureRestoreCompleted(testCtx, restore)

	require.Error(t, outcome.err)
	assert.ErrorIs(t, outcome.err, assert.AnError)
	assert.ErrorContains(t, outcome.err, "failed to persist the completed restore test-restore")
}
