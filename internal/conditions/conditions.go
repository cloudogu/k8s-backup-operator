package conditions

import (
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// ReasonPending marks a milestone that has been declared but not observed yet.
	ReasonPending = "Pending"
)

// Missing returns the desired conditions the resource does not carry yet, so it sets only the missing ones.
func Missing(existing []metav1.Condition, desired []metav1.Condition) []metav1.Condition {
	var missing []metav1.Condition

	for _, condition := range desired {
		if meta.FindStatusCondition(existing, condition.Type) != nil {
			continue
		}

		missing = append(missing, condition)
	}

	return missing
}

// WillChange reports whether persisting desired would change the stored condition's status,
// reason or message. The reconciler replays every stage on each reconciliation, so this is the guard
// that turns a per-reconciliation observation into one time actions, for example, a log line per reached state.
func WillChange(conditions []metav1.Condition, desired metav1.Condition) bool {
	current := meta.FindStatusCondition(conditions, desired.Type)
	if current == nil {
		return true
	}

	return current.Status != desired.Status ||
		current.Reason != desired.Reason ||
		current.Message != desired.Message
}

// ElapsedInCurrentStatus reports how long the condition has been in its current status. It returns
// zero for a condition that does not exist yet. Only a status change advances LastTransitionTime, so
// a reason or message update does not restart the measurement.
func ElapsedInCurrentStatus(conditions []metav1.Condition, conditionType string, now time.Time) time.Duration {
	condition := meta.FindStatusCondition(conditions, conditionType)
	if condition == nil || condition.LastTransitionTime.IsZero() {
		return 0
	}

	elapsed := now.Sub(condition.LastTransitionTime.Time)
	if elapsed < 0 {
		return 0
	}
	return elapsed
}

// FormatWaitDuration renders a wait truncated to whole minutes.
func FormatWaitDuration(elapsed time.Duration) string {
	minutes := int(elapsed.Truncate(time.Minute).Minutes())
	switch {
	case minutes < 1:
		return "less than 1m"
	case minutes < 60:
		return fmt.Sprintf("%dm", minutes)
	default:
		return fmt.Sprintf("%dh%dm", minutes/60, minutes%60)
	}
}
