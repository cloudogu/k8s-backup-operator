package cleanup

import (
	"context"
	"errors"
	"fmt"
	"github.com/cloudogu/k8s-backup-operator/pkg/retry"
	k8sErr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/log"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/strings/slices"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	deleteVerb                   = "delete"
	customResourceDefinitionKind = "CustomResourceDefinition"
	podKind                      = "Pod"
	veleroGroup                  = "velero.io"
)

var defaultCleanupSelector = &metav1.LabelSelector{
	MatchExpressions: []metav1.LabelSelectorRequirement{
		{
			Key:      "app",
			Operator: metav1.LabelSelectorOpIn,
			Values:   []string{"ces"},
		},
		{
			Key:      "k8s.cloudogu.com/part-of",
			Operator: metav1.LabelSelectorOpNotIn,
			Values:   []string{"backup"},
		},
	},
}

type defaultCleanupManager struct {
	namespace       string
	client          k8sClient
	discoveryClient discoveryInterface
}

// NewManager creates a new instance of defaultCleanupManager.
func NewManager(namespace string, client k8sClient, discoveryClient discoveryInterface) Manager {
	return &defaultCleanupManager{namespace: namespace, client: client, discoveryClient: discoveryClient}
}

// Cleanup deletes all components with labels app=ces and not k8s.cloudogu.com/part-of=backup.
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
		return fmt.Errorf("failed to create selector from given label selector %s: %w", labelSelector, err)
	}

	var errs []error
	for _, list := range lists {
		deleteErrs := c.deleteApiResourcesByLabelSelector(ctx, list, selector)
		errs = append(errs, deleteErrs...)
	}

	if len(errs) != 0 {
		return fmt.Errorf("failed to delete api resources with label selector %q: %w", selector, errors.Join(errs...))
	}

	return nil
}

func (c *defaultCleanupManager) deleteApiResourcesByLabelSelector(ctx context.Context, list *metav1.APIResourceList, selector labels.Selector) []error {
	if len(list.APIResources) == 0 {
		return nil
	}
	gv, err := schema.ParseGroupVersion(list.GroupVersion)
	if err != nil {
		log.FromContext(ctx).Error(err, fmt.Sprintf("failed to delete api resources with group version: %s", list.GroupVersion))
		return nil
	}

	var errs []error
	for _, resource := range list.APIResources {
		if len(resource.Verbs) != 0 && slices.Contains(resource.Verbs, deleteVerb) {
			resource.Group = gv.Group
			resource.Version = gv.Version

			deleteErrs := c.deleteByLabelSelector(ctx, resource, selector)
			errs = append(errs, deleteErrs...)
		}
	}

	return errs
}

func (c *defaultCleanupManager) deleteByLabelSelector(ctx context.Context, resource metav1.APIResource, labelSelector labels.Selector) []error {
	gvk := groupVersionKind(resource)
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(gvk)

	// Skip crd deletion because we need the provider and snapshot-controller components.
	if gvk.Kind == customResourceDefinitionKind ||
		// Skip pod deletion because this would kill our backup operator.
		gvk.Kind == podKind ||
		// Skip velero resource deletion because we need those to restore.
		gvk.Group == veleroGroup {
		return nil
	}

	listOptions := client.ListOptions{LabelSelector: &client.MatchingLabelsSelector{Selector: labelSelector}}
	propagationPolicy := metav1.DeletePropagationBackground
	deleteOptions := client.DeleteOptions{PropagationPolicy: &propagationPolicy}
	if resource.Namespaced {
		listOptions.Namespace = c.namespace
	}

	objectList := &unstructured.UnstructuredList{}
	objectList.SetGroupVersionKind(gvk)
	err := c.client.List(ctx, objectList, &listOptions)
	if err != nil {
		return []error{fmt.Errorf("failed to list objects in %s: %w", gvk, err)}
	}

	var errs []error
	for _, item := range objectList.Items {
		err := c.removeFinalizers(ctx, &item)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to remove finalizers for %s/%s (%s): %w", item.GetNamespace(), item.GetName(), gvk, err))
		}

		err = c.client.Delete(ctx, &item, &deleteOptions)
		if err != nil && !k8sErr.IsNotFound(err) {
			errs = append(errs, fmt.Errorf("failed to delete %s/%s (%s): %w", item.GetNamespace(), item.GetName(), gvk, err))
		}
	}

	return errs
}

func (c *defaultCleanupManager) removeFinalizers(ctx context.Context, item client.Object) error {
	err := retry.OnConflict(func() error {
		err := c.client.Get(ctx, types.NamespacedName{
			Namespace: item.GetNamespace(),
			Name:      item.GetName(),
		}, item)
		if err != nil {
			if k8sErr.IsNotFound(err) {
				return nil
			}
			return err
		}

		item.SetFinalizers(make([]string, 0))
		return c.client.Update(ctx, item)
	})
	return err
}

func groupVersionKind(resource metav1.APIResource) schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   resource.Group,
		Version: resource.Version,
		Kind:    resource.Kind,
	}
}
