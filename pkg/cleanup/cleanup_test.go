package cleanup

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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
			{Kind: "Example2", Verbs: metav1.Verbs{"create", "update", "delete"}},
			{Namespaced: false, Kind: "MyClusterScopedObject", Verbs: metav1.Verbs{"create", "update", "delete"}},
		}, GroupVersion: "k8s.example.com/v2"},
		{APIResources: []metav1.APIResource{
			{Kind: "Example3", Verbs: metav1.Verbs{"create", "delete"}},
		}, GroupVersion: "k8s.example.com/v3"},
	}

	expectedExample2 := &unstructured.Unstructured{Object: map[string]interface{}{"apiVersion": "k8s.example.com/v2", "kind": "Example2"}}
	expectedExample3 := &unstructured.Unstructured{Object: map[string]interface{}{"apiVersion": "k8s.example.com/v3", "kind": "Example3"}}
	expectedMyNamespacedObject := &unstructured.Unstructured{Object: map[string]interface{}{"apiVersion": "k8s.example.com/v1", "kind": "MyNamespacedObject"}}
	expectedMyClusterScopedObject := &unstructured.Unstructured{Object: map[string]interface{}{"apiVersion": "k8s.example.com/v2", "kind": "MyClusterScopedObject"}}
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
	t.Run("should fail on deletion", func(t *testing.T) {
		// given
		clientMock := newMockK8sClient(t)
		clientMock.EXPECT().DeleteAllOf(testCtx, expectedExample2, mock.Anything).Return(assert.AnError)
		clientMock.EXPECT().DeleteAllOf(testCtx, expectedExample3, mock.Anything).Return(assert.AnError)
		clientMock.EXPECT().DeleteAllOf(testCtx, expectedMyNamespacedObject, mock.Anything).Return(assert.AnError)
		clientMock.EXPECT().DeleteAllOf(testCtx, expectedMyClusterScopedObject, mock.Anything).Return(assert.AnError)

		discoveryMock := newMockDiscoveryInterface(t)
		discoveryMock.EXPECT().ServerPreferredResources().Return(serverKnownResources, nil)

		sut := &defaultCleanupManager{
			namespace:       testNamespace,
			client:          clientMock,
			discoveryClient: discoveryMock,
		}

		// when
		err := sut.Cleanup(testCtx)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to delete api resources with label selector \"app in (ces),k8s.cloudogu.com/part-of notin (backup)\"")
		assert.ErrorContains(t, err, "failed delete for group version k8s.example.com/v1")
		assert.ErrorContains(t, err, "failed delete for kind MyNamespacedObject in namespace test-ns")
		assert.ErrorContains(t, err, "failed delete for group version k8s.example.com/v2")
		assert.ErrorContains(t, err, "failed delete for kind Example2")
		assert.ErrorContains(t, err, "failed delete for kind MyClusterScopedObject")
		assert.ErrorContains(t, err, "failed delete for group version k8s.example.com/v3")
		assert.ErrorContains(t, err, "failed delete for kind Example3")
	})
	t.Run("should succeed on deletion", func(t *testing.T) {
		// given
		clientMock := newMockK8sClient(t)
		clientMock.EXPECT().DeleteAllOf(testCtx, expectedExample2, mock.Anything).Return(nil)
		clientMock.EXPECT().DeleteAllOf(testCtx, expectedExample3, mock.Anything).Return(nil)
		clientMock.EXPECT().DeleteAllOf(testCtx, expectedMyNamespacedObject, mock.Anything).Return(nil)
		clientMock.EXPECT().DeleteAllOf(testCtx, expectedMyClusterScopedObject, mock.Anything).Return(nil)

		discoveryMock := newMockDiscoveryInterface(t)
		discoveryMock.EXPECT().ServerPreferredResources().Return(serverKnownResources, nil)

		sut := &defaultCleanupManager{
			namespace:       testNamespace,
			client:          clientMock,
			discoveryClient: discoveryMock,
		}

		// when
		err := sut.Cleanup(testCtx)

		// then
		require.NoError(t, err)
	})
}
