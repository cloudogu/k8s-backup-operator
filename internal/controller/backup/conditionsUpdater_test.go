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
}

func TestConditionsUpdaterDoesNotRecordNewOrUnchangedConditions(t *testing.T) {
	backup := newBackupForTest("ns", "backup")
	backup.Status.Conditions = []metav1.Condition{{
		Type:   backupv1.ConditionPrepared,
		Status: metav1.ConditionTrue,
	}}
	fakeClient := newFakeClientBuilder(t).WithStatusSubresource(backup).WithObjects(backup).Build()
	metrics.BackupConditionTransitionsTotal.Reset()

	err := newConditionsUpdater(fakeClient).updateStatus(context.Background(), backup, func(status *backupv1.BackupStatus) {
		status.Conditions[0].Reason = "StillPrepared"
		status.Conditions = append(status.Conditions, metav1.Condition{
			Type:   backupv1.ConditionSucceeded,
			Status: metav1.ConditionUnknown,
		})
	})

	require.NoError(t, err)
	assert.Equal(t, 0, testutil.CollectAndCount(metrics.BackupConditionTransitionsTotal))
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

	err := newConditionsUpdater(fakeClient).updateStatus(context.Background(), backup, func(status *backupv1.BackupStatus) {
		status.Conditions[0].Status = metav1.ConditionFalse
	})

	require.ErrorIs(t, err, assert.AnError)
	assert.Equal(t, 0, testutil.CollectAndCount(metrics.BackupConditionTransitionsTotal))
}
