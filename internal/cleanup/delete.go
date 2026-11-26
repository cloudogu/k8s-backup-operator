package cleanup

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cloudogu/k8s-backup-operator/internal/logging"
	k8sErr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// deleteMatching deletes all resources of the given kind in the namespace that match the selector.
// It adds the wait for each deletion to the wait group.
func (c *DefaultCleanupManager) deleteMatching(ctx context.Context, wg *sync.WaitGroup, gvk schema.GroupVersionKind, selector labels.Selector) error {
	logging.Info(ctx, "deleting resources before restore...", "kind", gvk.Kind, "selector", selector.String())

	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(gvk)
	if err := c.client.List(ctx, list, client.InNamespace(c.namespace), client.MatchingLabelsSelector{Selector: selector}); err != nil {
		return fmt.Errorf("failed to list %s: %w", gvk.Kind, err)
	}

	for _, item := range list.Items {
		// The items of a list do not necessarily carry their kind.
		item.SetGroupVersionKind(gvk)
		key := client.ObjectKeyFromObject(&item)

		// Delete in foreground, so that all depending resources are deleted before the resource.
		err := c.client.Delete(ctx, &item, client.PropagationPolicy(metav1.DeletePropagationForeground))
		if k8sErr.IsNotFound(err) {
			logging.Debug(ctx, "resource was already deleted", "kind", gvk.Kind, "name", key.Name)
			continue
		} else if err != nil {
			return fmt.Errorf("failed to delete %s %q: %w", gvk.Kind, key.Name, err)
		}

		wg.Go(func() { c.waitForDeletion(ctx, gvk, key) })
	}

	return nil
}

// waitForDeletion polls the resource until it is gone or the context ends.
func (c *DefaultCleanupManager) waitForDeletion(ctx context.Context, gvk schema.GroupVersionKind, key client.ObjectKey) {
	var lastReportedError string

	for {
		logging.Debug(ctx, "waiting for resource to be deleted", "kind", gvk.Kind, "ns", key.Namespace, "name", key.Name)
		resource := &unstructured.Unstructured{}
		resource.SetGroupVersionKind(gvk)
		err := c.client.Get(ctx, key, resource)

		if ctx.Err() != nil {
			logging.Info(context.WithoutCancel(ctx), "context was cancelled or deadline exceeded, stop waiting for resource deletion",
				"kind", gvk.Kind, "ns", key.Namespace, "name", key.Name)
			return
		} else if k8sErr.IsNotFound(err) {
			logging.Info(ctx, "resource was deleted successfully", "kind", gvk.Kind, "ns", key.Namespace, "name", key.Name)
			return
		}

		// The same error repeats every few seconds, so only a new one is reported.
		if err != nil && err.Error() != lastReportedError {
			lastReportedError = err.Error()
			logging.Error(ctx, err, "cannot observe whether the resource was deleted, still waiting",
				"kind", gvk.Kind, "ns", key.Namespace, "name", key.Name)
		}

		select {
		case <-ctx.Done():
		case <-time.After(c.deleteWaitTime):
		}
	}
}
