package backup

import (
	"context"
	"testing"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/k8s-backup-operator/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestConditionsUpdaterRecordsPersistedConditionStatusTransitions(t *testing.T) {
	backup := newBackupForTest("ns", "backup")
	backup.Status.Conditions = []metav1.Condition{{
		Type:   backupv1.ConditionPrepared,
		Status: metav1.ConditionUnknown,
	}}
	fakeClient := newFakeClientBuilder(t).WithStatusSubresource(backup).WithObjects(backup).Build()
	metrics.BackupConditionTransitionsTotal.Reset()
	metrics.BackupStatusTransitionsTotal.Reset()

	err := newConditionsUpdater(fakeClient).updateStatus(context.Background(), backup, func(status *backupv1.BackupStatus) {
		status.Conditions[0].Status = metav1.ConditionTrue
	})

	require.NoError(t, err)
	counter := metrics.BackupConditionTransitionsTotal.WithLabelValues(
		backup.Namespace,
		backup.Name,
		backupv1.ConditionPrepared,
		string(metav1.ConditionUnknown),
		string(metav1.ConditionTrue),
	)
	assert.Equal(t, 1.0, testutil.ToFloat64(counter))
	assert.Equal(t, 1.0, testutil.ToFloat64(metrics.BackupStatusTransitionsTotal.WithLabelValues(
		backup.Namespace, backup.Name, backupv1.BackupStatusInProgress,
	)))
}

func TestConditionsUpdaterDoesNotRecordNewOrUnchangedConditions(t *testing.T) {
	backup := newBackupForTest("ns", "backup")
	backup.Status.Conditions = []metav1.Condition{{
		Type:   backupv1.ConditionPrepared,
		Status: metav1.ConditionTrue,
	}}
	fakeClient := newFakeClientBuilder(t).WithStatusSubresource(backup).WithObjects(backup).Build()
	metrics.BackupConditionTransitionsTotal.Reset()
	metrics.BackupStatusTransitionsTotal.Reset()

	err := newConditionsUpdater(fakeClient).updateStatus(context.Background(), backup, func(status *backupv1.BackupStatus) {
		status.Conditions[0].Reason = "StillPrepared"
		status.Conditions = append(status.Conditions, metav1.Condition{
			Type:   backupv1.ConditionSucceeded,
			Status: metav1.ConditionUnknown,
		})
	})

	require.NoError(t, err)
	assert.Equal(t, 0, testutil.CollectAndCount(metrics.BackupConditionTransitionsTotal))
	assert.Equal(t, 1.0, testutil.ToFloat64(metrics.BackupStatusTransitionsTotal.WithLabelValues(
		backup.Namespace, backup.Name, backupv1.BackupStatusInProgress,
	)))
}

func TestConditionsUpdaterDoesNotRecordFailedPatches(t *testing.T) {
	backup := newBackupForTest("ns", "backup")
	backup.Status.Conditions = []metav1.Condition{{
		Type:   backupv1.ConditionPrepared,
		Status: metav1.ConditionUnknown,
	}}
	fakeClient := newFakeClientBuilder(t).
		WithStatusSubresource(backup).
		WithObjects(backup).
		WithInterceptorFuncs(interceptor.Funcs{
			SubResourcePatch: func(
				_ context.Context,
				_ client.Client,
				_ string,
				_ client.Object,
				_ client.Patch,
				_ ...client.SubResourcePatchOption,
			) error {
				return assert.AnError
			},
		}).
		Build()
	metrics.BackupConditionTransitionsTotal.Reset()
	metrics.BackupStatusTransitionsTotal.Reset()

	err := newConditionsUpdater(fakeClient).updateStatus(context.Background(), backup, func(status *backupv1.BackupStatus) {
		status.Conditions[0].Status = metav1.ConditionFalse
	})

	require.ErrorIs(t, err, assert.AnError)
	assert.Equal(t, 0, testutil.CollectAndCount(metrics.BackupConditionTransitionsTotal))
	assert.Equal(t, 0, testutil.CollectAndCount(metrics.BackupStatusTransitionsTotal))
}

func TestLegacyBackupStatusFor(t *testing.T) {
	metrics.BackupStatusTransitionsTotal.Reset()
	assert.Equal(t, 0, testutil.CollectAndCount(metrics.BackupStatusTransitionsTotal))
	tests := []struct {
		name       string
		conditions []metav1.Condition
		current    string
		expected   string
	}{
		{name: "preserves a status without conditions", current: backupv1.BackupStatusNew, expected: backupv1.BackupStatusNew},
		{name: "maps a non-terminal workflow condition to in progress", conditions: []metav1.Condition{{Type: backupv1.ConditionPrepared, Status: metav1.ConditionFalse}}, expected: backupv1.BackupStatusInProgress},
		{name: "maps running provider backup to in progress", conditions: []metav1.Condition{{Type: backupv1.ConditionSucceeded, Status: metav1.ConditionUnknown}}, expected: backupv1.BackupStatusInProgress},
		{name: "maps successful provider backup to completed", conditions: []metav1.Condition{{Type: backupv1.ConditionSucceeded, Status: metav1.ConditionTrue}}, expected: backupv1.BackupStatusCompleted},
		{name: "maps failed provider backup to failed", conditions: []metav1.Condition{{Type: backupv1.ConditionSucceeded, Status: metav1.ConditionFalse}}, expected: backupv1.BackupStatusFailed},
		{name: "maps canceled backup to failed", conditions: []metav1.Condition{{Type: backupv1.ConditionCanceled, Status: metav1.ConditionTrue}}, expected: backupv1.BackupStatusFailed},
		{name: "maps deleting backup to deleting with highest priority", conditions: []metav1.Condition{{Type: backupv1.ConditionDeleting, Status: metav1.ConditionTrue}, {Type: backupv1.ConditionSucceeded, Status: metav1.ConditionTrue}}, expected: backupv1.BackupStatusDeleting},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backup := newBackupForTest("ns", "backup")
			backup.Status.Status = tt.current
			backup.Status.Conditions = tt.conditions

			assert.Equal(t, tt.expected, legacyBackupStatusFor(backup))
		})
	}
}

func TestConditionsUpdaterPersistsAndRecordsLegacyStatusTransitions(t *testing.T) {
	backup := newBackupForTest("ns", "backup")
	fakeClient := newFakeClientBuilder(t).WithStatusSubresource(backup).WithObjects(backup).Build()
	metrics.BackupStatusTransitionsTotal.Reset()
	updater := newConditionsUpdater(fakeClient)

	err := updater.updateStatus(context.Background(), backup, func(status *backupv1.BackupStatus) {
		status.Conditions = append(status.Conditions, metav1.Condition{
			Type:   backupv1.ConditionSucceeded,
			Status: metav1.ConditionUnknown,
		})
	})
	require.NoError(t, err)
	assert.Equal(t, backupv1.BackupStatusInProgress, backup.Status.Status)

	err = updater.updateStatus(context.Background(), backup, func(status *backupv1.BackupStatus) {
		status.Conditions[0].Status = metav1.ConditionTrue
	})
	require.NoError(t, err)
	assert.Equal(t, backupv1.BackupStatusCompleted, backup.Status.Status)

	stored := &backupv1.Backup{}
	require.NoError(t, fakeClient.Get(context.Background(), client.ObjectKeyFromObject(backup), stored))
	assert.Equal(t, backupv1.BackupStatusCompleted, stored.Status.Status)

	assert.Equal(t, 1.0, testutil.ToFloat64(metrics.BackupStatusTransitionsTotal.WithLabelValues(
		backup.Namespace, backup.Name, backupv1.BackupStatusInProgress,
	)))
	assert.Equal(t, 1.0, testutil.ToFloat64(metrics.BackupStatusTransitionsTotal.WithLabelValues(
		backup.Namespace, backup.Name, backupv1.BackupStatusCompleted,
	)))
}
