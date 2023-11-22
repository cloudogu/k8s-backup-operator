package retention

import (
	"fmt"
)

// intervalCalendar stores the time intervals of an interval-based retention strategy and validates the configuration
type intervalCalendar struct {
	name          StrategyId
	timeIntervals []timeInterval
}

// invalidCalendarConfigError contains the message and intervalName of an invalid retention strategy
//
//nolint:unused
type invalidCalendarConfigError struct {
	message      string
	intervalName string
}

//nolint:unused
func (e *invalidCalendarConfigError) Error() string {
	return fmt.Sprintf("%s: Please check the interval %s", e.message, e.intervalName)
}

func newIntervalCalendar(name StrategyId) *intervalCalendar {
	return &intervalCalendar{name, nil}
}

func (ic intervalCalendar) addTimeIntervals(timeIntervals []timeInterval) intervalCalendar {
	ic.timeIntervals = append(ic.timeIntervals, timeIntervals...)

	return ic
}

//nolint:unused
func (ic intervalCalendar) validateConfig() error {
	var current = 0
	for _, interval := range ic.timeIntervals {
		if current != interval.start {
			return &invalidCalendarConfigError{"gaps or overlaps between interval borders are not allowed",
				interval.name}
		}
		current = interval.end + 1
	}
	return nil
}
