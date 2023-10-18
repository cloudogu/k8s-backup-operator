package requeue

import (
	"fmt"
	"time"
)

type GenericRequeueableError struct {
	ErrMsg string
	Err    error
}

// Error returns the string representation of the wrapped error.
func (gre *GenericRequeueableError) Error() string {
	return fmt.Sprintf("%s: %s", gre.ErrMsg, gre.Err.Error())
}

// GetRequeueTime returns the time until the component should be requeued.
func (gre *GenericRequeueableError) GetRequeueTime(requeueTimeNanos time.Duration) time.Duration {
	return getRequeueTime(requeueTimeNanos)
}

func getRequeueTime(currentRequeueTime time.Duration) time.Duration {
	const initialRequeueTime = 15 * time.Second
	const linearCutoffThreshold6Hours = 6 * time.Hour

	if currentRequeueTime == 0 {
		return initialRequeueTime
	}

	nextRequeueTime := currentRequeueTime * 2

	if nextRequeueTime >= linearCutoffThreshold6Hours {
		return linearCutoffThreshold6Hours
	}

	return nextRequeueTime
}
