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

// backupStatuses possible states for backup resources
var backupStatuses = []string{v1.BackupStatusInProgress, v1.BackupStatusCompleted, v1.BackupStatusFailed, v1.BackupStatusDeleting}

// RegisterMetrics registers custom metrics with the global prometheus registry
func RegisterMetrics() {
	metrics.Registry.MustRegister(BackupReconcileTotal, BackupStatusTransitionsTotal, RestoreReconcileTotal, RestoreStatusTransitionsTotal)
}

// ### Backup ###

func UpdateBackupStatusMetrics(namespace, name, newStatus string) {
	// count transitions
	BackupStatusTransitionsTotal.WithLabelValues(namespace, name, newStatus).Inc()
}

// InitBackupStatusMetrics initializes the metrics for a backup resource
func InitBackupStatusMetrics(namespace, name string) {
	// all status values need to be initialized to 0 to monitor status increases
	for _, status := range backupStatuses {
		BackupStatusTransitionsTotal.WithLabelValues(namespace, name, status).Add(0)
	}

	UpdateBackupStatusMetrics(namespace, name, v1.BackupStatusNew)
}

func UpdateBackupReconcileTotalMetric() {
	BackupReconcileTotal.Inc()
}

// ### Restore ###

func UpdateRestoreStatusMetrics(namespace, name, backupName, newStatus string) {
	// count transitions
	RestoreStatusTransitionsTotal.WithLabelValues(namespace, name, newStatus).Inc()
}

// InitRestoreStatusMetrics initializes the metrics for a restore resource
func InitRestoreStatusMetrics(namespace, name, backupName string) {
	// all status values need to be initialized to 0 to monitor status increases
	for _, status := range backupStatuses {
		RestoreStatusTransitionsTotal.WithLabelValues(namespace, name, status).Add(0)
	}

	UpdateRestoreStatusMetrics(namespace, name, v1.BackupStatusNew, backupName)
}

func UpdateRestoreReconcileTotalMetric() {
	RestoreReconcileTotal.Inc()
}
