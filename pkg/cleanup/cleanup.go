package cleanup

import (
	"context"
	"fmt"
	"github.com/cloudogu/k8s-backup-operator/pkg/retry"
	k8sErr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/strings/slices"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sync"
	"time"
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
	var wg sync.WaitGroup

	objects, err := c.findObjects(ctx, defaultCleanupSelector)
	if err != nil {
		return fmt.Errorf("find object: %w", err)
	}
	for _, object := range objects {
		err2 := c.removeFinalizers(ctx, &object)
		if err2 != nil {
			return fmt.Errorf(
				"remove finalizer of object: namespace=%s, kind=%s, name=%s: %w",
				object.GetNamespace(),
				object.GetKind(),
				object.GetName(),
				err2,
			)
		}
		err3 := c.deleteObject(ctx, &object)
		if err3 != nil {
			object.GetKind()
			return fmt.Errorf(
				"delete object namespace=%s, kind=%s, name=%s: %w",
				object.GetNamespace(),
				object.GetKind(),
				object.GetName(),
				err3,
			)
		}
		c.waitForObjectToBeDeleted(ctx, &object, &wg)
	}

	wg.Wait()
	return nil
}

func (c *defaultCleanupManager) findObjects(ctx context.Context, labelSelector *metav1.LabelSelector) ([]unstructured.Unstructured, error) {
	resources, err := c.findResources()
	if err != nil {
		return []unstructured.Unstructured{}, fmt.Errorf("find resources: %w", err)
	}

	var result []unstructured.Unstructured
	for _, resource := range resources {
		selector, err2 := metav1.LabelSelectorAsSelector(labelSelector)
		if err2 != nil {
			return []unstructured.Unstructured{}, fmt.Errorf("convert label selector: %w", err2)
		}

		objects := &unstructured.UnstructuredList{}
		gvk := schema.GroupVersionKind{
			Group:   resource.Group,
			Version: resource.Version,
			Kind:    resource.Kind,
		}
		objects.SetGroupVersionKind(gvk)

		listOptions := client.ListOptions{LabelSelector: &client.MatchingLabelsSelector{Selector: selector}}
		if resource.Namespaced {
			listOptions.Namespace = c.namespace
		}

		err3 := c.client.List(ctx, objects, &listOptions)
		if err3 != nil {
			return []unstructured.Unstructured{}, fmt.Errorf("list objects of resource (%s): %w", gvk, err3)
		}

		for _, item := range objects.Items {
			result = append(result, item)
		}
	}

	return result, nil
}

func (c *defaultCleanupManager) findResources() ([]metav1.APIResource, error) {
	resourcesByGroupAndVersion, err := c.discoveryClient.ServerPreferredResources()
	if err != nil {
		return []metav1.APIResource{}, fmt.Errorf("fetching supported resources: %w", err)
	}

	var result []metav1.APIResource
	for _, resourceList := range resourcesByGroupAndVersion {
		gv, err2 := schema.ParseGroupVersion(resourceList.GroupVersion)
		if err2 != nil {
			return []metav1.APIResource{}, fmt.Errorf("parse group and version from string '%s': %w", resourceList.GroupVersion, err)
		}

		for _, resource := range resourceList.APIResources {
			resource.Group = gv.Group
			resource.Version = gv.Version
			include := len(resource.Verbs) != 0 && slices.Contains(resource.Verbs, deleteVerb)
			exclude := resource.Kind == customResourceDefinitionKind || // Skip crd deletion because we need the provider and snapshot-controller components.
				resource.Kind == podKind || // Skip pod deletion because this would kill our backup operator.
				resource.Group == veleroGroup // Skip velero resource deletion because we need those to restore.
			if include && !exclude {
				result = append(result, resource)
			}
		}
	}

	return result, nil
}

func (c *defaultCleanupManager) deleteObject(ctx context.Context, object client.Object) error {
	propagationPolicy := metav1.DeletePropagationBackground
	deleteOptions := client.DeleteOptions{PropagationPolicy: &propagationPolicy}
	return c.client.Delete(ctx, object, &deleteOptions)
}

func (c *defaultCleanupManager) waitForObjectToBeDeleted(ctx context.Context, object client.Object, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			msg := fmt.Sprintf("Wait for object to be deleted. ns=%s, name=%s", object.GetNamespace(), object.GetName())
			log.FromContext(ctx).Info(msg, "ns", object.GetNamespace(), "name", object.GetName())
			if c.existObject(ctx, object) {
				time.Sleep(time.Second * 10)
			} else {
				break
			}
		}
	}()
}

func (c *defaultCleanupManager) existObject(ctx context.Context, object client.Object) bool {
	objectKey := types.NamespacedName{
		Namespace: object.GetNamespace(),
		Name:      object.GetName(),
	}
	err := c.client.Get(ctx, objectKey, object)
	return !k8sErr.IsNotFound(err)
}

func (c *defaultCleanupManager) removeFinalizers(ctx context.Context, object client.Object) error {
	objectKey := types.NamespacedName{
		Namespace: object.GetNamespace(),
		Name:      object.GetName(),
	}
	err := retry.OnConflict(func() error {
		err := c.client.Get(ctx, objectKey, object)
		if err != nil {
			if k8sErr.IsNotFound(err) {
				return nil
			}
			return err
		}

		object.SetFinalizers(make([]string, 0))
		return c.client.Update(ctx, object)
	})
	return err
}
