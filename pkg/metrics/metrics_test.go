package metrics

import (
	"testing"

	v1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
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

		valNew := testutil.ToFloat64(RestoreStatusTransitionsTotal.WithLabelValues(namespace, name, v1.RestoreStatusNew, backupName))
		assert.Equal(t, 1.0, valNew, "expected status '%s' to be 1", v1.RestoreStatusNew)

		expectedZeroStatuses := []string{v1.RestoreStatusInProgress, v1.RestoreStatusCompleted, v1.RestoreStatusFailed, v1.RestoreStatusDeleting}
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

func TestMetricsVariables(t *testing.T) {
	t.Run("should have defined metric variables", func(t *testing.T) {
		assert.NotNil(t, BackupReconcileTotal)
		assert.NotNil(t, BackupStatusTransitionsTotal)
		assert.NotNil(t, RestoreReconcileTotal)
		assert.NotNil(t, RestoreStatusTransitionsTotal)
	})

	t.Run("should be registrable in a new registry", func(t *testing.T) {
		registry := prometheus.NewRegistry()

		err := registry.Register(BackupReconcileTotal)

		assert.NoError(t, err)
	})
}
