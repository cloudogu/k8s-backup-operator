package cleanup

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/cloudogu/k8s-backup-operator/internal/logging"
	k8sErr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

const (
	backupScopeLabelKey   = "k8s.cloudogu.com/backup-scope"
	componentNameLabelKey = "k8s.cloudogu.com/component.name"
)

var additionalResourceDeleteWaitTime = defaultWaitTime

type dynamicClient interface {
	Resource(resource schema.GroupVersionResource) dynamic.NamespaceableResourceInterface
}

type unstructuredClient interface {
	Get(ctx context.Context, name string, options metav1.GetOptions, subresources ...string) (*unstructured.Unstructured, error)
	List(ctx context.Context, opts metav1.ListOptions) (*unstructured.UnstructuredList, error)
	Delete(ctx context.Context, name string, options metav1.DeleteOptions, subresources ...string) error
}

type defaultAdditionalResourceManager struct {
	clients map[schema.GroupVersionResource]unstructuredClient
}

// newAdditionalResourceManager creates a new instance of defaultAdditionalResourceManager.
func newAdditionalResourceManager(clients map[schema.GroupVersionResource]unstructuredClient) *defaultAdditionalResourceManager {
	return &defaultAdditionalResourceManager{clients: clients}
}

// cleanupAdditionalResources deletes the additional resources matching the selector before restoring the backup.
// It adds those deletions to the wait group.
func (c *defaultAdditionalResourceManager) cleanupAdditionalResources(ctx context.Context, wg *sync.WaitGroup, selector labels.Selector) error {
	logging.Info(ctx, "starting cleanup of additional resources before restore...", "selector", selector.String())

	sortedKeys := slices.SortedFunc(maps.Keys(c.clients), func(a schema.GroupVersionResource, b schema.GroupVersionResource) int {
		return strings.Compare(a.Group+a.Version+a.Resource, b.Group+b.Version+b.Resource)
	})
	for _, gvr := range sortedKeys {
		logging.Debug(ctx, "listing additional resources", "gvr", gvr.String())

		client := c.clients[gvr]
		list, err := client.List(ctx, metav1.ListOptions{LabelSelector: selector.String()})
		if err != nil {
			return fmt.Errorf("failed to list %s: %w", gvr.Resource, err)
		}

		// Delete resources in foreground, so that all depending resources are deleted before the resource.
		propagationPolicyForeground := metav1.DeletePropagationForeground

		for _, resource := range list.Items {
			name := resource.GetName()

			if deleteErr := client.Delete(ctx, name, metav1.DeleteOptions{PropagationPolicy: &propagationPolicyForeground}); deleteErr != nil && !k8sErr.IsNotFound(deleteErr) {
				return fmt.Errorf("failed to delete %s %q: %w", resource.GetKind(), name, deleteErr)
			} else if k8sErr.IsNotFound(deleteErr) {
				logging.Debug(ctx, "resource was already deleted", "resource", name, "kind", resource.GetKind())
				continue
			}

			wg.Go(func() { waitForResourceDeletion(ctx, client, &resource) })
		}
	}

	return nil
}

func waitForResourceDeletion(ctx context.Context, client unstructuredClient, resource *unstructured.Unstructured) {
	for {
		logging.Debug(ctx, "waiting for resource to be deleted", "ns", resource.GetNamespace(), "kind", resource.GetKind(), "Name", resource.GetName())
		_, err := client.Get(ctx, resource.GetName(), metav1.GetOptions{})

		if ctx.Err() != nil {
			logging.Info(context.WithoutCancel(ctx), "context was cancelled or deadline exceeded, stop waiting for resource deletion",
				"ns", resource.GetNamespace(), "kind", resource.GetKind(), "Name", resource.GetName())
			break
		} else if exists := !k8sErr.IsNotFound(err); exists {
			// wait for 3 seconds and try again
			time.Sleep(additionalResourceDeleteWaitTime)
		} else {
			logging.Info(ctx, "resource was deleted successfully", "ns", resource.GetNamespace(), "kind", resource.GetKind(), "Name", resource.GetName())
			break
		}
	}
}
