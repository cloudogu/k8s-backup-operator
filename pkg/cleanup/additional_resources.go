package cleanup

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	k8sErr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const backupScopeLabelKey = "k8s.cloudogu.com/backup-scope"

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
func newAdditionalResourceManager(dynamicClient dynamicClient, namespace string) *defaultAdditionalResourceManager {
	gvrs := []schema.GroupVersionResource{
		corev1.SchemeGroupVersion.WithResource("configmaps"),
		corev1.SchemeGroupVersion.WithResource("secrets"),
		corev1.SchemeGroupVersion.WithResource("persistentvolumeclaims"),
	}
	clients := make(map[schema.GroupVersionResource]unstructuredClient, len(gvrs))
	for _, gvr := range gvrs {
		clients[gvr] = dynamicClient.Resource(gvr).Namespace(namespace)
	}
	return &defaultAdditionalResourceManager{clients: clients}
}

// cleanupAdditionalResources deletes all additional resources that need to be deleted before restoring the backup.
// It adds those deletions to the wait group.
func (c *defaultAdditionalResourceManager) cleanupAdditionalResources(ctx context.Context, wg *sync.WaitGroup) error {
	log.FromContext(ctx).Info("starting cleanup of additional resources before restore...")

	sortedKeys := slices.SortedFunc(maps.Keys(c.clients), func(a schema.GroupVersionResource, b schema.GroupVersionResource) int {
		return strings.Compare(a.Group+a.Version+a.Resource, b.Group+b.Version+b.Resource)
	})
	for _, gvr := range sortedKeys {
		log.FromContext(ctx).Info("listing additional resources", "gvr", gvr.String())

		client := c.clients[gvr]
		list, err := client.List(ctx, metav1.ListOptions{LabelSelector: backupScopeLabelKey})
		if err != nil {
			return fmt.Errorf("failed to list %s: %w", gvr.Resource, err)
		}

		// Delete resources in foreground, so that all depending resources are deleted before the resource.
		propagationPolicyForeground := metav1.DeletePropagationForeground

		for _, resource := range list.Items {
			if err := client.Delete(ctx, resource.GetName(), metav1.DeleteOptions{PropagationPolicy: &propagationPolicyForeground}); err != nil {
				return fmt.Errorf("failed to delete %s %q: %w", resource.GetKind(), resource.GetName(), err)
			}

			wg.Go(func() { c.waitForResourceDeletion(ctx, client, &resource) })
		}
	}

	return nil
}

func (c *defaultAdditionalResourceManager) waitForResourceDeletion(ctx context.Context, client unstructuredClient, resource *unstructured.Unstructured) {
	for {
		log.FromContext(ctx).Info("waiting for resource to be deleted", "ns", resource.GetNamespace(), "kind", resource.GetKind(), "Name", resource.GetName())
		_, err := client.Get(ctx, resource.GetName(), metav1.GetOptions{})

		exists := !k8sErr.IsNotFound(err)
		if exists {
			// wait for 3 seconds and try again
			time.Sleep(additionalResourceDeleteWaitTime)
		} else {
			log.FromContext(ctx).Info("resource was deleted successfully", "ns", resource.GetNamespace(), "kind", resource.GetKind(), "Name", resource.GetName())
			break
		}
	}
}
