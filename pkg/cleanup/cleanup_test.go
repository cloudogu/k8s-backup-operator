package cleanup

import (
	"context"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
)

func TestFindResources(t *testing.T) {
	t.Run("should only delete resources that can be deleted", func(t *testing.T) {
		discoveryMock := newMockDiscoveryInterface(t)

		resources := []*metav1.APIResourceList{
			{APIResources: []metav1.APIResource{
				{Kind: "Example", Verbs: metav1.Verbs{"create", "update"}},
			}, GroupVersion: "k8s.example.com/v1"},
			{APIResources: []metav1.APIResource{
				{Kind: "MyKind", Verbs: metav1.Verbs{"create", "update", "delete"}},
			}, GroupVersion: "k8s.example.com/v2"},
		}

		discoveryMock.EXPECT().ServerPreferredResources().Return(resources, nil)
		sut := &defaultCleanupManager{discoveryClient: discoveryMock}

		result, _ := sut.findResources()

		assert.Equal(t, 1, len(result))
		assert.Equal(t, "MyKind", result[0].Kind)
	})

	t.Run("should use GroupVersion from APIResourceList as Group and Version for APIResource", func(t *testing.T) {
		discoveryMock := newMockDiscoveryInterface(t)

		resources := []*metav1.APIResourceList{
			{APIResources: []metav1.APIResource{
				{Kind: "MyKind", Group: "AAA", Version: "VVV", Verbs: metav1.Verbs{"create", "update", "delete"}},
			}, GroupVersion: "k8s.example.com/v2"},
		}

		discoveryMock.EXPECT().ServerPreferredResources().Return(resources, nil)
		sut := &defaultCleanupManager{discoveryClient: discoveryMock}

		result, _ := sut.findResources()

		assert.Equal(t, 1, len(result))
		assert.Equal(t, "MyKind", result[0].Kind)
		assert.Equal(t, "k8s.example.com", result[0].Group)
		assert.Equal(t, "v2", result[0].Version)
	})

	t.Run("should not delete CustomResourceDefinition, Pods and resources under ApiGroup velero.io", func(t *testing.T) {
		discoveryMock := newMockDiscoveryInterface(t)

		resources := []*metav1.APIResourceList{
			{APIResources: []metav1.APIResource{
				{Kind: "CustomResourceDefinition", Verbs: metav1.Verbs{"create", "update", "delete"}},
				{Kind: "Pod", Verbs: metav1.Verbs{"create", "update", "delete"}},
			}, GroupVersion: "k8s.example.com/v1"},
			{APIResources: []metav1.APIResource{
				{Kind: "MyKind", Verbs: metav1.Verbs{"create", "update", "delete"}},
			}, GroupVersion: "k8s.example.com/v2"},
			{APIResources: []metav1.APIResource{
				{Kind: "VeleroKind1", Group: "velero.io", Verbs: metav1.Verbs{"create", "update", "delete"}},
				{Kind: "VeleroKind1", Group: "velero.io", Verbs: metav1.Verbs{"create", "update", "delete"}},
			}, GroupVersion: "velero.io/v2"},
		}

		discoveryMock.EXPECT().ServerPreferredResources().Return(resources, nil)
		sut := &defaultCleanupManager{discoveryClient: discoveryMock}

		result, _ := sut.findResources()

		assert.Equal(t, 1, len(result))
		assert.Equal(t, "MyKind", result[0].Kind)
	})

	t.Run("should propagate errors while fetching resources", func(t *testing.T) {
		discoveryMock := newMockDiscoveryInterface(t)
		discoveryMock.EXPECT().ServerPreferredResources().Return([]*metav1.APIResourceList{}, assert.AnError)

		sut := &defaultCleanupManager{discoveryClient: discoveryMock}

		_, err := sut.findResources()

		assert.Error(t, err)
		assert.ErrorContains(t, err, "fetching supported resources")

	})
}

func TestFindObjects(t *testing.T) {
	t.Run("only object with label app=ces and k8s.cloudogu.com/part-of != 'backup' should be deleted", func(t *testing.T) {
		ctx := context.TODO()

		resources := []*metav1.APIResourceList{
			{APIResources: []metav1.APIResource{
				{Kind: "MyKind", Group: "k8s.example.com", Version: "v2", Verbs: metav1.Verbs{"create", "update", "delete"}},
			}, GroupVersion: "k8s.example.com/v2"},
		}

		discoveryMock := newMockDiscoveryInterface(t)
		discoveryMock.EXPECT().ServerPreferredResources().Return(resources, nil)

		groupVersionKind := schema.GroupVersionKind{Group: "k8s.example.com", Version: "v2", Kind: "MyKind"}
		objectList := &unstructured.UnstructuredList{}
		objectList.SetGroupVersionKind(groupVersionKind)

		selector, _ := metav1.LabelSelectorAsSelector(defaultCleanupSelector)
		listOptions := client.ListOptions{LabelSelector: &client.MatchingLabelsSelector{Selector: selector}}

		clientMock := newMockK8sClient(t)
		clientMock.EXPECT().List(ctx, objectList, &listOptions).Return(nil)

		sut := &defaultCleanupManager{discoveryClient: discoveryMock, client: clientMock}

		sut.findObjects(ctx, defaultCleanupSelector)
	})

	t.Run("should return objects", func(t *testing.T) {
		ctx := context.TODO()

		resources := []*metav1.APIResourceList{
			{APIResources: []metav1.APIResource{
				{Kind: "MyKind", Group: "k8s.example.com", Version: "v2", Verbs: metav1.Verbs{"create", "update", "delete"}},
			}, GroupVersion: "k8s.example.com/v2"},
		}

		discoveryMock := newMockDiscoveryInterface(t)
		discoveryMock.EXPECT().ServerPreferredResources().Return(resources, nil)

		groupVersionKind := schema.GroupVersionKind{Group: "k8s.example.com", Version: "v2", Kind: "MyKind"}
		objectList := &unstructured.UnstructuredList{}
		objectList.SetGroupVersionKind(groupVersionKind)

		selector, _ := metav1.LabelSelectorAsSelector(defaultCleanupSelector)
		listOptions := client.ListOptions{LabelSelector: &client.MatchingLabelsSelector{Selector: selector}}

		clientMock := newMockK8sClient(t)
		clientMock.EXPECT().List(ctx, objectList, &listOptions).RunAndReturn(func(ctx context.Context, list client.ObjectList, option ...client.ListOption) error {
			t.Helper()
			ul, _ := list.(*unstructured.UnstructuredList)
			item := unstructured.Unstructured{}
			item.SetGroupVersionKind(groupVersionKind)
			item.SetName("MyKindObject")
			ul.Items = []unstructured.Unstructured{item}
			return nil
		})

		sut := &defaultCleanupManager{discoveryClient: discoveryMock, client: clientMock}

		objects, err := sut.findObjects(ctx, defaultCleanupSelector)

		assert.NoError(t, err)
		assert.Equal(t, 1, len(objects))
		assert.Equal(t, "MyKindObject", objects[0].GetName())
	})

	t.Run("should propagate error while find resources", func(t *testing.T) {
		ctx := context.TODO()

		discoveryMock := newMockDiscoveryInterface(t)
		discoveryMock.EXPECT().ServerPreferredResources().Return(nil, assert.AnError)

		sut := &defaultCleanupManager{discoveryClient: discoveryMock}

		_, err := sut.findObjects(ctx, nil)

		assert.Error(t, err)
		assert.ErrorContains(t, err, "find resources")
	})

	t.Run("should propagate error while list objects", func(t *testing.T) {
		ctx := context.TODO()

		resources := []*metav1.APIResourceList{
			{APIResources: []metav1.APIResource{
				{Kind: "MyKind", Group: "k8s.example.com", Version: "v2", Verbs: metav1.Verbs{"create", "update", "delete"}},
			}, GroupVersion: "k8s.example.com/v2"},
		}

		discoveryMock := newMockDiscoveryInterface(t)
		discoveryMock.EXPECT().ServerPreferredResources().Return(resources, nil)

		groupVersionKind := schema.GroupVersionKind{Group: "k8s.example.com", Version: "v2", Kind: "MyKind"}
		objectList := &unstructured.UnstructuredList{}
		objectList.SetGroupVersionKind(groupVersionKind)

		selector, _ := metav1.LabelSelectorAsSelector(defaultCleanupSelector)
		listOptions := client.ListOptions{LabelSelector: &client.MatchingLabelsSelector{Selector: selector}}

		clientMock := newMockK8sClient(t)
		clientMock.EXPECT().List(ctx, objectList, &listOptions).Return(assert.AnError)

		sut := &defaultCleanupManager{discoveryClient: discoveryMock, client: clientMock}

		_, err := sut.findObjects(ctx, defaultCleanupSelector)

		assert.Error(t, err)
		assert.ErrorContains(t, err, "list objects of resource")
	})

}

func TestDeleteObject(t *testing.T) {
	t.Run("should delete object", func(t *testing.T) {
		ctx := context.TODO()
		clientMock := newMockK8sClient(t)

		propagationPolicy := metav1.DeletePropagationBackground
		deleteOptions := client.DeleteOptions{PropagationPolicy: &propagationPolicy}
		item := unstructured.Unstructured{}

		clientMock.EXPECT().Delete(ctx, &item, &deleteOptions).Return(nil)

		sut := &defaultCleanupManager{client: clientMock}

		err := sut.deleteObject(ctx, &item)

		assert.NoError(t, err)
	})

}

func TestExistObject(t *testing.T) {
	t.Run("should return false if object was not found", func(t *testing.T) {
		ctx := context.TODO()

		clientMock := newMockK8sClient(t)

		object := unstructured.Unstructured{}
		object.SetNamespace("ns")
		object.SetName("aName")
		objectKey := types.NamespacedName{Namespace: "ns", Name: "aName"}

		e := errors.NewNotFound(schema.GroupResource{Group: "example.com", Resource: "Pod"}, "aName")
		clientMock.EXPECT().Get(ctx, objectKey, &object).Return(e)

		sut := &defaultCleanupManager{client: clientMock}

		result := sut.existObject(ctx, &object)

		assert.False(t, result)

	})
}

func TestRemoveFinalizer(t *testing.T) {
	t.Run("should retry if a update conflict occurred", func(t *testing.T) {
		t.Skip("How to test retry.OnConflict calls?")

		ctx := context.TODO()
		object := unstructured.Unstructured{}
		object.SetNamespace("ns")
		object.SetName("aName")
		objectKey := types.NamespacedName{Namespace: "ns", Name: "aName"}

		clientMock := newMockK8sClient(t)
		resource := schema.GroupResource{Group: "example.com", Resource: "Pod"}

		clientMock.EXPECT().Get(ctx, objectKey, &object).Return(nil)
		clientMock.EXPECT().Update(ctx, &object).Return(errors.NewConflict(resource, "aName", assert.AnError))
		clientMock.EXPECT().Get(ctx, objectKey, &object).Return(nil)
		clientMock.EXPECT().Update(ctx, &object).Return(nil)

		sut := &defaultCleanupManager{client: clientMock}

		err := sut.removeFinalizers(ctx, &object)

		assert.NoError(t, err)
	})
}
