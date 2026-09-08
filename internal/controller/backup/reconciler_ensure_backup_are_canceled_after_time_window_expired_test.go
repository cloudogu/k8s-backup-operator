package backup

import (
	"context"
	"strconv"
	"testing"
	"time"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestReconcilerEnsureBackupAreCanceledAfterTimeWindowExpired(t *testing.T) {
	t.Run("If the time window has not yet expired, set canceled to false and proceed to the next step", func(t *testing.T) {
		baseTime := time.Now()
		timeLimitInMinutes := 10
		backup := newBackupForTest("ns", "backup")
		backup.CreationTimestamp = metav1.NewTime(baseTime)
		backupConfigMap := newBackupConfigMapForReconcilerTest(timeLimitInMinutes)
		counter := &callCounter{}
		fakeClient := newFakeClientBuilderWithCounter(t, counter).
			WithObjects(backup, backupConfigMap).
			WithStatusSubresource(backup).
			Build()
		clockMock := NewMockClock(t)
		clockMock.EXPECT().
			Now().
			Return(baseTime.Add(time.Duration(timeLimitInMinutes)*time.Minute - time.Millisecond))
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, clockMock, "default")

		nextAction, err := reconciler.ensureBackupIsCanceledAfterTimeWindowExpired(context.Background(), backup)

		assert.NoError(t, err)
		assert.Equal(t, Next, nextAction)

		canceledCondition := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionCanceled)
		assert.NotNil(t, canceledCondition)
		assert.Equal(t, metav1.ConditionFalse, canceledCondition.Status)
		assert.Equal(t, reasonTimeWindowNotExpired, canceledCondition.Reason)

		assert.Equal(t, 1, counter.configMapGetCount)
		assert.Equal(t, 1, counter.subResourcePatchCount)
	})

	t.Run("If the time window has expired and the backup has not started, set canceled to true and requeue for the finalization", func(t *testing.T) {
		baseTime := time.Now()
		backup := newBackupForTest("ns", "backup")
		backup.CreationTimestamp = metav1.NewTime(baseTime)
		backupConfigMap := newBackupConfigMapForReconcilerTest(10)
		counter := &callCounter{}
		fakeClient := newFakeClientBuilderWithCounter(t, counter).
			WithObjects(backup, backupConfigMap).
			WithStatusSubresource(backup).
			Build()
		clockMock := NewMockClock(t)
		clockMock.EXPECT().
			Now().
			Return(baseTime.Add(10*time.Minute + time.Millisecond))
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, clockMock, "default")

		require.True(t, backup.Status.StartTimestamp.IsZero())

		nextAction, err := reconciler.ensureBackupIsCanceledAfterTimeWindowExpired(context.Background(), backup)

		assert.NoError(t, err)
		assert.Equal(t, Retry, nextAction)

		canceledCondition := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionCanceled)
		assert.NotNil(t, canceledCondition)
		assert.Equal(t, metav1.ConditionTrue, canceledCondition.Status)
		assert.Equal(t, reasonTimeWindowExpiredBackupNotStarted, canceledCondition.Reason)

		assert.Equal(t, 1, counter.configMapGetCount)
		assert.Equal(t, 1, counter.subResourcePatchCount)
	})

	t.Run("If the time window has expired and the velero backup is still running, set canceled to true and requeue for the finalization", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		baseTime := time.Now()
		backup.CreationTimestamp = metav1.NewTime(baseTime)
		backup.Status.StartTimestamp = metav1.NewTime(baseTime.Add(2 * time.Minute))
		backupConfigMap := newBackupConfigMapForReconcilerTest(10)
		veleroBackup := newVeleroBackupForReconcilerTest("ns", "backup", velerov1.BackupPhaseInProgress)
		counter := &callCounter{}
		fakeClient := newFakeClientBuilderWithCounter(t, counter).
			WithObjects(backup, backupConfigMap, veleroBackup).
			WithStatusSubresource(backup).
			Build()
		clockMock := NewMockClock(t)
		clockMock.EXPECT().
			Now().
			Return(baseTime.Add(10*time.Minute + 5*time.Minute))
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, clockMock, "default")

		nextAction, err := reconciler.ensureBackupIsCanceledAfterTimeWindowExpired(context.Background(), backup)

		assert.NoError(t, err)
		assert.Equal(t, Retry, nextAction)

		canceledCondition := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionCanceled)
		assert.NotNil(t, canceledCondition)
		assert.Equal(t, metav1.ConditionTrue, canceledCondition.Status)
		assert.Equal(t, reasonTimeWindowExpiredBackupInProgress, canceledCondition.Reason)

		assert.Equal(t, 1, counter.configMapGetCount)
		assert.Equal(t, 1, counter.veleroBackupGetCount)
		assert.Equal(t, 1, counter.subResourcePatchCount)
	})

	t.Run("If the time window has expired and the velero backup has failed, proceed to the next step", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		baseTime := time.Now()
		backup.CreationTimestamp = metav1.NewTime(baseTime)
		backup.Status.StartTimestamp = metav1.NewTime(baseTime.Add(2 * time.Minute))
		backupConfigMap := newBackupConfigMapForReconcilerTest(10)
		veleroBackup := newVeleroBackupForReconcilerTest("ns", "backup", velerov1.BackupPhaseFailed)
		counter := &callCounter{}
		fakeClient := newFakeClientBuilderWithCounter(t, counter).
			WithObjects(backup, backupConfigMap, veleroBackup).
			WithStatusSubresource(backup).
			Build()
		clockMock := NewMockClock(t)
		clockMock.EXPECT().
			Now().
			Return(baseTime.Add(10*time.Minute + 5*time.Minute))
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, clockMock, "default")

		nextAction, err := reconciler.ensureBackupIsCanceledAfterTimeWindowExpired(context.Background(), backup)

		assert.NoError(t, err)
		assert.Equal(t, Next, nextAction)

		canceledCondition := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionCanceled)
		assert.NotNil(t, canceledCondition)
		assert.Equal(t, metav1.ConditionFalse, canceledCondition.Status)
		assert.Equal(t, reasonTimeWindowExpiredBackupTerminated, canceledCondition.Reason)

		assert.Equal(t, 1, counter.configMapGetCount)
		assert.Equal(t, 1, counter.veleroBackupGetCount)
		assert.Equal(t, 1, counter.subResourcePatchCount)
	})

	t.Run("If the time window has expired and the velero backup no longer exists, set canceled to true and requeue", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		baseTime := time.Now()
		backup.CreationTimestamp = metav1.NewTime(baseTime)
		backup.Status.StartTimestamp = metav1.NewTime(baseTime.Add(2 * time.Minute))
		backupConfigMap := newBackupConfigMapForReconcilerTest(10)
		counter := &callCounter{}
		fakeClient := newFakeClientBuilderWithCounter(t, counter).
			WithObjects(backup, backupConfigMap).
			WithStatusSubresource(backup).
			Build()
		clockMock := NewMockClock(t)
		clockMock.EXPECT().
			Now().
			Return(baseTime.Add(10*time.Minute + 5*time.Minute))
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, clockMock, "default")

		nextAction, err := reconciler.ensureBackupIsCanceledAfterTimeWindowExpired(context.Background(), backup)

		assert.NoError(t, err)
		assert.Equal(t, Retry, nextAction)

		canceledCondition := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionCanceled)
		assert.NotNil(t, canceledCondition)
		assert.Equal(t, metav1.ConditionTrue, canceledCondition.Status)
		assert.Equal(t, reasonTimeWindowExpiredProviderBackupMissing, canceledCondition.Reason)

		assert.Equal(t, 1, counter.configMapGetCount)
		assert.Equal(t, 1, counter.veleroBackupGetCount)
		assert.Equal(t, 1, counter.subResourcePatchCount)
	})

	t.Run("If the time window has expired and the velero backup has succeeded, proceed to the next step", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		baseTime := time.Now()
		backup.CreationTimestamp = metav1.NewTime(baseTime)
		backup.Status.StartTimestamp = metav1.NewTime(baseTime.Add(2 * time.Minute))
		backupConfigMap := newBackupConfigMapForReconcilerTest(10)
		veleroBackup := newVeleroBackupForReconcilerTest("ns", "backup", velerov1.BackupPhaseCompleted)
		counter := &callCounter{}
		fakeClient := newFakeClientBuilderWithCounter(t, counter).
			WithObjects(backup, backupConfigMap, veleroBackup).
			WithStatusSubresource(backup).
			Build()
		clockMock := NewMockClock(t)
		clockMock.EXPECT().
			Now().
			Return(baseTime.Add(10*time.Minute + 5*time.Minute))
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, clockMock, "default")

		nextAction, err := reconciler.ensureBackupIsCanceledAfterTimeWindowExpired(context.Background(), backup)

		assert.NoError(t, err)
		assert.Equal(t, Next, nextAction)

		canceledCondition := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionCanceled)
		assert.NotNil(t, canceledCondition)
		assert.Equal(t, metav1.ConditionFalse, canceledCondition.Status)
		assert.Equal(t, reasonTimeWindowExpiredBackupTerminated, canceledCondition.Reason)

		assert.Equal(t, 1, counter.configMapGetCount)
		assert.Equal(t, 1, counter.veleroBackupGetCount)
		assert.Equal(t, 1, counter.subResourcePatchCount)
	})
}

func newBackupConfigMapForReconcilerTest(retryLimitInMinutes int) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      backupConfigMapName,
		},
		Data: map[string]string{
			backupRetryTimeLimitKey: strconv.Itoa(retryLimitInMinutes),
		},
	}
}
