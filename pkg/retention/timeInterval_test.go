package retention

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRetentionModeValidation(t *testing.T) {
	// valid retention modes
	assert.True(t, validateRetentionMode("ALL"))
	assert.True(t, validateRetentionMode(keepAllIntervalMode))
	assert.True(t, validateRetentionMode("OLDEST"))
	assert.True(t, validateRetentionMode(keepOldestIntervalMode))
	// invalid retention modes
	assert.False(t, validateRetentionMode("IM_NOT_A_VALID_RETENTION_MODE"))
	assert.False(t, validateRetentionMode("all"))
	assert.False(t, validateRetentionMode("oldest"))
}

func TestRetentionPolicyIntervalCalculation(t *testing.T) {
	var testTime = time.Date(2019, 03, 01, 1, 1, 0, 0, time.UTC)
	var sevenDaysInterval = newTimeInterval("sevenDays", 0, 7, "ALL")
	var thirtyDaysInterval = newTimeInterval("thirtyDays", 8, 30, "ALL")
	var ninetyDaysInterval = newTimeInterval("ninetyDays", 31, 90, "ALL")

	var intervalChecks = []struct {
		interval     timeInterval
		timestamp    time.Time
		isInInterval bool
	}{
		{sevenDaysInterval, testTime.AddDate(0, 0, 1), false},
		{sevenDaysInterval, testTime.AddDate(0, 0, 0), true},
		{sevenDaysInterval, testTime.AddDate(0, 0, -1), true},
		{sevenDaysInterval, testTime.AddDate(0, 0, -7), true},
		{sevenDaysInterval, testTime.AddDate(0, 0, -7).Add(-time.Hour * 2), false},
		{sevenDaysInterval, testTime.AddDate(0, 0, -8), false},

		{thirtyDaysInterval, testTime.AddDate(0, 0, 0), false},
		{thirtyDaysInterval, testTime.AddDate(0, 0, -7), false},
		{thirtyDaysInterval, testTime.AddDate(0, 0, -8), true},
		{thirtyDaysInterval, testTime.AddDate(0, 0, -25), true},
		{thirtyDaysInterval, testTime.AddDate(0, 0, -30), true},
		{thirtyDaysInterval, testTime.AddDate(0, 0, -31), false},

		{ninetyDaysInterval, testTime.AddDate(0, 0, 0), false},
		{ninetyDaysInterval, testTime.AddDate(0, 0, -29), false},
		{ninetyDaysInterval, testTime.AddDate(0, 0, -30), false},
		{ninetyDaysInterval, testTime.AddDate(0, 0, -45), true},
		{ninetyDaysInterval, testTime.AddDate(0, 0, -90), true},
		{ninetyDaysInterval, testTime.AddDate(0, 0, -91), false},
	}

	for _, ic := range intervalChecks {
		t.Run("TestIsTimestampInInterval", func(t *testing.T) {
			actual := ic.interval.isTimestampInInterval(ic.timestamp, testTime)
			expected := ic.isInInterval
			assert.Equal(t, expected, actual)
		})

	}
}
