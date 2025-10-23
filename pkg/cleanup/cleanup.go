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

// Cleanup deletes all components with labels app=ces and not k8s.cloudogu.com/part-of=backup.
func (c *DefaultCleanupManager) Cleanup(ctx context.Context) error {
	log.FromContext(ctx).Info("starting cleanup of dogus before restore...")

	var wg sync.WaitGroup

	doguList, err := c.doguClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list dogus: %w", err)
	}

	for _, dogu := range doguList.Items {
		if err := c.doguClient.Delete(ctx, dogu.Name, metav1.DeleteOptions{}); err != nil {
			return fmt.Errorf("failed to delete dogu %s: %w", dogu.Name, err)
		}

		c.waitForDoguToBeDeleted(ctx, &dogu, &wg)
	}

	wg.Wait()
	log.FromContext(ctx).Info("... cleanup finished. All dogus were deleted successfully.")

	return nil
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
				time.Sleep(time.Second * 3)
			} else {
				log.FromContext(ctx).Info("dogu was deleted successfully", "ns", dogu.GetNamespace(), "Name", dogu.GetName())
				break
			}
		}
	}()
}
