package retention

func newKeepLast7DaysOldestOf1Month1Quarter1HalfYear1YearStrategy() Strategy {
	calendar := newIntervalCalendar(KeepLast7DaysOldestOf1Month1Quarter1HalfYear1YearStrategy).
		addTimeIntervals([]timeInterval{
			newTimeInterval("sevenDays", 0, 7, keepAllIntervalMode),
			newTimeInterval("thirtyDays", 8, 30, keepOldestIntervalMode),
			newTimeInterval("ninetyDays", 31, 90, keepOldestIntervalMode),
			newTimeInterval("oneHundredEightyDays", 91, 180, keepOldestIntervalMode),
			newTimeInterval("threeHundredSixtyDays", 181, 360, keepOldestIntervalMode),
		})

	clock := &clock{}
	rs := newIntervalBasedStrategy(KeepLast7DaysOldestOf1Month1Quarter1HalfYear1YearStrategy, calendar, clock)
	return rs
}
