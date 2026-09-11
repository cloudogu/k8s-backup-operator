package backup

import (
	"context"
	"testing"
	"time"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func canceledBackupForReconcilerTest(namespace string, name string) *backupv1.Backup {
	backup := newBackupForTest(namespace, name)
	backup.Status.StartTimestamp = metav1.NewTime(time.Now())
	meta.SetStatusCondition(&backup.Status.Conditions, metav1.Condition{
		Type:   backupv1.ConditionCanceled,
		Status: metav1.ConditionTrue,
		Reason: reasonTimeWindowExpiredBackupInProgress,
	})

	return backup
}

func getDeleteBackupRequestForTest(t *testing.T, k8sClient client.Client, namespace string, name string) *velerov1.DeleteBackupRequest {
	t.Helper()
	deleteRequest := &velerov1.DeleteBackupRequest{}
	require.NoError(t, k8sClient.Get(context.Background(), types.NamespacedName{Namespace: namespace, Name: name}, deleteRequest))

	return deleteRequest
}

func TestReconcilerEnsureCanceledProviderBackupDeleted(t *testing.T) {
	t.Run("A backup that was not canceled owns its provider backup and keeps it", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		veleroBackup := newVeleroBackupForReconcilerTest("ns", "backup", velerov1.BackupPhaseCompleted)
		fakeClient := newFakeClientBuilder(t).
			WithObjects(backup, veleroBackup).
			WithStatusSubresource(backup).
			Build()
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default")

		nextAction, err := reconciler.ensureCanceledProviderBackupDeleted(context.Background(), backup)

		assert.NoError(t, err)
		assert.Equal(t, Next, nextAction)

		err = fakeClient.Get(context.Background(), types.NamespacedName{Namespace: "ns", Name: "backup"}, &velerov1.DeleteBackupRequest{})
		assert.True(t, apierrors.IsNotFound(err), "the provider backup of a backup that was not canceled must not be deleted")
	})

	t.Run("Report the completion once when the provider backup of a canceled backup is gone", func(t *testing.T) {
		backup := canceledBackupForReconcilerTest("ns", "backup")
		fakeClient := newFakeClientBuilder(t).
			WithObjects(backup).
			WithStatusSubresource(backup).
			Build()
		recorder := events.NewFakeRecorder(10)
		reconciler := NewReconciler(fakeClient, recorder, nil, newRealClock(), "default")

		nextAction, err := reconciler.ensureCanceledProviderBackupDeleted(context.Background(), backup)

		assert.NoError(t, err)
		assert.Equal(t, Next, nextAction, "a settled backup must not requeue forever")

		deletingCondition := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionDeleting)
		require.NotNil(t, deletingCondition)
		assert.Equal(t, metav1.ConditionFalse, deletingCondition.Status)
		assert.Equal(t, reasonCanceledProviderBackupDeleted, deletingCondition.Reason,
			"the Deleting condition must not keep reporting a deletion that is over")

		err = fakeClient.Get(context.Background(), types.NamespacedName{Namespace: "ns", Name: "backup"}, &velerov1.DeleteBackupRequest{})
		assert.True(t, apierrors.IsNotFound(err), "there is nothing left to request a deletion for")

		// second pass to verify the completion is reported only once
		nextAction, err = reconciler.ensureCanceledProviderBackupDeleted(context.Background(), backup)
		assert.NoError(t, err)
		assert.Equal(t, Next, nextAction)
		require.Len(t, recorder.Events, 1, "the completed deletion must be reported only once")
		assert.Contains(t, <-recorder.Events, reasonCanceledProviderBackupDeleted)
	})

	t.Run("A canceled backup that never started left no provider backup to report on", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		meta.SetStatusCondition(&backup.Status.Conditions, metav1.Condition{
			Type:   backupv1.ConditionCanceled,
			Status: metav1.ConditionTrue,
			Reason: reasonTimeWindowExpiredBackupNotStarted,
		})
		require.True(t, backup.Status.StartTimestamp.IsZero())
		counter := &callCounter{}
		fakeClient := newFakeClientBuilderWithCounter(t, counter).
			WithObjects(backup).
			WithStatusSubresource(backup).
			Build()
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default")

		nextAction, err := reconciler.ensureCanceledProviderBackupDeleted(context.Background(), backup)

		assert.NoError(t, err)
		assert.Equal(t, Next, nextAction)
		assert.Equal(t, 0, counter.veleroBackupGetCount, "a run that never started has no provider backup to look for")
		assert.Nil(t, meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionDeleting))
	})

	t.Run("Request the deletion of the provider backup a canceled run left behind", func(t *testing.T) {
		backup := canceledBackupForReconcilerTest("ns", "backup")
		veleroBackup := newVeleroBackupForReconcilerTest("ns", "backup", velerov1.BackupPhasePartiallyFailed)
		fakeClient := newFakeClientBuilder(t).
			WithObjects(backup, veleroBackup).
			WithStatusSubresource(backup).
			Build()
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default")

		nextAction, err := reconciler.ensureCanceledProviderBackupDeleted(context.Background(), backup)

		assert.NoError(t, err)
		assert.Equal(t, Retry, nextAction, "the deletion of the provider backup has to be followed up on")

		deleteRequest := getDeleteBackupRequestForTest(t, fakeClient, "ns", "backup")
		assert.Equal(t, "backup", deleteRequest.Spec.BackupName)

		deletingCondition := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionDeleting)
		require.NotNil(t, deletingCondition)
		assert.Equal(t, metav1.ConditionFalse, deletingCondition.Status,
			"the canceled backup itself is not being deleted, only its provider backup")
		assert.Equal(t, reasonBackupDeleting, deletingCondition.Reason)
		assert.Contains(t, deletingCondition.Message, "Provider backup of canceled backup")
	})

	t.Run("Wait for a provider backup that is still running before requesting its deletion", func(t *testing.T) {
		backup := canceledBackupForReconcilerTest("ns", "backup")
		veleroBackup := newVeleroBackupForReconcilerTest("ns", "backup", velerov1.BackupPhaseWaitingForPluginOperations)
		fakeClient := newFakeClientBuilder(t).
			WithObjects(backup, veleroBackup).
			WithStatusSubresource(backup).
			Build()
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default")

		nextAction, err := reconciler.ensureCanceledProviderBackupDeleted(context.Background(), backup)

		assert.NoError(t, err)
		assert.Equal(t, Retry, nextAction)

		err = fakeClient.Get(context.Background(), types.NamespacedName{Namespace: "ns", Name: "backup"}, &velerov1.DeleteBackupRequest{})
		assert.True(t, apierrors.IsNotFound(err), "velero refuses to delete a running backup, so no request is kept around")

		deletingCondition := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionDeleting)
		require.NotNil(t, deletingCondition)
		assert.Equal(t, metav1.ConditionFalse, deletingCondition.Status)
		assert.Equal(t, reasonWaitingForProviderBackupCompletion, deletingCondition.Reason)
	})

	t.Run("Abort when the provider backup of a canceled backup cannot be read", func(t *testing.T) {
		backup := canceledBackupForReconcilerTest("ns", "backup")
		counter := &callCounter{veleroBackupGetCallError: assert.AnError}
		fakeClient := newFakeClientBuilderWithCounter(t, counter).
			WithObjects(backup).
			WithStatusSubresource(backup).
			Build()
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default")

		nextAction, err := reconciler.ensureCanceledProviderBackupDeleted(context.Background(), backup)

		assert.Error(t, err)
		assert.Equal(t, Abort, nextAction)
	})
}
