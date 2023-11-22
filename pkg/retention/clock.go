package retention

import "time"

type clock struct{}

func (c *clock) Now() time.Time {
	return time.Now()
}
