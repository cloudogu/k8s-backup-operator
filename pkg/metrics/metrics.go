package metrics

import (
	v1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	BackupReconcileTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "backup_reconcile_total",
		Help: "Total number of reconciles of the backup custom resource.",
	})

	RestoreReconcileTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "restore_reconcile_total",
		Help: "Total number of reconciles of the restore custom resource.",
	})

	BackupStatusTransitionsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "backup_status_transitions_total",
			Help: "Number of backup status transitions labeled by 'to'.",
		},
		[]string{"namespace", "name", "to"},
	)

	RestoreStatusTransitionsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "restore_status_transitions_total",
			Help: "Number of restore status transitions labeled by 'to'.",
		},
		[]string{"namespace", "name", "to", "backup_name"},
	)
)

// RegisterMetrics registers custom metrics with the global prometheus registry
func RegisterMetrics() {
	metrics.Registry.MustRegister(BackupReconcileTotal, BackupStatusTransitionsTotal, RestoreReconcileTotal, RestoreStatusTransitionsTotal)
}

// ### Backup ###

// UpdateBackupStatusMetrics updates the metrics for a backup resource with the new status
func UpdateBackupStatusMetrics(namespace, name, newStatus string) {
	// count transitions
	BackupStatusTransitionsTotal.WithLabelValues(namespace, name, newStatus).Inc()
}

// InitBackupStatusMetrics initializes the metrics for a backup resource
func InitBackupStatusMetrics(namespace, name string) {
	// all status values need to be initialized to 0 to monitor status increases
	backupStatuses := []string{v1.BackupStatusInProgress, v1.BackupStatusCompleted, v1.BackupStatusFailed, v1.BackupStatusDeleting}
	for _, status := range backupStatuses {
		BackupStatusTransitionsTotal.WithLabelValues(namespace, name, status).Add(0)
	}

	UpdateBackupStatusMetrics(namespace, name, v1.BackupStatusNew)
}

// UpdateBackupReconcileTotalMetric increments the metric for the total number of reconciles of the backup resource
func UpdateBackupReconcileTotalMetric() {
	BackupReconcileTotal.Inc()
}

// ### Restore ###

// UpdateRestoreStatusMetrics updates the metrics for a restore resource with the new status
func UpdateRestoreStatusMetrics(namespace, name, backupName, newStatus string) {
	// count transitions
	RestoreStatusTransitionsTotal.WithLabelValues(namespace, name, newStatus, backupName).Inc()
}

// InitRestoreStatusMetrics initializes the metrics for a restore resource
func InitRestoreStatusMetrics(namespace, name, backupName string) {
	// all status values need to be initialized to 0 to monitor status increases
	restoreStatuses := []string{v1.RestoreStatusInProgress, v1.RestoreStatusCompleted, v1.RestoreStatusFailed, v1.RestoreStatusDeleting}
	for _, status := range restoreStatuses {
		RestoreStatusTransitionsTotal.WithLabelValues(namespace, name, status, backupName).Add(0)
	}

	UpdateRestoreStatusMetrics(namespace, name, backupName, v1.RestoreStatusNew)
}

// UpdateRestoreReconcileTotalMetric increments the metric for the total number of reconciles of the restore resource
func UpdateRestoreReconcileTotalMetric() {
	RestoreReconcileTotal.Inc()
}
