package cleanup

import (
	"context"
	"fmt"
	"sync"
	"time"

	doguv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	k8sErr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const defaultWaitTime = time.Second * 3
const defaultCleanupTimeout = time.Minute * 15

var doguDeleteWaitTime = defaultWaitTime
var cleanupTimeout = defaultCleanupTimeout

type doguClient interface {
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*doguv2.Dogu, error)
	List(ctx context.Context, opts metav1.ListOptions) (*doguv2.DoguList, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
}

type DefaultCleanupManager struct {
	doguClient doguClient
}

// NewManager creates a new instance of DefaultCleanupManager.
func NewManager(doguClient doguClient) *DefaultCleanupManager {
	return &DefaultCleanupManager{doguClient: doguClient}
}

// Cleanup deletes all resources that need to be deleted before restoring the backup.
// It waits for the deletion of all dogus to complete.
// If the deletion of a dogu fails, the cleanup will fail.
func (c *DefaultCleanupManager) Cleanup(ctx context.Context) error {
	log.FromContext(ctx).Info("starting cleanup of dogus before restore...")

	// Create a timeout context for the entire cleanup
	ctx, cancel := context.WithTimeout(ctx, cleanupTimeout)
	defer cancel()

	var wg sync.WaitGroup

	doguList, err := c.doguClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list dogus: %w", err)
	}

	// Delete dogus in foreground, so that all depending ressources are deleted before the dogu.
	propagationPolicyForeground := metav1.DeletePropagationForeground

	for _, dogu := range doguList.Items {
		if err := c.doguClient.Delete(ctx, dogu.Name, metav1.DeleteOptions{PropagationPolicy: &propagationPolicyForeground}); err != nil {
			return fmt.Errorf("failed to delete dogu %s: %w", dogu.Name, err)
		}

		c.waitForDoguToBeDeleted(ctx, &dogu, &wg)
	}

	// Wait for all goroutines OR timeout
	allDeletesDone := make(chan struct{})
	go func() {
		wg.Wait()
		close(allDeletesDone)
	}()

	select {
	case <-allDeletesDone:
		log.FromContext(ctx).Info("... cleanup finished. All dogus were deleted successfully.")
		return nil
	case <-ctx.Done():
		return fmt.Errorf("cleanup timed out: %w", ctx.Err())
	}
}

func (c *DefaultCleanupManager) waitForDoguToBeDeleted(ctx context.Context, dogu *doguv2.Dogu, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			log.FromContext(ctx).Info("waiting for dogu to be deleted", "ns", dogu.GetNamespace(), "Name", dogu.GetName())
			_, err := c.doguClient.Get(ctx, dogu.GetName(), metav1.GetOptions{})

			exists := !k8sErr.IsNotFound(err)
			if exists {
				// wait for 3 seconds and try again
				time.Sleep(doguDeleteWaitTime)
			} else {
				log.FromContext(ctx).Info("dogu was deleted successfully", "ns", dogu.GetNamespace(), "Name", dogu.GetName())
				break
			}
		}
	}()
}
