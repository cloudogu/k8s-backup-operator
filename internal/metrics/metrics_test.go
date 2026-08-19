package metrics

import (
	"testing"

	v1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestUpdateBackupReconcileTotalMetric(t *testing.T) {
	t.Run("should increment backup reconcile total metric", func(t *testing.T) {
		initial := testutil.ToFloat64(BackupReconcileTotal)

		UpdateBackupReconcileTotalMetric()

		current := testutil.ToFloat64(BackupReconcileTotal)
		assert.Equal(t, initial+1, current)
	})
}

func TestInitBackupStatusMetrics(t *testing.T) {
	t.Run("should initialize backup status metrics correctly", func(t *testing.T) {
		namespace := "test-ns"
		name := "test-backup"
		BackupStatusTransitionsTotal.Reset()

		InitBackupStatusMetrics(namespace, name)

		valNew := testutil.ToFloat64(BackupStatusTransitionsTotal.WithLabelValues(namespace, name, v1.BackupStatusNew))
		assert.Equal(t, 1.0, valNew, "expected status '%s' to be 1", v1.BackupStatusNew)

		expectedZeroStatuses := []string{v1.BackupStatusInProgress, v1.BackupStatusCompleted, v1.BackupStatusFailed, v1.BackupStatusDeleting}
		for _, status := range expectedZeroStatuses {
			val := testutil.ToFloat64(BackupStatusTransitionsTotal.WithLabelValues(namespace, name, status))
			assert.Equal(t, 0.0, val, "expected status '%s' to be initialized to 0", status)
		}
	})
}

func TestUpdateBackupStatusMetrics(t *testing.T) {
	t.Run("should increment specific backup status metric", func(t *testing.T) {
		namespace := "test-ns"
		name := "test-backup-update"
		status := v1.BackupStatusFailed

		counter := BackupStatusTransitionsTotal.WithLabelValues(namespace, name, status)
		initial := testutil.ToFloat64(counter)

		UpdateBackupStatusMetrics(namespace, name, status)

		current := testutil.ToFloat64(counter)
		assert.Equal(t, initial+1, current)
	})
}

func TestInitBackupConditionTransitionMetrics(t *testing.T) {
	namespace := "test-ns"
	name := "test-backup-conditions"
	BackupConditionTransitionsTotal.Reset()

	InitBackupConditionTransitionMetrics(namespace, name)
	// call it twice to check if no duplicated rows were created
	InitBackupConditionTransitionMetrics(namespace, name)

	conditionTypes := []string{
		v1.ConditionPrepared,
		v1.ConditionDeleting,
		v1.ConditionCanceled,
		v1.ConditionSucceeded,
	}
	conditionStatuses := []metav1.ConditionStatus{
		metav1.ConditionUnknown,
		metav1.ConditionTrue,
		metav1.ConditionFalse,
	}

	type conditionTransition struct {
		condition string
		from      string
		to        string
	}
	expectedTransitions := map[conditionTransition]struct{}{}
	for _, conditionType := range conditionTypes {
		for _, from := range conditionStatuses {
			for _, to := range conditionStatuses {
				if from == to {
					continue
				}
				expectedTransitions[conditionTransition{condition: conditionType, from: string(from), to: string(to)}] = struct{}{}
			}
		}
	}

	metrics := make(chan prometheus.Metric)
	go func() {
		BackupConditionTransitionsTotal.Collect(metrics)
		close(metrics)
	}()
	for metric := range metrics {
		metricDTO := &dto.Metric{}
		assert.NoError(t, metric.Write(metricDTO))

		labels := make(map[string]string, len(metricDTO.Label))
		for _, label := range metricDTO.Label {
			labels[label.GetName()] = label.GetValue()
		}
		assert.Len(t, labels, 5)
		assert.Equal(t, namespace, labels["namespace"])
		assert.Equal(t, name, labels["name"])
		assert.Zero(t, metricDTO.GetCounter().GetValue())

		transition := conditionTransition{
			condition: labels["condition"],
			from:      labels["from"],
			to:        labels["to"],
		}
		assert.Contains(t, expectedTransitions, transition)
		delete(expectedTransitions, transition)
	}
	assert.Empty(t, expectedTransitions)
}

func TestUpdateBackupConditionTransitionMetric(t *testing.T) {
	namespace := "test-ns"
	name := "test-backup-condition-update"
	conditionType := v1.ConditionPrepared
	from := string(metav1.ConditionUnknown)
	to := string(metav1.ConditionTrue)
	BackupConditionTransitionsTotal.Reset()

	UpdateBackupConditionTransitionMetric(namespace, name, conditionType, from, to)

	counter := BackupConditionTransitionsTotal.WithLabelValues(namespace, name, conditionType, from, to)
	assert.Equal(t, 1.0, testutil.ToFloat64(counter))
}
func TestUpdateRestoreReconcileTotalMetric(t *testing.T) {
	t.Run("should increment restore reconcile total metric", func(t *testing.T) {
		initial := testutil.ToFloat64(RestoreReconcileTotal)

		UpdateRestoreReconcileTotalMetric()

		current := testutil.ToFloat64(RestoreReconcileTotal)
		assert.Equal(t, initial+1, current)
	})
}

func TestInitRestoreStatusMetrics(t *testing.T) {
	t.Run("should initialize restore status metrics correctly", func(t *testing.T) {
		namespace := "test-ns"
		name := "test-restore"
		backupName := "source-backup"
		RestoreStatusTransitionsTotal.Reset()

		InitRestoreStatusMetrics(namespace, name, backupName)
		InitRestoreStatusMetrics(namespace, name, backupName)

		assert.Equal(t, 5, testutil.CollectAndCount(RestoreStatusTransitionsTotal))
		expectedZeroStatuses := []string{v1.RestoreStatusNew, v1.RestoreStatusInProgress, v1.RestoreStatusCompleted, v1.RestoreStatusFailed, v1.RestoreStatusDeleting}
		for _, status := range expectedZeroStatuses {
			val := testutil.ToFloat64(RestoreStatusTransitionsTotal.WithLabelValues(namespace, name, status, backupName))
			assert.Equal(t, 0.0, val, "expected status '%s' to be initialized to 0", status)
		}
	})
}

func TestUpdateRestoreStatusMetrics(t *testing.T) {
	t.Run("should increment specific restore status metric", func(t *testing.T) {
		namespace := "test-ns"
		name := "test-restore-update"
		backupName := "source-backup"
		status := v1.RestoreStatusCompleted

		counter := RestoreStatusTransitionsTotal.WithLabelValues(namespace, name, status, backupName)
		initial := testutil.ToFloat64(counter)

		UpdateRestoreStatusMetrics(namespace, name, backupName, status)

		current := testutil.ToFloat64(counter)
		assert.Equal(t, initial+1, current)
	})
}

func TestInitRestoreConditionTransitionMetrics(t *testing.T) {
	namespace := "test-ns"
	name := "test-restore-conditions"
	backupName := "source-backup"
	RestoreConditionTransitionsTotal.Reset()

	InitRestoreConditionTransitionMetrics(namespace, name, backupName)
	InitRestoreConditionTransitionMetrics(namespace, name, backupName)

	conditionTypes := []string{
		v1.ConditionSuccessful,
		v1.ConditionPrepared,
		v1.ConditionProviderRestoreSuccessful,
		v1.ConditionWorkloadsRecovered,
		v1.ConditionBackupsSynchronized,
	}
	conditionStatuses := []metav1.ConditionStatus{
		metav1.ConditionUnknown,
		metav1.ConditionTrue,
		metav1.ConditionFalse,
	}

	type conditionTransition struct {
		condition string
		from      string
		to        string
	}
	expectedTransitions := map[conditionTransition]struct{}{}
	for _, conditionType := range conditionTypes {
		for _, from := range conditionStatuses {
			for _, to := range conditionStatuses {
				if from == to {
					continue
				}
				expectedTransitions[conditionTransition{condition: conditionType, from: string(from), to: string(to)}] = struct{}{}
			}
		}
	}

	metrics := make(chan prometheus.Metric)
	go func() {
		RestoreConditionTransitionsTotal.Collect(metrics)
		close(metrics)
	}()
	for metric := range metrics {
		metricDTO := &dto.Metric{}
		assert.NoError(t, metric.Write(metricDTO))

		labels := make(map[string]string, len(metricDTO.Label))
		for _, label := range metricDTO.Label {
			labels[label.GetName()] = label.GetValue()
		}
		assert.Len(t, labels, 6)
		assert.Equal(t, namespace, labels["namespace"])
		assert.Equal(t, name, labels["name"])
		assert.Equal(t, backupName, labels["backup_name"])
		assert.Zero(t, metricDTO.GetCounter().GetValue())

		transition := conditionTransition{
			condition: labels["condition"],
			from:      labels["from"],
			to:        labels["to"],
		}
		assert.Contains(t, expectedTransitions, transition)
		delete(expectedTransitions, transition)
	}
	assert.Empty(t, expectedTransitions)
}

func TestUpdateRestoreConditionTransitionMetric(t *testing.T) {
	namespace := "test-ns"
	name := "test-restore-condition-update"
	backupName := "source-backup"
	conditionType := v1.ConditionPrepared
	from := string(metav1.ConditionUnknown)
	to := string(metav1.ConditionTrue)
	RestoreConditionTransitionsTotal.Reset()

	UpdateRestoreConditionTransitionMetric(namespace, name, backupName, conditionType, from, to)

	counter := RestoreConditionTransitionsTotal.WithLabelValues(namespace, name, backupName, conditionType, from, to)
	assert.Equal(t, 1.0, testutil.ToFloat64(counter))
}
