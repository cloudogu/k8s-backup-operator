package retry

import (
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	"time"
)

// OnConflict provides a K8s-way "retrier" mechanism to avoid conflicts on resource updates.
func OnConflict(fn func() error) error {
	return retry.RetryOnConflict(wait.Backoff{
		Duration: 1500 * time.Millisecond,
		Factor:   1.5,
		Jitter:   0,
		Steps:    9999,
		Cap:      30 * time.Second,
	}, fn)
}
