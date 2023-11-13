package retention

import (
	"time"
)

/*
timeInterval defines an interval in days
e.g., timeInterval{"sevenDaysInterval", 0, 7, "ALL"}
retentionMode specifies the saving behavior currently supported values are "ALL" to keep all within the interval
or "OLDEST" to keep the oldest backup within the interval
*/
type timeInterval struct {
	name          string
	start         int
	end           int
	retentionMode string
}

const (
	keepAllIntervalMode    = "ALL"
	keepOldestIntervalMode = "OLDEST"
)

/*
Add more retention modes here. The resulting behaviour should be added in customRetentionPolicy.go
the first element is the default used in newTimeInterval
*/
var validRetentionModes = []string{keepAllIntervalMode, keepOldestIntervalMode}

func validateRetentionMode(retentionMode string) bool {
	for _, validRetentionMode := range validRetentionModes {
		if retentionMode == validRetentionMode {
			return true
		}
	}
	return false
}

/*
newTimeInterval creates a time interval
params:

	    name = the name of the interval e.g., KeepLastSevenDaysStrategy
		start = the start of the interval (interpreted as days from now)
		end = the end of the interval (interpreted as days from now) should be greater than start
		retentionMode = defines the policy of the interval
		  supports: "ALL" = keep all Backup within the interval
		            "OLDEST" = keep the oldest Backup within the interval
		  fallback: When not supported or wrong, the first entry of validRetentionModes is used (ALL)
*/
func newTimeInterval(name string, start int, end int, retentionMode string) timeInterval {
	if validateRetentionMode(retentionMode) {
		return timeInterval{name, start, end, retentionMode}
	}
	return timeInterval{name, start, end, validRetentionModes[0]}
}

// isTimestampInInterval checks if a timestamp is inside a give interval
func (interval timeInterval) isTimestampInInterval(timestamp time.Time, current time.Time) bool {
	// let's travel back in time ✈
	// The intervalStart is defined as the currentTime - interval.start e.g., interval.start=7 => intervalStart = current-7
	day := time.Hour * 24
	intervalStart := current.Add(day * time.Duration(-interval.start)).Truncate(day)
	// intervalEnd is defined as the currentTime - interval.end e.g., interval.end=30 => intervalEnd = current-30
	intervalEnd := current.Add(day * time.Duration(-interval.end)).Truncate(day)

	// make date comparisons without involving the exact time => time of date is ignored
	timeStampFormatted := timestamp.Truncate(day)

	/* sample to understand the functionality
	 * Backup was started 12.05.2019 at 9o'clock => timestamp 12.05.2019:09:00:00
	 * today is the 17.05.2019; the interval is defined from 0 to 7 (17.05.2019-10.05.2019)
	 * because the 12.05 is between 10.05. and 17.05. the function returns true
	 */
	return (timeStampFormatted.After(intervalEnd) || timeStampFormatted.Equal(intervalEnd)) &&
		(timeStampFormatted.Before(intervalStart) || timeStampFormatted.Equal(intervalStart))
}
