package cleanup

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cloudogu/k8s-backup-operator/internal/logging"
	doguv2 "github.com/cloudogu/k8s-dogu-lib/v3/api/v2"
	k8sErr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

const doguNameLabelKey = "k8s.cloudogu.com/dogu.name"

var doguDeleteWaitTime = defaultWaitTime

type doguClient interface {
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*doguv2.Dogu, error)
	List(ctx context.Context, opts metav1.ListOptions) (*doguv2.DoguList, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
}

type defaultDoguManager struct {
	doguClient doguClient
	pvcClient  unstructuredClient
}

// newDoguManager creates a new instance of defaultDoguManager.
func newDoguManager(doguClient doguClient, pvcClient unstructuredClient) *defaultDoguManager {
	return &defaultDoguManager{doguClient: doguClient, pvcClient: pvcClient}
}

// cleanupDogus deletes the dogus and the dogu pvcs matching the selectors before restoring the backup.
// The pvcs are deleted separately because the statefulset pvcs of v3 dogus are not owned by their dogu.
// It adds those deletions to the wait group.
func (c *defaultDoguManager) cleanupDogus(ctx context.Context, wg *sync.WaitGroup, doguSelector, pvcSelector labels.Selector) error {
	logging.Info(ctx, "starting cleanup of dogus before restore...", "dogus", doguSelector.String(), "pvcs", pvcSelector.String())

	doguList, err := c.doguClient.List(ctx, metav1.ListOptions{LabelSelector: doguSelector.String()})
	if err != nil {
		return fmt.Errorf("failed to list dogus: %w", err)
	}

	// Delete dogus in foreground, so that all depending resources are deleted before the dogu.
	deleteOptions := metav1.DeleteOptions{PropagationPolicy: new(metav1.DeletePropagationForeground)}

	for _, dogu := range doguList.Items {
		if deleteErr := c.doguClient.Delete(ctx, dogu.Name, deleteOptions); deleteErr != nil && !k8sErr.IsNotFound(deleteErr) {
			return fmt.Errorf("failed to delete dogu %q: %w", dogu.Name, deleteErr)
		} else if k8sErr.IsNotFound(deleteErr) {
			logging.Debug(ctx, "dogu was already deleted", "dogu", dogu.Name)
			continue
		}
		wg.Go(func() { c.waitForDoguDeletion(ctx, &dogu) })
	}

	doguPvcList, err := c.pvcClient.List(ctx, metav1.ListOptions{LabelSelector: pvcSelector.String()})
	if err != nil {
		return fmt.Errorf("failed to list dogu pvcs: %w", err)
	}

	for _, pvc := range doguPvcList.Items {
		name := pvc.GetName()
		if deleteErr := c.pvcClient.Delete(ctx, name, deleteOptions); deleteErr != nil && !k8sErr.IsNotFound(deleteErr) {
			return fmt.Errorf("failed to delete dogu pvc %q: %w", name, deleteErr)
		} else if k8sErr.IsNotFound(deleteErr) {
			logging.Debug(ctx, "pvc was already deleted", "pvc", name)
			continue
		}
		wg.Go(func() { waitForResourceDeletion(ctx, c.pvcClient, &pvc) })
	}

	return nil
}

func (c *defaultDoguManager) waitForDoguDeletion(ctx context.Context, dogu *doguv2.Dogu) {
	var lastReportedError string

	for {
		logging.Debug(ctx, "waiting for dogu to be deleted", "ns", dogu.GetNamespace(), "Name", dogu.GetName())
		_, err := c.doguClient.Get(ctx, dogu.GetName(), metav1.GetOptions{})

		if ctx.Err() != nil {
			logging.Info(context.WithoutCancel(ctx), "context was cancelled or deadline exceeded, stop waiting for dogu deletion",
				"ns", dogu.GetNamespace(), "Name", dogu.GetName())
			break
		} else if exists := !k8sErr.IsNotFound(err); exists {
			// The same error repeats every few seconds, so only a new one is reported.
			if err != nil && err.Error() != lastReportedError {
				lastReportedError = err.Error()
				logging.Error(ctx, err, "cannot observe whether the dogu was deleted, still waiting",
					"ns", dogu.GetNamespace(), "Name", dogu.GetName())
			}

			// wait for 3 seconds and try again
			time.Sleep(doguDeleteWaitTime)
		} else {
			logging.Info(ctx, "dogu was deleted successfully", "ns", dogu.GetNamespace(), "Name", dogu.GetName())
			break
		}
	}
}
