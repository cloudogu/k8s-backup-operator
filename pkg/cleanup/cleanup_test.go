package cleanup

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
)

const testNamespace = "test-ns"

var testCtx = context.TODO()

func TestNewManager(t *testing.T) {
	actual := NewManager(testNamespace, nil, nil)
	require.NotEmpty(t, actual)
}

func Test_defaultCleanupManager_Cleanup(t *testing.T) {
	serverKnownResources := []*metav1.APIResourceList{
		{APIResources: []metav1.APIResource{
			{Kind: "Example", Verbs: metav1.Verbs{"create", "update"}},
			{Kind: "MyObject", Verbs: metav1.Verbs{"create", "update"}},
			{Namespaced: true, Kind: "MyNamespacedObject", Verbs: metav1.Verbs{"create", "update", "delete"}},
		}, GroupVersion: "k8s.example.com/v1"},
		{APIResources: []metav1.APIResource{
			{Namespaced: false, Kind: "MyClusterScopedObject", Verbs: metav1.Verbs{"create", "update", "delete"}},
		}, GroupVersion: "k8s.example.com/v2"},
		{APIResources: []metav1.APIResource{
			{Namespaced: true, Kind: "Pod", Verbs: metav1.Verbs{"create", "update", "delete", "list", "delete", "patch", "get"}},
		}, GroupVersion: "v1"},
		{APIResources: []metav1.APIResource{
			{Namespaced: false, Kind: "CustomResourceDefinition", Verbs: metav1.Verbs{"create", "update", "delete", "list", "delete", "patch", "get"}},
		}, GroupVersion: "apiextensions.k8s.io/v1"},
	}
	t.Run("should fail to list the servers known api resources", func(t *testing.T) {
		// given
		discoveryMock := newMockDiscoveryInterface(t)
		discoveryMock.EXPECT().ServerPreferredResources().Return(nil, assert.AnError)
		sut := &defaultCleanupManager{discoveryClient: discoveryMock}

		// when
		err := sut.Cleanup(testCtx)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to get resource lists from server")
	})
	t.Run("should fail to create label selector", func(t *testing.T) {
		// given
		originalSelector := *defaultCleanupSelector
		defer func() { defaultCleanupSelector = &originalSelector }()
		defaultCleanupSelector = &metav1.LabelSelector{MatchExpressions: []metav1.LabelSelectorRequirement{{Operator: "br0k€n"}}}

		discoveryMock := newMockDiscoveryInterface(t)
		discoveryMock.EXPECT().ServerPreferredResources().Return(nil, nil)
		sut := &defaultCleanupManager{discoveryClient: discoveryMock}

		// when
		err := sut.Cleanup(testCtx)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to create selector from given label selector &LabelSelector{MatchLabels:map[string]string{},MatchExpressions:[]LabelSelectorRequirement{LabelSelectorRequirement{Key:,Operator:br0k€n,Values:[],},},}")
	})
	t.Run("should return early if api resources for a group are empty", func(t *testing.T) {
		// given
		emptyListList := []*metav1.APIResourceList{{APIResources: make([]metav1.APIResource, 0)}}
		discoveryMock := newMockDiscoveryInterface(t)
		discoveryMock.EXPECT().ServerPreferredResources().Return(emptyListList, nil)
		sut := &defaultCleanupManager{discoveryClient: discoveryMock}

		// when
		err := sut.Cleanup(testCtx)

		// then
		require.NoError(t, err)
	})
	t.Run("should return early if group version of api resources is not parsable", func(t *testing.T) {
		// given
		emptyListList := []*metav1.APIResourceList{{GroupVersion: "k8s/example/com/invalid/v1", APIResources: make([]metav1.APIResource, 1)}}
		discoveryMock := newMockDiscoveryInterface(t)
		discoveryMock.EXPECT().ServerPreferredResources().Return(emptyListList, nil)
		sut := &defaultCleanupManager{discoveryClient: discoveryMock}

		// when
		err := sut.Cleanup(testCtx)

		// then
		require.NoError(t, err)
	})
	t.Run("should fail to list objects", func(t *testing.T) {
		// given

		discoveryMock := newMockDiscoveryInterface(t)
		discoveryMock.EXPECT().ServerPreferredResources().Return(serverKnownResources, nil)

		clientMock := newMockK8sClient(t)
		expectedMyNamespacedObject, expectedMyClusterScopedObject := initTestLists()
		clientMock.EXPECT().List(testCtx, expectedMyNamespacedObject, mock.Anything).Return(assert.AnError)
		clientMock.EXPECT().List(testCtx, expectedMyClusterScopedObject, mock.Anything).Return(assert.AnError)

		sut := &defaultCleanupManager{client: clientMock, discoveryClient: discoveryMock}

		// when
		err := sut.Cleanup(testCtx)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to delete api resources with label selector \"app in (ces),k8s.cloudogu.com/part-of notin (backup)\"")
		assert.ErrorContains(t, err, "failed to list objects in k8s.example.com/v1, Kind=MyNamespacedObject")
		assert.ErrorContains(t, err, "failed to list objects in k8s.example.com/v2, Kind=MyClusterScopedObject")
	})
	t.Run("should fail to get objects when removing finalizer", func(t *testing.T) {
		// given
		discoveryMock := newMockDiscoveryInterface(t)
		discoveryMock.EXPECT().ServerPreferredResources().Return(serverKnownResources, nil)

		clientMock := newMockK8sClient(t)
		clientMock.EXPECT().List(testCtx, mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, list client.ObjectList, option ...client.ListOption) error {
			t.Helper()
			ul, ok := list.(*unstructured.UnstructuredList)
			require.True(t, ok, "Type of list should be UnstructuredList")
			fillTestList(ul)
			return nil
		})
		expectGet(clientMock, assert.AnError)
		expectDelete(clientMock, false, nil)

		sut := &defaultCleanupManager{client: clientMock, discoveryClient: discoveryMock}

		// when
		err := sut.Cleanup(testCtx)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to delete api resources with label selector \"app in (ces),k8s.cloudogu.com/part-of notin (backup)\"")
		assert.ErrorContains(t, err, "failed to remove finalizers for test-ns/item1 (k8s.example.com/v1, Kind=MyNamespacedObject)")
		assert.ErrorContains(t, err, "failed to remove finalizers for test-ns/item2 (k8s.example.com/v1, Kind=MyNamespacedObject)")
		assert.ErrorContains(t, err, "failed to remove finalizers for /item1 (k8s.example.com/v2, Kind=MyClusterScopedObject)")
		assert.ErrorContains(t, err, "failed to remove finalizers for /item2 (k8s.example.com/v2, Kind=MyClusterScopedObject)")
	})
	t.Run("should not fail if not found", func(t *testing.T) {
		// given
		discoveryMock := newMockDiscoveryInterface(t)
		discoveryMock.EXPECT().ServerPreferredResources().Return(serverKnownResources, nil)

		clientMock := newMockK8sClient(t)
		clientMock.EXPECT().List(testCtx, mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, list client.ObjectList, option ...client.ListOption) error {
			t.Helper()
			ul, ok := list.(*unstructured.UnstructuredList)
			require.True(t, ok, "Type of list should be UnstructuredList")
			fillTestList(ul)
			return nil
		})
		notFoundErr := errors.NewNotFound(schema.GroupResource{}, "")
		expectGet(clientMock, notFoundErr)
		expectDelete(clientMock, false, notFoundErr)

		sut := &defaultCleanupManager{client: clientMock, discoveryClient: discoveryMock}

		// when
		err := sut.Cleanup(testCtx)

		// then
		require.NoError(t, err)
	})
	t.Run("should fail to update objects when removing finalizer", func(t *testing.T) {
		// given
		discoveryMock := newMockDiscoveryInterface(t)
		discoveryMock.EXPECT().ServerPreferredResources().Return(serverKnownResources, nil)

		clientMock := newMockK8sClient(t)
		clientMock.EXPECT().List(testCtx, mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, list client.ObjectList, option ...client.ListOption) error {
			t.Helper()
			ul, ok := list.(*unstructured.UnstructuredList)
			require.True(t, ok, "Type of list should be UnstructuredList")
			fillTestList(ul)
			return nil
		})
		expectGet(clientMock, nil)
		expectUpdate(clientMock, assert.AnError)
		expectDelete(clientMock, true, nil)

		sut := &defaultCleanupManager{client: clientMock, discoveryClient: discoveryMock}

		// when
		err := sut.Cleanup(testCtx)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to delete api resources with label selector \"app in (ces),k8s.cloudogu.com/part-of notin (backup)\"")
		assert.ErrorContains(t, err, "failed to remove finalizers for test-ns/item1 (k8s.example.com/v1, Kind=MyNamespacedObject)")
		assert.ErrorContains(t, err, "failed to remove finalizers for test-ns/item2 (k8s.example.com/v1, Kind=MyNamespacedObject)")
		assert.ErrorContains(t, err, "failed to remove finalizers for /item1 (k8s.example.com/v2, Kind=MyClusterScopedObject)")
		assert.ErrorContains(t, err, "failed to remove finalizers for /item2 (k8s.example.com/v2, Kind=MyClusterScopedObject)")
	})
	t.Run("should fail to delete objects", func(t *testing.T) {
		// given
		discoveryMock := newMockDiscoveryInterface(t)
		discoveryMock.EXPECT().ServerPreferredResources().Return(serverKnownResources, nil)

		clientMock := newMockK8sClient(t)
		clientMock.EXPECT().List(testCtx, mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, list client.ObjectList, option ...client.ListOption) error {
			t.Helper()
			ul, ok := list.(*unstructured.UnstructuredList)
			require.True(t, ok, "Type of list should be UnstructuredList")
			fillTestList(ul)
			return nil
		})
		expectGet(clientMock, nil)
		expectUpdate(clientMock, nil)
		expectDelete(clientMock, true, assert.AnError)

		sut := &defaultCleanupManager{client: clientMock, discoveryClient: discoveryMock}

		// when
		err := sut.Cleanup(testCtx)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to delete api resources with label selector \"app in (ces),k8s.cloudogu.com/part-of notin (backup)\"")
		assert.ErrorContains(t, err, "failed to delete test-ns/item1 (k8s.example.com/v1, Kind=MyNamespacedObject)")
		assert.ErrorContains(t, err, "failed to delete test-ns/item2 (k8s.example.com/v1, Kind=MyNamespacedObject)")
		assert.ErrorContains(t, err, "failed to delete /item1 (k8s.example.com/v2, Kind=MyClusterScopedObject)")
		assert.ErrorContains(t, err, "failed to delete /item2 (k8s.example.com/v2, Kind=MyClusterScopedObject)")
	})
	t.Run("should succeed to delete objects", func(t *testing.T) {
		// given
		discoveryMock := newMockDiscoveryInterface(t)
		discoveryMock.EXPECT().ServerPreferredResources().Return(serverKnownResources, nil)

		clientMock := newMockK8sClient(t)
		clientMock.EXPECT().List(testCtx, mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, list client.ObjectList, option ...client.ListOption) error {
			t.Helper()
			ul, ok := list.(*unstructured.UnstructuredList)
			require.True(t, ok, "Type of list should be UnstructuredList")
			fillTestList(ul)
			return nil
		})
		expectGet(clientMock, nil)
		expectUpdate(clientMock, nil)
		expectDelete(clientMock, true, nil)

		sut := &defaultCleanupManager{client: clientMock, discoveryClient: discoveryMock}

		// when
		err := sut.Cleanup(testCtx)

		// then
		require.NoError(t, err)
	})
}

func expectUpdate(clientMock *mockK8sClient, err error) {
	clientMock.EXPECT().Update(testCtx,
		&unstructured.Unstructured{Object: map[string]interface{}{"apiVersion": "k8s.example.com/v1", "kind": "MyNamespacedObject", "metadata": map[string]interface{}{"finalizers": []interface{}{}, "labels": map[string]interface{}{"app": "ces"}, "name": "item1", "namespace": "test-ns"}}},
	).Return(err)
	clientMock.EXPECT().Update(testCtx,
		&unstructured.Unstructured{Object: map[string]interface{}{"apiVersion": "k8s.example.com/v1", "kind": "MyNamespacedObject", "metadata": map[string]interface{}{"finalizers": []interface{}{}, "labels": map[string]interface{}{"app": "ces"}, "name": "item2", "namespace": "test-ns"}}},
	).Return(err)
	clientMock.EXPECT().Update(testCtx,
		&unstructured.Unstructured{Object: map[string]interface{}{"apiVersion": "k8s.example.com/v2", "kind": "MyClusterScopedObject", "metadata": map[string]interface{}{"finalizers": []interface{}{}, "labels": map[string]interface{}{"app": "ces"}, "name": "item1"}}},
	).Return(err)
	clientMock.EXPECT().Update(testCtx,
		&unstructured.Unstructured{Object: map[string]interface{}{"apiVersion": "k8s.example.com/v2", "kind": "MyClusterScopedObject", "metadata": map[string]interface{}{"finalizers": []interface{}{}, "labels": map[string]interface{}{"app": "ces"}, "name": "item2"}}},
	).Return(err)
}

func expectDelete(clientMock *mockK8sClient, withFinalizers bool, err error) {
	propagationPolicy := metav1.DeletePropagationBackground
	items := []*unstructured.Unstructured{
		{Object: map[string]interface{}{"apiVersion": "k8s.example.com/v1", "kind": "MyNamespacedObject", "metadata": map[string]interface{}{"labels": map[string]interface{}{"app": "ces"}, "name": "item1", "namespace": "test-ns"}}},
		{Object: map[string]interface{}{"apiVersion": "k8s.example.com/v1", "kind": "MyNamespacedObject", "metadata": map[string]interface{}{"finalizers": []interface{}{"my-finalizer"}, "labels": map[string]interface{}{"app": "ces"}, "name": "item2", "namespace": "test-ns"}}},
		{Object: map[string]interface{}{"apiVersion": "k8s.example.com/v2", "kind": "MyClusterScopedObject", "metadata": map[string]interface{}{"labels": map[string]interface{}{"app": "ces"}, "name": "item1"}}},
		{Object: map[string]interface{}{"apiVersion": "k8s.example.com/v2", "kind": "MyClusterScopedObject", "metadata": map[string]interface{}{"finalizers": []interface{}{"my-finalizer"}, "labels": map[string]interface{}{"app": "ces"}, "name": "item2"}}},
	}

	for _, item := range items {
		if withFinalizers {
			item.SetFinalizers(make([]string, 0))
		}
		clientMock.EXPECT().Delete(testCtx,
			item,
			&client.DeleteOptions{GracePeriodSeconds: (*int64)(nil), Preconditions: (*metav1.Preconditions)(nil), PropagationPolicy: &propagationPolicy, Raw: (*metav1.DeleteOptions)(nil), DryRun: []string(nil)},
		).Return(err)
	}
}

func expectGet(clientMock *mockK8sClient, err error) {
	clientMock.EXPECT().Get(testCtx, types.NamespacedName{Name: "item1", Namespace: testNamespace},
		&unstructured.Unstructured{Object: map[string]interface{}{"apiVersion": "k8s.example.com/v1", "kind": "MyNamespacedObject", "metadata": map[string]interface{}{"labels": map[string]interface{}{"app": "ces"}, "name": "item1", "namespace": "test-ns"}}},
	).Return(err)
	clientMock.EXPECT().Get(testCtx, types.NamespacedName{Name: "item2", Namespace: testNamespace},
		&unstructured.Unstructured{Object: map[string]interface{}{"apiVersion": "k8s.example.com/v1", "kind": "MyNamespacedObject", "metadata": map[string]interface{}{"finalizers": []interface{}{"my-finalizer"}, "labels": map[string]interface{}{"app": "ces"}, "name": "item2", "namespace": "test-ns"}}},
	).Return(err)
	clientMock.EXPECT().Get(testCtx, types.NamespacedName{Name: "item1"},
		&unstructured.Unstructured{Object: map[string]interface{}{"apiVersion": "k8s.example.com/v2", "kind": "MyClusterScopedObject", "metadata": map[string]interface{}{"labels": map[string]interface{}{"app": "ces"}, "name": "item1"}}},
	).Return(err)
	clientMock.EXPECT().Get(testCtx, types.NamespacedName{Name: "item2"},
		&unstructured.Unstructured{Object: map[string]interface{}{"apiVersion": "k8s.example.com/v2", "kind": "MyClusterScopedObject", "metadata": map[string]interface{}{"finalizers": []interface{}{"my-finalizer"}, "labels": map[string]interface{}{"app": "ces"}, "name": "item2"}}},
	).Return(err)
}

func initTestLists() (*unstructured.UnstructuredList, *unstructured.UnstructuredList) {
	expectedMyNamespacedObject := &unstructured.UnstructuredList{Object: map[string]interface{}{"apiVersion": "k8s.example.com/v1", "kind": "MyNamespacedObject"}}
	expectedMyClusterScopedObject := &unstructured.UnstructuredList{Object: map[string]interface{}{"apiVersion": "k8s.example.com/v2", "kind": "MyClusterScopedObject"}}
	return expectedMyNamespacedObject, expectedMyClusterScopedObject
}

func fillTestList(list *unstructured.UnstructuredList) {
	item1 := unstructured.Unstructured{}
	item1.SetGroupVersionKind(list.GroupVersionKind())
	item1.SetName("item1")
	item1.SetLabels(map[string]string{"app": "ces"})

	item2 := unstructured.Unstructured{}
	item2.SetGroupVersionKind(list.GroupVersionKind())
	item2.SetName("item2")
	item2.SetFinalizers([]string{"my-finalizer"})
	item2.SetLabels(map[string]string{"app": "ces"})

	if list.GetKind() == "MyNamespacedObject" {
		item1.SetNamespace(testNamespace)
		item2.SetNamespace(testNamespace)
	}

	list.Items = []unstructured.Unstructured{item1, item2}
}
