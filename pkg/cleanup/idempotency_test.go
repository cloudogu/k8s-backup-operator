package cleanup

import (
	"context"
	"sync"
	"testing"

	doguv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestRepeatedDoguCleanupDeletesEachPersistedObjectOnce(t *testing.T) {
	ctx := t.Context()
	dogu := doguv2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "dogu", Namespace: "ecosystem"}}
	client := &persistentDoguClient{objects: map[string]doguv2.Dogu{dogu.Name: dogu}}
	manager := newDoguManager(client)

	for range 2 {
		var waitGroup sync.WaitGroup
		require.NoError(t, manager.cleanupDogus(ctx, &waitGroup))
		waitGroup.Wait()
	}

	assert.Equal(t, 1, client.deleteCalls)
	assert.Empty(t, client.objects)
}

func TestRepeatedAdditionalResourceCleanupDeletesEachPersistedObjectOnce(t *testing.T) {
	ctx := t.Context()
	resource := &unstructured.Unstructured{}
	resource.SetAPIVersion("v1")
	resource.SetKind("ConfigMap")
	resource.SetName("configuration")
	resource.SetNamespace("ecosystem")
	resource.SetLabels(map[string]string{backupScopeLabelKey: "restore"})

	resourceClient := &persistentUnstructuredClient{
		objects: map[string]*unstructured.Unstructured{resource.GetName(): resource},
	}
	manager := &defaultAdditionalResourceManager{
		clients: map[schema.GroupVersionResource]unstructuredClient{
			{Version: "v1", Resource: "configmaps"}: resourceClient,
		},
	}

	for range 2 {
		var waitGroup sync.WaitGroup
		require.NoError(t, manager.cleanupAdditionalResources(ctx, &waitGroup))
		waitGroup.Wait()
	}

	assert.Equal(t, 1, resourceClient.deleteCalls)
	assert.Empty(t, resourceClient.objects)
}

func TestDoguGetNotFoundRaceCompletesDeletionWaitWithoutSleeping(t *testing.T) {
	client := &persistentDoguClient{objects: map[string]doguv2.Dogu{}}
	manager := newDoguManager(client)

	manager.waitForDoguDeletion(t.Context(), &doguv2.Dogu{
		ObjectMeta: metav1.ObjectMeta{Name: "already-gone", Namespace: "ecosystem"},
	})

	assert.Equal(t, 1, client.getCalls)
}

func TestAdditionalResourceGetNotFoundRaceCompletesDeletionWaitWithoutSleeping(t *testing.T) {
	client := &persistentUnstructuredClient{objects: map[string]*unstructured.Unstructured{}}
	resource := &unstructured.Unstructured{}
	resource.SetName("already-gone")
	resource.SetNamespace("ecosystem")
	manager := &defaultAdditionalResourceManager{}

	manager.waitForResourceDeletion(t.Context(), client, resource)

	assert.Equal(t, 1, client.getCalls)
}

type persistentDoguClient struct {
	mutex       sync.Mutex
	objects     map[string]doguv2.Dogu
	deleteCalls int
	getCalls    int
}

func (c *persistentDoguClient) Get(
	_ context.Context,
	name string,
	_ metav1.GetOptions,
) (*doguv2.Dogu, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.getCalls++
	dogu, exists := c.objects[name]
	if !exists {
		return nil, apierrors.NewNotFound(
			schema.GroupResource{Group: "k8s.cloudogu.com", Resource: "dogus"},
			name,
		)
	}
	return dogu.DeepCopy(), nil
}

func (c *persistentDoguClient) List(
	_ context.Context,
	_ metav1.ListOptions,
) (*doguv2.DoguList, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	result := &doguv2.DoguList{}
	for _, dogu := range c.objects {
		result.Items = append(result.Items, *dogu.DeepCopy())
	}
	return result, nil
}

func (c *persistentDoguClient) Delete(
	_ context.Context,
	name string,
	_ metav1.DeleteOptions,
) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if _, exists := c.objects[name]; !exists {
		return apierrors.NewNotFound(
			schema.GroupResource{Group: "k8s.cloudogu.com", Resource: "dogus"},
			name,
		)
	}
	c.deleteCalls++
	delete(c.objects, name)
	return nil
}

type persistentUnstructuredClient struct {
	mutex       sync.Mutex
	objects     map[string]*unstructured.Unstructured
	deleteCalls int
	getCalls    int
}

func (c *persistentUnstructuredClient) Get(
	_ context.Context,
	name string,
	_ metav1.GetOptions,
	_ ...string,
) (*unstructured.Unstructured, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.getCalls++
	resource, exists := c.objects[name]
	if !exists {
		return nil, apierrors.NewNotFound(
			schema.GroupResource{Resource: "resources"},
			name,
		)
	}
	return resource.DeepCopy(), nil
}

func (c *persistentUnstructuredClient) List(
	_ context.Context,
	_ metav1.ListOptions,
) (*unstructured.UnstructuredList, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	result := &unstructured.UnstructuredList{}
	for _, resource := range c.objects {
		result.Items = append(result.Items, *resource.DeepCopy())
	}
	return result, nil
}

func (c *persistentUnstructuredClient) Delete(
	_ context.Context,
	name string,
	_ metav1.DeleteOptions,
	_ ...string,
) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if _, exists := c.objects[name]; !exists {
		return apierrors.NewNotFound(schema.GroupResource{Resource: "resources"}, name)
	}
	c.deleteCalls++
	delete(c.objects, name)
	return nil
}
