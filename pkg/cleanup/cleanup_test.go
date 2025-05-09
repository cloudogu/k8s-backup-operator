package cleanup

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestCleanUp(t *testing.T) {
	t.Run("should successfully clean up object", func(t *testing.T) {
		ctx := context.TODO()

		resources := []*metav1.APIResourceList{
			{APIResources: []metav1.APIResource{
				{Kind: "MyKind", Group: "k8s.example.com", Version: "v2", Verbs: metav1.Verbs{"create", "update", "delete"}},
			}, GroupVersion: "k8s.example.com/v2"},
		}

		discoveryMock := newMockDiscoveryInterface(t)
		discoveryMock.EXPECT().ServerPreferredResources().Return(resources, nil)

		object := unstructured.Unstructured{}
		object.SetNamespace("ns")
		object.SetName("aName")

		groupVersionKind := schema.GroupVersionKind{Group: "k8s.example.com", Version: "v2", Kind: "MyKind"}
		objectList := &unstructured.UnstructuredList{}
		objectList.SetGroupVersionKind(groupVersionKind)

		selector, _ := metav1.LabelSelectorAsSelector(defaultCleanupSelector)
		listOptions := client.ListOptions{LabelSelector: &client.MatchingLabelsSelector{Selector: selector}}

		clientMock := newMockK8sClient(t)
		clientMock.EXPECT().List(ctx, objectList, &listOptions).RunAndReturn(
			func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
				c := list.(*unstructured.UnstructuredList)
				c.Items = []unstructured.Unstructured{object}
				return nil
			})

		propagationPolicy := metav1.DeletePropagationBackground
		deleteOptions := client.DeleteOptions{PropagationPolicy: &propagationPolicy}
		clientMock.EXPECT().Delete(ctx, &object, &deleteOptions).Return(nil)

		notFoundError := errors.NewNotFound(schema.GroupResource{Group: "example.com", Resource: "Pod"}, "aName")
		clientMock.EXPECT().Get(ctx, mock.Anything, mock.Anything).Return(notFoundError)

		configMap := &corev1.ConfigMap{
			Data: map[string]string{
				"cleanup": `
exclude:
- group: k8s.example.com
  version: v2
  kind: MyKind
  name: nothing
`,
			},
		}
		configMapClientMock := newMockConfigMapClient(t)
		configMapClientMock.EXPECT().Get(ctx, "k8s-backup-operator-cleanup-exclude", mock.Anything).Return(configMap, nil)

		sut := &defaultCleanupManager{discoveryClient: discoveryMock, client: clientMock, configMapClient: configMapClientMock}

		err := sut.Cleanup(ctx)
		assert.NoError(t, err)
	})
}

func TestFindResources(t *testing.T) {
	t.Run("should only find resources that can be deleted", func(t *testing.T) {
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
			}, GroupVersion: "k8s.example.com/v1"},
			{APIResources: []metav1.APIResource{
				{Kind: "MyKind", Verbs: metav1.Verbs{"create", "update", "delete"}},
			}, GroupVersion: "k8s.example.com/v2"},
			{APIResources: []metav1.APIResource{
				{Kind: "VeleroKind1", Group: "velero.io", Verbs: metav1.Verbs{"create", "update", "delete"}},
				{Kind: "VeleroKind2", Group: "velero.io", Verbs: metav1.Verbs{"create", "update", "delete"}},
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

		configMapClientMock := newMockConfigMapClient(t)
		configMapClientMock.EXPECT().Get(ctx, "k8s-backup-operator-cleanup-exclude", mock.Anything).Return(nil, assert.AnError)

		sut := &defaultCleanupManager{discoveryClient: discoveryMock, client: clientMock, configMapClient: configMapClientMock}

		_, _ = sut.findObjects(ctx, defaultCleanupSelector)
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

		configMap := &corev1.ConfigMap{
			Data: map[string]string{
				"cleanup": `
exclude:
- group: k8s.example.com
  version: v2
  kind: MyKind
  name: nothing
`,
			},
		}
		configMapClientMock := newMockConfigMapClient(t)
		configMapClientMock.EXPECT().Get(ctx, "k8s-backup-operator-cleanup-exclude", mock.Anything).Return(configMap, nil)

		sut := &defaultCleanupManager{discoveryClient: discoveryMock, client: clientMock, configMapClient: configMapClientMock}

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

	t.Run("should not return an error if resource was not found", func(t *testing.T) {
		ctx := context.TODO()
		clientMock := newMockK8sClient(t)

		propagationPolicy := metav1.DeletePropagationBackground
		deleteOptions := client.DeleteOptions{PropagationPolicy: &propagationPolicy}
		item := unstructured.Unstructured{}

		e := errors.NewNotFound(schema.GroupResource{Group: "example.com", Resource: "Pod"}, "aName")
		clientMock.EXPECT().Delete(ctx, &item, &deleteOptions).Return(e)

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
		ctx := context.TODO()
		object := unstructured.Unstructured{}
		object.SetNamespace("ns")
		object.SetName("aName")
		objectKey := types.NamespacedName{Namespace: "ns", Name: "aName"}

		clientMock := newMockK8sClient(t)
		resource := schema.GroupResource{Group: "example.com", Resource: "Pod"}

		clientMock.EXPECT().Get(ctx, objectKey, &object).Return(nil).Twice()
		clientMock.EXPECT().Update(ctx, &object).Return(errors.NewConflict(resource, "aName", assert.AnError)).Once()
		clientMock.EXPECT().Update(ctx, &object).Return(nil).Once()

		sut := &defaultCleanupManager{client: clientMock}

		err := sut.removeFinalizers(ctx, &object)

		assert.NoError(t, err)
	})
	t.Run("should remove finalizers", func(t *testing.T) {
		ctx := context.TODO()
		object := unstructured.Unstructured{}
		object.SetNamespace("ns")
		object.SetName("aName")
		object.SetFinalizers([]string{"myFinalizer"})
		objectKey := types.NamespacedName{Namespace: "ns", Name: "aName"}

		clientMock := newMockK8sClient(t)
		resource := schema.GroupResource{Group: "example.com", Resource: "Pod"}

		clientMock.EXPECT().Get(ctx, objectKey, &object).Return(nil).Twice()
		clientMock.EXPECT().Update(ctx, &object).Return(errors.NewConflict(resource, "aName", assert.AnError)).Once()
		clientMock.EXPECT().Update(ctx, &object).Return(nil).Once()

		sut := &defaultCleanupManager{client: clientMock}

		err := sut.removeFinalizers(ctx, &object)

		assert.NoError(t, err)
		assert.Equal(t, []string{}, object.GetFinalizers())
	})
	t.Run("should not remove kubernetes.io finalizers", func(t *testing.T) {
		ctx := context.TODO()
		object := unstructured.Unstructured{}
		object.SetNamespace("ns")
		object.SetName("aName")
		object.SetFinalizers([]string{"kubernetes.io/myFinalizer"})
		objectKey := types.NamespacedName{Namespace: "ns", Name: "aName"}

		clientMock := newMockK8sClient(t)
		resource := schema.GroupResource{Group: "example.com", Resource: "Pod"}

		clientMock.EXPECT().Get(ctx, objectKey, &object).Return(nil).Twice()
		clientMock.EXPECT().Update(ctx, &object).Return(errors.NewConflict(resource, "aName", assert.AnError)).Once()
		clientMock.EXPECT().Update(ctx, &object).Return(nil).Once()

		sut := &defaultCleanupManager{client: clientMock}

		err := sut.removeFinalizers(ctx, &object)

		assert.NoError(t, err)
		assert.Equal(t, []string{"kubernetes.io/myFinalizer"}, object.GetFinalizers())
	})
}

func TestWaitForObjectToBeDeleted(t *testing.T) {
	t.Run("should not wait if a object is not found", func(t *testing.T) {
		ctx := context.TODO()
		object := unstructured.Unstructured{}
		object.SetNamespace("ns")
		object.SetName("aName")
		objectKey := types.NamespacedName{Namespace: "ns", Name: "aName"}

		clientMock := newMockK8sClient(t)

		notFoundError := errors.NewNotFound(schema.GroupResource{Group: "example.com", Resource: "Pod"}, "aName")
		clientMock.EXPECT().Get(ctx, objectKey, &object).Return(notFoundError)

		sut := &defaultCleanupManager{client: clientMock}
		var wg sync.WaitGroup
		sut.waitForObjectToBeDeleted(ctx, &object, &wg)
		wg.Wait()
	})
	t.Run("should wait if a object still exists", func(t *testing.T) {
		ctx := context.TODO()
		object := unstructured.Unstructured{}
		object.SetNamespace("ns")
		object.SetName("aName")
		objectKey := types.NamespacedName{Namespace: "ns", Name: "aName"}

		clientMock := newMockK8sClient(t)

		notFoundError := errors.NewNotFound(schema.GroupResource{Group: "example.com", Resource: "Pod"}, "aName")
		clientMock.EXPECT().Get(ctx, objectKey, &object).Return(nil).Once()
		clientMock.EXPECT().Get(ctx, objectKey, &object).Return(notFoundError).Once()

		sut := &defaultCleanupManager{client: clientMock}
		var wg sync.WaitGroup
		sut.waitForObjectToBeDeleted(ctx, &object, &wg)
		wg.Wait()
	})
}

func Test_filterObjects(t *testing.T) {
	type args struct {
		objects   []unstructured.Unstructured
		toExclude []ExcludeEntry
	}
	tests := []struct {
		name string
		args args
		want []unstructured.Unstructured
	}{
		{
			name: "filter ces-loadbalancer service by kind and Name",
			args: args{
				objects: []unstructured.Unstructured{
					{Object: map[string]interface{}{"apiVersion": "v1", "kind": "Service", "metadata": map[string]interface{}{"name": "ces-loadbalancer"}}},
					{Object: map[string]interface{}{"apiVersion": "v1", "kind": "Pod", "metadata": map[string]interface{}{"name": "test"}}},
				},
				toExclude: []ExcludeEntry{
					{
						Name:    "ces-loadbalancer",
						Group:   "*",
						Version: "*",
						Kind:    "Service",
					},
				},
			},
			want: []unstructured.Unstructured{{Object: map[string]interface{}{"apiVersion": "v1", "kind": "Pod", "metadata": map[string]interface{}{"name": "test"}}}},
		},
		{
			name: "filter ces-loadbalancer by Name",
			args: args{
				objects: []unstructured.Unstructured{
					{Object: map[string]interface{}{"apiVersion": "v1", "kind": "Service", "metadata": map[string]interface{}{"name": "ces-loadbalancer"}}},
					{Object: map[string]interface{}{"apiVersion": "v1", "kind": "Pod", "metadata": map[string]interface{}{"name": "test"}}},
				},
				toExclude: []ExcludeEntry{
					{
						Name:    "ces-loadbalancer",
						Group:   "*",
						Version: "*",
						Kind:    "*",
					},
				},
			},
			want: []unstructured.Unstructured{{Object: map[string]interface{}{"apiVersion": "v1", "kind": "Pod", "metadata": map[string]interface{}{"name": "test"}}}},
		},
		{
			name: "gvks to exclude are not in objects",
			args: args{
				objects: []unstructured.Unstructured{
					{Object: map[string]interface{}{"apiVersion": "v1", "kind": "Service", "metadata": map[string]interface{}{"name": "ces-loadbalancer-service"}}},
					{Object: map[string]interface{}{"apiVersion": "v1", "kind": "Pod", "metadata": map[string]interface{}{"name": "test"}}},
				},
				toExclude: []ExcludeEntry{
					{
						Name:    "ces-loadbalancer",
						Group:   "*",
						Version: "*",
						Kind:    "*",
					},
				},
			},
			want: []unstructured.Unstructured{
				{Object: map[string]interface{}{"apiVersion": "v1", "kind": "Service", "metadata": map[string]interface{}{"name": "ces-loadbalancer-service"}}},
				{Object: map[string]interface{}{"apiVersion": "v1", "kind": "Pod", "metadata": map[string]interface{}{"name": "test"}}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, filterObjects(tt.args.objects, tt.args.toExclude), "filterObjects(%v, %v)", tt.args.objects, tt.args.toExclude)
		})
	}
}

func Test_defaultCleanupManager_readEntriesToExclude(t *testing.T) {
	tests := []struct {
		name              string
		configMapClientFn func(t *testing.T) configMapClient
		want              []ExcludeEntry
		wantErr           assert.ErrorAssertionFunc
	}{
		{
			name: "should return nothing if configmap is not found",
			configMapClientFn: func(t *testing.T) configMapClient {
				cmMock := newMockConfigMapClient(t)
				cmMock.EXPECT().Get(mock.Anything, "k8s-backup-operator-cleanup-exclude", metav1.GetOptions{}).
					Return(nil, errors.NewNotFound(schema.GroupResource{}, "k8s-backup-operator-cleanup-exclude"))
				return cmMock
			},
			want:    nil,
			wantErr: assert.NoError,
		},
		{
			name: "should fail for any other error when getting configmap",
			configMapClientFn: func(t *testing.T) configMapClient {
				cmMock := newMockConfigMapClient(t)
				cmMock.EXPECT().Get(mock.Anything, "k8s-backup-operator-cleanup-exclude", metav1.GetOptions{}).
					Return(nil, assert.AnError)
				return cmMock
			},
			want: nil,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, assert.AnError, i) &&
					assert.ErrorContains(t, err, "failed to get cleanup-exclude config", i)
			},
		},
		{
			name: "should fail to find cleanup key in configmap",
			configMapClientFn: func(t *testing.T) configMapClient {
				cmMock := newMockConfigMapClient(t)
				configMap := &corev1.ConfigMap{}
				cmMock.EXPECT().Get(mock.Anything, "k8s-backup-operator-cleanup-exclude", metav1.GetOptions{}).
					Return(configMap, nil)
				return cmMock
			},
			want: nil,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorContains(t, err, "cleanup-exclude config did not contain key \"cleanup\"", i)
			},
		},
		{
			name: "should fail to unmarshal cleanup in configmap",
			configMapClientFn: func(t *testing.T) configMapClient {
				cmMock := newMockConfigMapClient(t)
				configMap := &corev1.ConfigMap{
					Data: map[string]string{"cleanup": "{{"},
				}
				cmMock.EXPECT().Get(mock.Anything, "k8s-backup-operator-cleanup-exclude", metav1.GetOptions{}).
					Return(configMap, nil)
				return cmMock
			},
			want: nil,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorContains(t, err, "failed to unmarshal cleanup-exclude config", i)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &defaultCleanupManager{
				configMapClient: tt.configMapClientFn(t),
			}
			ctx := context.TODO()
			got, err := c.readEntriesToExclude(ctx)
			if !tt.wantErr(t, err, fmt.Sprintf("readEntriesToExclude(%v)", ctx)) {
				return
			}
			assert.Equalf(t, tt.want, got, "readEntriesToExclude(%v)", ctx)
		})
	}
}
