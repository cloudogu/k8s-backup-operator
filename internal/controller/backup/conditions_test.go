package backup

import (
	"context"
	"testing"
	"time"

	"github.com/cloudogu/k8s-backup-operator/internal/logging"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// logRecorder captures the messages of a logger split by level so tests can assert that a stage
// reports itself exactly once instead of once per reconciliation.
type logRecorder struct {
	infos  []string
	debugs []string
}

func (*logRecorder) Init(logr.RuntimeInfo) {}

func (*logRecorder) Enabled(int) bool { return true }

func (recorder *logRecorder) Info(level int, message string, _ ...any) {
	if level == 0 {
		recorder.infos = append(recorder.infos, message)
		return
	}
	recorder.debugs = append(recorder.debugs, message)
}

func (*logRecorder) Error(error, string, ...any) {}

func (recorder *logRecorder) WithValues(...any) logr.LogSink { return recorder }

func (recorder *logRecorder) WithName(string) logr.LogSink { return recorder }

// newRecordingContext returns a context whose logger records every message it receives.
func newRecordingContext() (context.Context, *logRecorder) {
	recorder := &logRecorder{}
	return logging.IntoContext(context.Background(), logr.New(recorder)), recorder
}

func TestConditionWillChange(t *testing.T) {
	desired := metav1.Condition{Type: "Prepared", Status: metav1.ConditionTrue, Reason: "Available", Message: "available"}

	t.Run("reports a change for a condition that does not exist yet", func(t *testing.T) {
		assert.True(t, conditionWillChange(nil, desired))
	})

	t.Run("reports no change for an identical condition", func(t *testing.T) {
		assert.False(t, conditionWillChange([]metav1.Condition{desired}, desired))
	})

	t.Run("reports a change for a differing status, reason or message", func(t *testing.T) {
		for name, stored := range map[string]metav1.Condition{
			"status":  {Type: "Prepared", Status: metav1.ConditionFalse, Reason: "Available", Message: "available"},
			"reason":  {Type: "Prepared", Status: metav1.ConditionTrue, Reason: "NotFound", Message: "available"},
			"message": {Type: "Prepared", Status: metav1.ConditionTrue, Reason: "Available", Message: "something else"},
		} {
			t.Run(name, func(t *testing.T) {
				assert.True(t, conditionWillChange([]metav1.Condition{stored}, desired))
			})
		}
	})

	t.Run("ignores conditions of another type", func(t *testing.T) {
		other := metav1.Condition{Type: "Succeeded", Status: metav1.ConditionTrue, Reason: "Available", Message: "available"}
		assert.True(t, conditionWillChange([]metav1.Condition{other}, desired))
	})
}

func TestElapsedInCurrentStatus(t *testing.T) {
	now := time.Date(2026, 8, 21, 5, 10, 0, 0, time.UTC)

	t.Run("returns the time since the last transition", func(t *testing.T) {
		conditions := []metav1.Condition{{
			Type:               "Succeeded",
			LastTransitionTime: metav1.NewTime(now.Add(-3 * time.Minute)),
		}}

		assert.Equal(t, 3*time.Minute, elapsedInCurrentStatus(conditions, "Succeeded", now))
	})

	t.Run("returns zero for a missing condition, a missing timestamp and a future timestamp", func(t *testing.T) {
		missingTimestamp := []metav1.Condition{{Type: "Succeeded"}}
		future := []metav1.Condition{{Type: "Succeeded", LastTransitionTime: metav1.NewTime(now.Add(time.Minute))}}

		assert.Zero(t, elapsedInCurrentStatus(nil, "Succeeded", now))
		assert.Zero(t, elapsedInCurrentStatus(missingTimestamp, "Succeeded", now))
		assert.Zero(t, elapsedInCurrentStatus(future, "Succeeded", now))
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
		assert.Equal(t, expected, formatWaitDuration(elapsed))
	}
}
