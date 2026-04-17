package cleanup

import (
	"context"
	"fmt"
	"sync"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/log"
)

const defaultWaitTime = time.Second * 3
const defaultCleanupTimeout = time.Minute * 15

var cleanupTimeout = defaultCleanupTimeout

type doguManager interface {
	cleanupDogus(ctx context.Context, wg *sync.WaitGroup) error
}

type additionalResourceManager interface {
	cleanupAdditionalResources(ctx context.Context, wg *sync.WaitGroup) error
}

type DefaultCleanupManager struct {
	doguManager
	additionalResourceManager
}

// NewManager creates a new instance of DefaultCleanupManager.
func NewManager(doguClient doguClient, dynamicClient dynamicClient, namespace string) *DefaultCleanupManager {
	return &DefaultCleanupManager{
		doguManager:               newDoguManager(doguClient),
		additionalResourceManager: newAdditionalResourceManager(dynamicClient, namespace),
	}
}

// Cleanup deletes all resources that need to be deleted before restoring the backup.
// It waits for the deletion of the resources to complete.
// If the deletion fails, the cleanup will fail.
func (c *DefaultCleanupManager) Cleanup(ctx context.Context) error {
	log.FromContext(ctx).Info("starting cleanup before restore...")

	// Create a timeout context for the entire cleanup
	ctx, cancel := context.WithTimeout(ctx, cleanupTimeout)
	defer cancel()

	var wg sync.WaitGroup

	err := c.cleanupDogus(ctx, &wg)
	if err != nil {
		return fmt.Errorf("failed to cleanup dogus: %w", err)
	}

	err = c.cleanupAdditionalResources(ctx, &wg)
	if err != nil {
		return fmt.Errorf("failed to cleanup additional resources: %w", err)
	}

	// Wait for all goroutines OR timeout
	allDeletesDone := make(chan struct{})
	go func() {
		wg.Wait()
		close(allDeletesDone)
	}()

	select {
	case <-allDeletesDone:
		log.FromContext(ctx).Info("... cleanup finished. All resources were deleted successfully.")
		return nil
	case <-ctx.Done():
		return fmt.Errorf("cleanup timed out: %w", ctx.Err())
	}
}
