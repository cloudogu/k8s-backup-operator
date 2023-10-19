package cleanup

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/strings/slices"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var defaultCleanupSelector = &metav1.LabelSelector{
	MatchExpressions: []metav1.LabelSelectorRequirement{
		{
			Operator: metav1.LabelSelectorOpIn,
			Values:   []string{"app=ces"},
		},
		{
			Operator: metav1.LabelSelectorOpNotIn,
			Values:   []string{"k8s.cloudogu.com/part-of=backup"},
		},
	},
}

type defaultCleanupManager struct {
	namespace       string
	client          k8sClient
	discoveryClient discoveryInterface
}

func NewManager(namespace string, client k8sClient, discoveryClient discoveryInterface) Manager {
	return &defaultCleanupManager{namespace: namespace, client: client, discoveryClient: discoveryClient}
}

func (c *defaultCleanupManager) Cleanup(ctx context.Context) error {
	return c.deleteResourcesByLabelSelector(ctx, defaultCleanupSelector)
}

func (c *defaultCleanupManager) deleteResourcesByLabelSelector(ctx context.Context, labelSelector *metav1.LabelSelector) error {
	lists, err := c.discoveryClient.ServerPreferredResources()
	if err != nil {
		return fmt.Errorf("failed to get resource lists from server: %w", err)
	}

	selector, err := metav1.LabelSelectorAsSelector(labelSelector)
	if err != nil {
		return fmt.Errorf("failed to create selector from given label selector: %w", err)
	}

	for _, list := range lists {
		err = c.deleteApiResourcesByLabelSelector(ctx, list, selector)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *defaultCleanupManager) deleteApiResourcesByLabelSelector(ctx context.Context, list *metav1.APIResourceList, selector labels.Selector) error {
	if len(list.APIResources) == 0 {
		return nil
	}

	for _, resource := range list.APIResources {
		if len(resource.Verbs) != 0 && slices.Contains(resource.Verbs, "delete") {
			err := c.deleteByLabelSelector(ctx, resource, selector)
			if err != nil {
				return err
			}

		}
	}

	return nil
}

func (c *defaultCleanupManager) deleteByLabelSelector(ctx context.Context, resource metav1.APIResource, labelSelector labels.Selector) error {
	gvk := GroupVersionKind(resource)
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(gvk)

	listOptions := client.ListOptions{LabelSelector: &client.MatchingLabelsSelector{Selector: labelSelector}}
	if resource.Namespaced {
		listOptions.Namespace = c.namespace
	}
	err := c.client.DeleteAllOf(ctx, u, &client.DeleteAllOfOptions{
		ListOptions: listOptions,
	})
	if err != nil {
		return fmt.Errorf("failed to delete all resources of kind %s with label selector %s in namespace %s: %w",
			gvk, labelSelector, c.namespace, err)
	}

	return nil
}

func GroupVersionKind(resource metav1.APIResource) schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   resource.Group,
		Version: resource.Version,
		Kind:    resource.Kind,
	}
}
