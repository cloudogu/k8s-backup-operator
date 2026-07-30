package restore

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	k8sv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
)

var restoreGroupResource = schema.GroupResource{Group: k8sv1.GroupVersion.Group, Resource: "restores"}

func updateStatusEcho(stored *[]*k8sv1.Restore) func(context.Context, *k8sv1.Restore, metav1.UpdateOptions) (*k8sv1.Restore, error) {
	return func(_ context.Context, restore *k8sv1.Restore, _ metav1.UpdateOptions) (*k8sv1.Restore, error) {
		*stored = append(*stored, restore.DeepCopy())

		return restore.DeepCopy(), nil
	}
}

func TestSetConditionsDoesNotWriteWhenTheEffectiveStatusIsUnchanged(t *testing.T) {
	var written []*k8sv1.Restore
	client := newMockRestoreStatusClient(t)
	client.EXPECT().UpdateStatus(testCtx, mock.Anything, metav1.UpdateOptions{}).RunAndReturn(updateStatusEcho(&written)).Once()
	updater := newConditionUpdater(client)

	condition := successful(metav1.ConditionUnknown, ReasonPreparing)

	first, err := updater.setConditions(testCtx, restoreWith(""), condition)
	require.NoError(t, err)

	second, err := updater.setConditions(testCtx, first, condition)
	require.NoError(t, err)

	require.Len(t, written, 1, "the identical second call must not write")
	assert.Same(t, first, second, "an unchanged status must return the restore unchanged")
}

func TestSetConditionsPreservesLastTransitionTimeWhileTheConditionStatusStays(t *testing.T) {
	var written []*k8sv1.Restore
	client := newMockRestoreStatusClient(t)
	client.EXPECT().UpdateStatus(testCtx, mock.Anything, metav1.UpdateOptions{}).RunAndReturn(updateStatusEcho(&written)).Times(3)
	updater := newConditionUpdater(client)

	running := successful(metav1.ConditionUnknown, ReasonProviderRestoreRunning)
	running.Message = "Velero is restoring."

	first, err := updater.setConditions(testCtx, restoreWith(""), running)
	require.NoError(t, err)
	firstCondition := findSuccessfulCondition(first)
	require.NotNil(t, firstCondition)
	transitionedAt := firstCondition.LastTransitionTime
	require.False(t, transitionedAt.IsZero())

	running.Message = "Velero is still restoring."
	second, err := updater.setConditions(testCtx, first, running)
	require.NoError(t, err)

	require.Len(t, written, 2, "a changed message must be written")
	secondCondition := findSuccessfulCondition(second)
	require.NotNil(t, secondCondition)
	assert.Equal(t, transitionedAt, secondCondition.LastTransitionTime)
	assert.Equal(t, "Velero is still restoring.", secondCondition.Message)

	failed := successful(metav1.ConditionFalse, ReasonProviderRestoreFailed)
	third, err := updater.setConditions(testCtx, second, failed)
	require.NoError(t, err)
	thirdCondition := findSuccessfulCondition(third)
	require.NotNil(t, thirdCondition)
	assert.NotEqual(t, transitionedAt, thirdCondition.LastTransitionTime,
		"a changed condition status must move LastTransitionTime")
}

func TestSetConditionsRecordsTheObservedGenerationAndWritesTheLegacyStatus(t *testing.T) {
	var written []*k8sv1.Restore
	client := newMockRestoreStatusClient(t)
	client.EXPECT().UpdateStatus(testCtx, mock.Anything, metav1.UpdateOptions{}).RunAndReturn(updateStatusEcho(&written)).Once()
	updater := newConditionUpdater(client)

	restore := restoreWith("")
	restore.Generation = 7

	updated, err := updater.setConditions(testCtx, restore, successful(metav1.ConditionTrue, ReasonRestoreCompleted))
	require.NoError(t, err)

	condition := findSuccessfulCondition(updated)
	require.NotNil(t, condition)
	assert.Equal(t, int64(7), condition.ObservedGeneration)
	assert.Equal(t, k8sv1.RestoreStatusCompleted, updated.Status.Status)
}

func TestSetConditionsRetriesOnConflictAndKeepsAConcurrentlyWrittenCondition(t *testing.T) {
	concurrent := metav1.Condition{
		Type:    k8sv1.ConditionWorkloadsRecovered,
		Status:  metav1.ConditionTrue,
		Reason:  ReasonRecoveringWorkloads,
		Message: "Written by someone else while we were updating.",
	}
	newerRestore := restoreWith("", concurrent)
	newerRestore.ResourceVersion = "2"

	var written []*k8sv1.Restore
	client := newMockRestoreStatusClient(t)
	client.EXPECT().UpdateStatus(testCtx, mock.Anything, metav1.UpdateOptions{}).
		Return(nil, apierrors.NewConflict(restoreGroupResource, testRestore, assert.AnError)).Once()
	client.EXPECT().Get(testCtx, testRestore, metav1.GetOptions{}).Return(newerRestore, nil).Once()
	client.EXPECT().UpdateStatus(testCtx, mock.Anything, metav1.UpdateOptions{}).RunAndReturn(updateStatusEcho(&written)).Once()
	updater := newConditionUpdater(client)

	stale := restoreWith("")
	stale.ResourceVersion = "1"

	updated, err := updater.setConditions(testCtx, stale, successful(metav1.ConditionTrue, ReasonRestoreCompleted))
	require.NoError(t, err)

	require.Len(t, written, 1)
	assert.Equal(t, "2", written[0].ResourceVersion, "the retry must build on the newer resource version")
	condition := findSuccessfulCondition(updated)
	require.NotNil(t, condition)
	assert.Equal(t, metav1.ConditionTrue, condition.Status)

	survivor := meta.FindStatusCondition(updated.Status.Conditions, k8sv1.ConditionWorkloadsRecovered)
	require.NotNil(t, survivor, "the concurrently written condition must not be dropped")
	assert.Equal(t, "Written by someone else while we were updating.", survivor.Message)
}

func TestSetConditionsReportsAFailedUpdateWithoutRetrying(t *testing.T) {
	client := newMockRestoreStatusClient(t)
	client.EXPECT().UpdateStatus(testCtx, mock.Anything, metav1.UpdateOptions{}).Return(nil, assert.AnError).Once()
	updater := newConditionUpdater(client)

	restore := restoreWith("")

	updated, err := updater.setConditions(testCtx, restore, successful(metav1.ConditionUnknown, ReasonPreparing))

	require.ErrorIs(t, err, assert.AnError)
	assert.Same(t, restore, updated, "a failed update must not pretend to have changed the restore")
}

func TestSetConditionsFromLegacyStatusWritesTheOutcomeOnlyOnce(t *testing.T) {
	var written []*k8sv1.Restore
	client := newMockRestoreStatusClient(t)
	client.EXPECT().UpdateStatus(testCtx, mock.Anything, metav1.UpdateOptions{}).RunAndReturn(updateStatusEcho(&written)).Once()
	updater := newConditionUpdater(client)

	restore, err := updater.setConditionsFromLegacyStatus(testCtx, restoreWith(k8sv1.RestoreStatusCompleted))
	require.NoError(t, err)

	require.Len(t, written, 1)
	condition := findSuccessfulCondition(restore)
	require.NotNil(t, condition)
	assert.Equal(t, metav1.ConditionTrue, condition.Status)
	assert.Equal(t, ReasonMigratedFromLegacyStatus, condition.Reason)
	assert.Equal(t, k8sv1.RestoreStatusCompleted, restore.Status.Status, "the legacy status must survive the migration")
	assert.Len(t, restore.Status.Conditions, 1, "milestone conditions must not be derived")

	again, err := updater.setConditionsFromLegacyStatus(testCtx, restore)
	require.NoError(t, err)
	require.Len(t, written, 1, "a restore that already carries the condition must not be written again")
	assert.Same(t, restore, again)
}

func TestSetConditionsFromLegacyStatusLeavesRestoresWithoutALegacyOutcomeAlone(t *testing.T) {
	client := newMockRestoreStatusClient(t)
	updater := newConditionUpdater(client)

	deleting := restoreWith(k8sv1.RestoreStatusDeleting)
	deleting.DeletionTimestamp = &metav1.Time{Time: time.Now()}

	for _, restore := range []*k8sv1.Restore{restoreWith(k8sv1.RestoreStatusNew), deleting} {
		result, err := updater.setConditionsFromLegacyStatus(testCtx, restore)

		require.NoError(t, err)
		assert.Same(t, restore, result)
		assert.Empty(t, result.Status.Conditions)
	}
}
