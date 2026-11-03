package restore

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	k8sv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
)

func TestSetConditionsDoesNotWriteWhenTheEffectiveStatusIsUnchanged(t *testing.T) {
	restore := restoreWith("")

	writes := &clientWrites{}
	updater := newConditionUpdater(newTestClientWithParent(t, writes.interceptor(), restore))

	condition := successful(metav1.ConditionUnknown, ReasonPreparing)

	first, err := updater.setConditions(testCtx, restore, condition)
	require.NoError(t, err)

	second, err := updater.setConditions(testCtx, first, condition)
	require.NoError(t, err)

	require.Equal(t, 1, writes.parent.statusUpdates, "the identical second call must not write")
	assert.Same(t, first, second, "an unchanged status must return the restore unchanged")
}

func TestSetConditionsPreservesLastTransitionTimeWhileTheConditionStatusStays(t *testing.T) {
	restore := restoreWith("")

	writes := &clientWrites{}
	updater := newConditionUpdater(newTestClientWithParent(t, writes.interceptor(), restore))

	running := successful(metav1.ConditionUnknown, ReasonProviderRestoreRunning)
	running.Message = "The provider is restoring."

	first, err := updater.setConditions(testCtx, restore, running)
	require.NoError(t, err)
	firstCondition := findSuccessfulCondition(first)
	require.NotNil(t, firstCondition)
	transitionedAt := firstCondition.LastTransitionTime
	require.False(t, transitionedAt.IsZero())

	running.Message = "The provider is still restoring."
	second, err := updater.setConditions(testCtx, first, running)
	require.NoError(t, err)

	require.Equal(t, 2, writes.parent.statusUpdates, "a changed message must be written")
	secondCondition := findSuccessfulCondition(second)
	require.NotNil(t, secondCondition)
	assert.Equal(t, transitionedAt, secondCondition.LastTransitionTime)
	assert.Equal(t, "The provider is still restoring.", secondCondition.Message)

	// A persisted timestamp is truncated to seconds, so a wall-clock comparison could not tell two
	// writes within the same second apart. The transition time is therefore given explicitly, which
	// still proves the point: a changed condition status takes the new time, a changed message does not.
	failed := successful(metav1.ConditionFalse, ReasonProviderRestoreFailed)
	failed.LastTransitionTime = metav1.NewTime(transitionedAt.Add(time.Minute))

	third, err := updater.setConditions(testCtx, second, failed)
	require.NoError(t, err)
	thirdCondition := findSuccessfulCondition(third)
	require.NotNil(t, thirdCondition)
	assert.Equal(t, failed.LastTransitionTime, thirdCondition.LastTransitionTime,
		"a changed condition status must move LastTransitionTime")
}

func TestSetConditionsRecordsTheObservedGenerationAndWritesTheLegacyStatus(t *testing.T) {
	restore := restoreWith("")
	restore.Generation = 7

	writes := &clientWrites{}
	updater := newConditionUpdater(newTestClientWithParent(t, writes.interceptor(), restore))

	updated, err := updater.setConditions(testCtx, restore, successful(metav1.ConditionTrue, ReasonRestoreCompleted))
	require.NoError(t, err)

	condition := findSuccessfulCondition(updated)
	require.NotNil(t, condition)
	assert.Equal(t, int64(7), condition.ObservedGeneration)
	assert.Equal(t, k8sv1.RestoreStatusCompleted, updated.Status.Status)

	stored := &k8sv1.Restore{}
	require.NoError(t, updater.client.Get(testCtx, client.ObjectKey{Namespace: testNamespace, Name: testRestore}, stored))
	assert.Equal(t, k8sv1.RestoreStatusCompleted, stored.Status.Status)
}

func TestSetConditionsRetriesOnConflictAndKeepsAConcurrentlyWrittenCondition(t *testing.T) {
	concurrent := metav1.Condition{
		Type:    k8sv1.ConditionWorkloadsRecovered,
		Status:  metav1.ConditionTrue,
		Reason:  ReasonRecoveringWorkloads,
		Message: "Written by someone else while we were updating.",
	}

	// The persisted Restore already carries the concurrent condition, and the caller holds a stale
	// copy without it. The fake client answers the first write with a genuine conflict because of the
	// outdated resource version, so the retry has to refetch to make progress.
	writes := &clientWrites{}
	updater := newConditionUpdater(newTestClientWithParent(t, writes.interceptor(), restoreWith("", concurrent)))

	stored := &k8sv1.Restore{}
	require.NoError(t, updater.client.Get(testCtx, client.ObjectKey{Namespace: testNamespace, Name: testRestore}, stored))
	stale := stored.DeepCopy()
	stale.ResourceVersion = "1"
	stale.Status.Conditions = nil

	updated, err := updater.setConditions(testCtx, stale, successful(metav1.ConditionTrue, ReasonRestoreCompleted))
	require.NoError(t, err)

	require.Equal(t, 2, writes.parent.statusUpdates, "the conflicting write and the successful retry")
	condition := findSuccessfulCondition(updated)
	require.NotNil(t, condition)
	assert.Equal(t, metav1.ConditionTrue, condition.Status)

	stored2 := &k8sv1.Restore{}
	require.NoError(t, updater.client.Get(testCtx, client.ObjectKey{Namespace: testNamespace, Name: testRestore}, stored2))
	survivor := meta.FindStatusCondition(stored2.Status.Conditions, k8sv1.ConditionWorkloadsRecovered)
	require.NotNil(t, survivor, "the concurrently written condition must not be dropped")
	assert.Equal(t, "Written by someone else while we were updating.", survivor.Message)
}

func TestSetConditionsReportsAFailedUpdateWithoutRetrying(t *testing.T) {
	writes := &clientWrites{}
	failing := interceptor.Funcs{
		SubResourceUpdate: func(_ context.Context, _ client.Client, _ string, _ client.Object, _ ...client.SubResourceUpdateOption) error {
			writes.parent.statusUpdates++

			return assert.AnError
		},
	}
	restore := restoreWith("")
	updater := newConditionUpdater(newTestClientWithParent(t, failing, restore))

	updated, err := updater.setConditions(testCtx, restore, successful(metav1.ConditionUnknown, ReasonPreparing))

	require.ErrorIs(t, err, assert.AnError)
	assert.Equal(t, 1, writes.parent.statusUpdates, "a non-conflict error must not be retried")
	assert.Same(t, restore, updated, "a failed update must not pretend to have changed the restore")
}

func TestSetConditionsFromLegacyStatusWritesTheOutcomeOnlyOnce(t *testing.T) {
	legacy := restoreWith(k8sv1.RestoreStatusCompleted)

	writes := &clientWrites{}
	updater := newConditionUpdater(newTestClientWithParent(t, writes.interceptor(), legacy))

	restore, err := updater.setConditionsFromLegacyStatus(testCtx, legacy)
	require.NoError(t, err)

	require.Equal(t, 1, writes.parent.statusUpdates)
	condition := findSuccessfulCondition(restore)
	require.NotNil(t, condition)
	assert.Equal(t, metav1.ConditionTrue, condition.Status)
	assert.Equal(t, ReasonMigratedFromLegacyStatus, condition.Reason)
	assert.Equal(t, k8sv1.RestoreStatusCompleted, restore.Status.Status, "the legacy status must survive the migration")
	assert.Len(t, restore.Status.Conditions, 1, "milestone conditions must not be derived")

	again, err := updater.setConditionsFromLegacyStatus(testCtx, restore)
	require.NoError(t, err)
	require.Equal(t, 1, writes.parent.statusUpdates, "a restore that already carries the condition must not be written again")
	assert.Same(t, restore, again)
}

func TestSetConditionsFromLegacyStatusLeavesRestoresWithoutALegacyOutcomeAlone(t *testing.T) {
	writes := &clientWrites{}
	updater := newConditionUpdater(newTestClientWithParent(t, writes.interceptor(), restoreWith(k8sv1.RestoreStatusNew)))

	deleting := restoreWith(k8sv1.RestoreStatusDeleting)
	deleting.DeletionTimestamp = &metav1.Time{Time: time.Now()}

	for _, restore := range []*k8sv1.Restore{restoreWith(k8sv1.RestoreStatusNew), deleting} {
		result, err := updater.setConditionsFromLegacyStatus(testCtx, restore)

		require.NoError(t, err)
		assert.Same(t, restore, result)
		assert.Empty(t, result.Status.Conditions)
	}

	assert.Equal(t, 0, writes.total())
}
