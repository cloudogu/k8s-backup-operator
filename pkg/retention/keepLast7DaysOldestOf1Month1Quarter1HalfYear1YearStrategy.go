package retention

import "fmt"

type keepLast7DaysOldestOf1Month1Quarter1HalfYear1YearStrategy struct {
	Strategy
}

func newKeepLast7DaysOldestOf1Month1Quarter1HalfYear1YearStrategy() (Strategy, error) {
	calendar := newIntervalCalendar(KeepLast7DaysOldestOf1Month1Quarter1HalfYear1YearStrategy).
		addTimeIntervals([]timeInterval{
			newTimeInterval("sevenDays", 0, 7, keepAllIntervalMode),
			newTimeInterval("thirtyDays", 8, 30, keepOldestIntervalMode),
			newTimeInterval("ninetyDays", 31, 90, keepOldestIntervalMode),
			newTimeInterval("oneHundredEightyDays", 91, 180, keepOldestIntervalMode),
			newTimeInterval("threeHundredSixtyDays", 181, 360, keepOldestIntervalMode),
		})
	err := calendar.validateConfig()
	if err != nil {
		return nil, fmt.Errorf("interval calendar %q failed validation: %w", KeepLast7DaysOldestOf1Month1Quarter1HalfYear1YearStrategy, err)
	}

	clock := &clock{}
	rs := newIntervalBasedStrategy(KeepLast7DaysOldestOf1Month1Quarter1HalfYear1YearStrategy, calendar, clock)
	return &keepLast7DaysOldestOf1Month1Quarter1HalfYear1YearStrategy{rs}, nil
}
