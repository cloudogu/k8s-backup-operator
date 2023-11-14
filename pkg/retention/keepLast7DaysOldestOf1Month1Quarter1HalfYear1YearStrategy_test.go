package retention

import (
	"github.com/stretchr/testify/require"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigKeep7Days1Month1Quarter1YearRetentionStrategy(t *testing.T) {
	rs := newKeepLast7DaysOldestOf1Month1Quarter1HalfYear1YearStrategy()

	assert.Equal(t, rs.GetName(), KeepLast7DaysOldestOf1Month1Quarter1HalfYear1YearStrategy)

	require.IsType(t, &intervalBasedStrategy{}, rs)
	intervalStrat := rs.(*intervalBasedStrategy)
	// we can validate the config here, no need to do this in runtime
	err := intervalStrat.intervalCalendar.validateConfig()
	assert.NoError(t, err)
}
