package conditions

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestWillChange(t *testing.T) {
	desired := metav1.Condition{Type: "Prepared", Status: metav1.ConditionTrue, Reason: "Available", Message: "available"}

	t.Run("reports a change for a condition that does not exist yet", func(t *testing.T) {
		assert.True(t, WillChange(nil, desired))
	})

	t.Run("reports no change for an identical condition", func(t *testing.T) {
		assert.False(t, WillChange([]metav1.Condition{desired}, desired))
	})

	t.Run("reports a change for a differing status, reason or message", func(t *testing.T) {
		for name, stored := range map[string]metav1.Condition{
			"status":  {Type: "Prepared", Status: metav1.ConditionFalse, Reason: "Available", Message: "available"},
			"reason":  {Type: "Prepared", Status: metav1.ConditionTrue, Reason: "NotFound", Message: "available"},
			"message": {Type: "Prepared", Status: metav1.ConditionTrue, Reason: "Available", Message: "something else"},
		} {
			t.Run(name, func(t *testing.T) {
				assert.True(t, WillChange([]metav1.Condition{stored}, desired))
			})
		}
	})

	t.Run("ignores conditions of another type", func(t *testing.T) {
		other := metav1.Condition{Type: "Succeeded", Status: metav1.ConditionTrue, Reason: "Available", Message: "available"}
		assert.True(t, WillChange([]metav1.Condition{other}, desired))
	})
}

func TestElapsedInCurrentStatus(t *testing.T) {
	now := time.Date(2026, 8, 21, 5, 10, 0, 0, time.UTC)

	t.Run("returns the time since the last transition", func(t *testing.T) {
		conditions := []metav1.Condition{{
			Type:               "Succeeded",
			LastTransitionTime: metav1.NewTime(now.Add(-3 * time.Minute)),
		}}

		assert.Equal(t, 3*time.Minute, ElapsedInCurrentStatus(conditions, "Succeeded", now))
	})

	t.Run("returns zero for a missing condition, a missing timestamp and a future timestamp", func(t *testing.T) {
		missingTimestamp := []metav1.Condition{{Type: "Succeeded"}}
		future := []metav1.Condition{{Type: "Succeeded", LastTransitionTime: metav1.NewTime(now.Add(time.Minute))}}

		assert.Zero(t, ElapsedInCurrentStatus(nil, "Succeeded", now))
		assert.Zero(t, ElapsedInCurrentStatus(missingTimestamp, "Succeeded", now))
		assert.Zero(t, ElapsedInCurrentStatus(future, "Succeeded", now))
	})
}

func TestFormatWaitDuration(t *testing.T) {
	for expected, elapsed := range map[string]time.Duration{
		"less than 1m": 59 * time.Second,
		"1m":           90 * time.Second,
		"59m":          59 * time.Minute,
		"1h0m":         time.Hour,
		"2h5m":         2*time.Hour + 5*time.Minute,
	} {
		assert.Equal(t, expected, FormatWaitDuration(elapsed))
	}
}
