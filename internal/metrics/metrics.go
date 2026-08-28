package metrics

import (
	v1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/prometheus/client_golang/prometheus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	BackupConditionTransitionsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "backup_condition_transitions_total",
			Help: "Number of persisted backup condition status transitions.",
		},
		[]string{
			"namespace",
			"name",
			"condition",
			"from",
			"to",
		},
	)

	RestoreStatusTransitionsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "restore_status_transitions_total",
			Help: "Number of restore status transitions labeled by 'to'.",
		},
		[]string{"namespace", "name", "to", "backup_name"},
	)
	RestoreConditionTransitionsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "restore_condition_transitions_total",
			Help: "Number of persisted restore condition status transitions.",
		},
		[]string{
			"namespace",
			"name",
			"backup_name",
			"condition",
			"from",
			"to",
		},
	)
	InvalidLeaseTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "backup_operator_invalid_lease_total",
			Help: "Number of invalid lease acquire attempts",
		},
		[]string{"namespace", "name"},
	)
)

// RegisterMetrics registers custom metrics with the global prometheus registry
func RegisterMetrics() {
	metrics.Registry.MustRegister(
		BackupReconcileTotal,
		BackupStatusTransitionsTotal,
		BackupConditionTransitionsTotal,
		RestoreReconcileTotal,
		RestoreStatusTransitionsTotal,
		RestoreConditionTransitionsTotal,
		InvalidLeaseTotal,
	)
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

// InitBackupConditionTransitionMetrics initializes every possible status transition for all backup conditions.
func InitBackupConditionTransitionMetrics(namespace, name string) {
	conditionTypes := []string{
		v1.ConditionPrepared,
		v1.ConditionDeleting,
		v1.ConditionCanceled,
		v1.ConditionSucceeded,
		v1.ConditionProviderSucceeded,
	}
	conditionStatuses := []metav1.ConditionStatus{
		metav1.ConditionUnknown,
		metav1.ConditionTrue,
		metav1.ConditionFalse,
	}

	for _, conditionType := range conditionTypes {
		for _, from := range conditionStatuses {
			for _, to := range conditionStatuses {
				if from == to {
					continue
				}
				BackupConditionTransitionsTotal.WithLabelValues(
					namespace,
					name,
					conditionType,
					string(from),
					string(to),
				).Add(0)
			}
		}
	}
}

// UpdateBackupConditionTransitionMetric increments the metric for a persisted backup condition status transition.
func UpdateBackupConditionTransitionMetric(namespace, name, conditionType, from, to string) {
	BackupConditionTransitionsTotal.WithLabelValues(namespace, name, conditionType, from, to).Inc()
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
	restoreStatuses := []string{v1.RestoreStatusNew, v1.RestoreStatusInProgress, v1.RestoreStatusCompleted, v1.RestoreStatusFailed, v1.RestoreStatusDeleting} //nolint:staticcheck // legacy restore status compatibility
	for _, status := range restoreStatuses {
		RestoreStatusTransitionsTotal.WithLabelValues(namespace, name, status, backupName).Add(0)
	}
}

// InitRestoreConditionTransitionMetrics initializes every possible status transition for all restore conditions.
func InitRestoreConditionTransitionMetrics(namespace, name, backupName string) {
	conditionTypes := []string{
		v1.ConditionSucceeded,
		v1.ConditionPrepared,
		v1.ConditionProviderSucceeded,
		v1.ConditionWorkloadsRecovered,
	}
	conditionStatuses := []metav1.ConditionStatus{
		metav1.ConditionUnknown,
		metav1.ConditionTrue,
		metav1.ConditionFalse,
	}

	for _, conditionType := range conditionTypes {
		for _, from := range conditionStatuses {
			for _, to := range conditionStatuses {
				if from == to {
					continue
				}

				RestoreConditionTransitionsTotal.WithLabelValues(
					namespace,
					name,
					backupName,
					conditionType,
					string(from),
					string(to),
				).Add(0)
			}
		}
	}
}

// UpdateRestoreConditionTransitionMetric increments the metric for a persisted restore condition status transition.
func UpdateRestoreConditionTransitionMetric(namespace, name, backupName, conditionType, from, to string) {
	RestoreConditionTransitionsTotal.WithLabelValues(namespace, name, backupName, conditionType, from, to).Inc()
}

// UpdateRestoreReconcileTotalMetric increments the metric for the total number of reconciles of the restore resource
func UpdateRestoreReconcileTotalMetric() {
	RestoreReconcileTotal.Inc()
}

// InitInvalidLeaseTotalMetric initializes the invalid lease metric for a resource without incrementing it.
func InitInvalidLeaseTotalMetric(namespace, name string) {
	InvalidLeaseTotal.WithLabelValues(namespace, name).Add(0)
}

// UpdateInvalidLeaseTotalMetric increments the metric for the total number of failed acquire lease attempts
func UpdateInvalidLeaseTotalMetric(namespace, name string) {
	InvalidLeaseTotal.WithLabelValues(namespace, name).Inc()
}
