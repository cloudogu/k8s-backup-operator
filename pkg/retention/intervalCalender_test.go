package retention

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestIntervalCalendarValidConfig(t *testing.T) {

	var (
		keep7Days1Month1Quarter1Year = newIntervalCalendar("testCalendar").addTimeIntervals([]timeInterval{
			newTimeInterval("sevenDays", 0, 7, keepAllIntervalMode),
			newTimeInterval("thirtyDays", 8, 30, keepOldestIntervalMode),
			newTimeInterval("ninetyDays", 31, 90, keepOldestIntervalMode),
			newTimeInterval("oneHundredEightyDays", 91, 180, keepOldestIntervalMode),
			newTimeInterval("threeHundredSixtyDays", 181, 360, keepOldestIntervalMode),
		})
	)
	err := keep7Days1Month1Quarter1Year.validateConfig()
	assert.Nil(t, err)

}

func TestIntervalCalendarInvalidConfig(t *testing.T) {

	var (
		testCalendar = newIntervalCalendar("testCalendar").addTimeIntervals([]timeInterval{
			newTimeInterval("A", 0, 7, keepAllIntervalMode),
			newTimeInterval("B", 14, 16, keepOldestIntervalMode),
		})
	)
	err := testCalendar.validateConfig()

	require.Error(t, err)
	require.IsType(t, &invalidCalendarConfigError{}, err)
	assert.Equal(t, err.(*invalidCalendarConfigError).intervalName, "B")
	assert.Equal(t, err.Error(), "gaps or overlaps between interval borders are not allowed: Please check the interval B")

	var (
		testCalendar2 = newIntervalCalendar("testCalendar2").addTimeIntervals([]timeInterval{
			newTimeInterval("A", 0, 7, keepAllIntervalMode),
			newTimeInterval("B", 8, 16, keepOldestIntervalMode),
			newTimeInterval("C", 16, 17, keepOldestIntervalMode),
		})
	)
	err = testCalendar2.validateConfig()

	require.Error(t, err)
	require.IsType(t, &invalidCalendarConfigError{}, err)
	assert.Equal(t, err.(*invalidCalendarConfigError).intervalName, "C")
	assert.Equal(t, err.Error(), "gaps or overlaps between interval borders are not allowed: Please check the interval C")
}

func TestIntervalCalendarAddIntervals(t *testing.T) {

	var (
		testCalendar = newIntervalCalendar("testCalendar").addTimeIntervals([]timeInterval{
			newTimeInterval("A", 0, 7, keepAllIntervalMode),
			newTimeInterval("B", 14, 16, keepOldestIntervalMode),
		})
	)

	assert.ElementsMatch(t, testCalendar.timeIntervals, []timeInterval{
		newTimeInterval("A", 0, 7, keepAllIntervalMode),
		newTimeInterval("B", 14, 16, keepOldestIntervalMode),
	})

	assert.Equal(t, StrategyId("testCalendar"), testCalendar.name)
}
