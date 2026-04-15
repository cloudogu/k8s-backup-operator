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

var doguDeleteWaitTime = defaultWaitTime

type doguClient interface {
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*doguv2.Dogu, error)
	List(ctx context.Context, opts metav1.ListOptions) (*doguv2.DoguList, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
}

type defaultDoguManager struct {
	doguClient doguClient
}

// newDoguManager creates a new instance of defaultDoguManager.
func newDoguManager(doguClient doguClient) *defaultDoguManager {
	return &defaultDoguManager{doguClient: doguClient}
}

// cleanupDogus deletes all dogus that need to be deleted before restoring the backup.
// It adds those deletions to the wait group.
func (c *defaultDoguManager) cleanupDogus(ctx context.Context, wg *sync.WaitGroup) error {
	log.FromContext(ctx).Info("starting cleanup of dogus before restore...")

	doguList, err := c.doguClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list dogus: %w", err)
	}

	// Delete dogus in foreground, so that all depending ressources are deleted before the dogu.
	propagationPolicyForeground := metav1.DeletePropagationForeground

	for _, dogu := range doguList.Items {
		if err := c.doguClient.Delete(ctx, dogu.Name, metav1.DeleteOptions{PropagationPolicy: &propagationPolicyForeground}); err != nil {
			return fmt.Errorf("failed to delete dogu %q: %w", dogu.Name, err)
		}

		wg.Go(func() { c.waitForDoguDeletion(ctx, &dogu) })
	}

	return nil
}

func (c *defaultDoguManager) waitForDoguDeletion(ctx context.Context, dogu *doguv2.Dogu) {
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
}
