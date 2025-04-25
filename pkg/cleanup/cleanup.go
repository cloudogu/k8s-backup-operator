package cleanup

import (
	"context"
	"fmt"
	"github.com/cloudogu/k8s-backup-operator/pkg/retry"
	"gopkg.in/yaml.v3"
	k8sErr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/strings/slices"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"strings"
	"sync"
	"time"
)

const (
	deleteVerb                   = "delete"
	customResourceDefinitionKind = "CustomResourceDefinition"
	endpointsKind                = "Endpoints"
	veleroGroup                  = "velero.io"
	configMapName                = "k8s-backup-operator-cleanup-exclude"
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

type ExcludeEntry struct {
	Group   string `yaml:"group"`
	Version string `yaml:"version"`
	Kind    string `yaml:"kind"`
	Name    string `yaml:"name"`
}

type groupVersionKindName struct {
	Gvk  schema.GroupVersionKind
	Name string
}

type defaultCleanupManager struct {
	namespace       string
	client          k8sClient
	discoveryClient discoveryInterface
	configMapClient configMapClient
}

// NewManager creates a new instance of defaultCleanupManager.
func NewManager(namespace string, client k8sClient, discoveryClient discoveryInterface, configMapClient configMapClient) Manager {
	return &defaultCleanupManager{namespace: namespace, client: client, discoveryClient: discoveryClient, configMapClient: configMapClient}
}

// Cleanup deletes all components with labels app=ces and not k8s.cloudogu.com/part-of=backup.
func (c *defaultCleanupManager) Cleanup(ctx context.Context) error {
	var wg sync.WaitGroup

	objects, err := c.findObjects(ctx, defaultCleanupSelector)
	if err != nil {
		return fmt.Errorf("find object: %w", err)
	}
	for _, object := range objects {
		err = c.removeFinalizers(ctx, &object)
		if err != nil {
			return objectErr("remove finalizer of object", object, err)
		}
		err = c.deleteObject(ctx, &object)
		if err != nil {
			return objectErr("delete object namespace", object, err)
		}
		c.waitForObjectToBeDeleted(ctx, &object, &wg)
	}

	wg.Wait()
	return nil
}

func objectErr(msg string, object unstructured.Unstructured, err error) error {
	return fmt.Errorf("%s: namespace=%s, kind=%s, Name=%s: %w", msg, object.GetNamespace(), object.GetKind(), object.GetName(), err)
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

		err = c.client.List(ctx, objects, &listOptions)
		if err != nil {
			return []unstructured.Unstructured{}, fmt.Errorf("list objects of resource (%s): %w", gvk, err)
		}

		result = append(result, objects.Items...)
	}

	gvksToExclude := c.readGvkToExclude(ctx)

	result = filterObjects(ctx, result, gvksToExclude)

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
			exclude := resource.Kind == customResourceDefinitionKind || // Skip crd deletion because we need the component-crd.
				resource.Group == veleroGroup || // Skip velero resource deletion because we need those to restore.
				resource.Kind == endpointsKind // Skip endpoint resources deletion because they are deleted by services
			if include && !exclude {
				result = append(result, resource)
			}
		}
	}

	return result, nil
}

func filterObjects(ctx context.Context, objects []unstructured.Unstructured, gvksToExclude []groupVersionKindName) []unstructured.Unstructured {
	filtered := []unstructured.Unstructured{}
	for _, obj := range objects {
		if !isObjectExcluded(obj, gvksToExclude) {
			filtered = append(filtered, obj)
		}
	}

	return filtered
}

func (c *defaultCleanupManager) deleteObject(ctx context.Context, object client.Object) error {
	propagationPolicy := metav1.DeletePropagationBackground
	deleteOptions := client.DeleteOptions{PropagationPolicy: &propagationPolicy}
	err := c.client.Delete(ctx, object, &deleteOptions)
	// The resource was already deleted by parent resource, so this is not a real error.
	if k8sErr.IsNotFound(err) {
		return nil
	}
	return err
}

func (c *defaultCleanupManager) waitForObjectToBeDeleted(ctx context.Context, object client.Object, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			if c.existObject(ctx, object) {
				time.Sleep(time.Second * 3)
			} else {
				break
			}
			log.FromContext(ctx).Info("Wait for object to be deleted", "ns", object.GetNamespace(), "Name", object.GetName(), "Gvk", object.GetObjectKind().GroupVersionKind())
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
		finalizers := make([]string, 0)
		for _, finalizer := range object.GetFinalizers() {
			if strings.HasPrefix(finalizer, "kubernetes.io") {
				log.FromContext(ctx).Info(fmt.Sprintf("not removing kubernetes finalizer for resource %s(%s): %v",
					object.GetName(), object.GetObjectKind().GroupVersionKind(), finalizers))
				finalizers = append(finalizers, finalizer)
			}
		}
		object.SetFinalizers(finalizers)
		return c.client.Update(ctx, object)
	})
	return err
}

func isObjectExcluded(resource unstructured.Unstructured, shouldBeExcluded []groupVersionKindName) bool {
	resourceGvkn := groupVersionKind(resource)
	for _, gvkn := range shouldBeExcluded {
		if gvkn.matches(resourceGvkn) {
			return true
		}
	}
	return false
}

func (c *defaultCleanupManager) readGvkToExclude(ctx context.Context) []groupVersionKindName {
	configMap, err := c.configMapClient.Get(ctx, configMapName, metav1.GetOptions{})
	if err != nil {
		log.FromContext(ctx).Info("No ConfigMap found: %s", "configmapName", "k8s-backup-operator-cleanup-exclude")
		return []groupVersionKindName{}
	}

	shouldBeExcludedString := configMap.Data["cleanup"]
	var exclude struct {
		Exclude []ExcludeEntry `yaml:"exclude"`
	}

	err = yaml.Unmarshal([]byte(shouldBeExcludedString), &exclude)
	if err != nil {
		log.FromContext(ctx).Info("failed to unmarshal config map")
		return []groupVersionKindName{}
	}
	var shouldBeExcluded []groupVersionKindName
	for _, entry := range exclude.Exclude {
		gvk := schema.GroupVersionKind{
			Group:   entry.Group,
			Version: entry.Version,
			Kind:    entry.Kind,
		}
		shouldBeExcluded = append(shouldBeExcluded, groupVersionKindName{
			Gvk:  gvk,
			Name: entry.Name,
		})
	}
	return shouldBeExcluded
}

func (g groupVersionKindName) matches(gvkn groupVersionKindName) bool {
	return (gvkn.Gvk.Group == g.Gvk.Group || g.Gvk.Group == "*" || g.Gvk.Group == "") &&
		(gvkn.Gvk.Version == g.Gvk.Version || g.Gvk.Version == "*" || g.Gvk.Version == "") &&
		(gvkn.Gvk.Kind == g.Gvk.Kind || g.Gvk.Kind == "*" || g.Gvk.Kind == "") &&
		(gvkn.Name == g.Name || g.Name == "*" || g.Name == "")
}

func groupVersionKind(resource unstructured.Unstructured) groupVersionKindName {
	return groupVersionKindName{
		Gvk:  resource.GroupVersionKind(),
		Name: resource.GetName(),
	}
}
