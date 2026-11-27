package cleanup

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	k8sErr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

const testNamespace = "ecosystem"

// newFakeClient creates a client like the one of the operator: core resources are known to the scheme,
// dogus only to the rest mapper, so they are handled as unstructured.
func newFakeClient(t *testing.T, funcs interceptor.Funcs, objects ...client.Object) client.WithWatch {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))

	restMapper := meta.NewDefaultRESTMapper(nil)
	restMapper.Add(doguGVK, meta.RESTScopeNamespace)
	for _, gvk := range componentResourceGVKs {
		restMapper.Add(gvk, meta.RESTScopeNamespace)
	}

	return fake.NewClientBuilder().
		WithScheme(scheme).
		WithRESTMapper(restMapper).
		WithObjects(objects...).
		WithInterceptorFuncs(funcs).
		Build()
}

func newResource(gvk schema.GroupVersionKind, name string, resourceLabels map[string]string) *unstructured.Unstructured {
	resource := &unstructured.Unstructured{}
	resource.SetGroupVersionKind(gvk)
	resource.SetNamespace(testNamespace)
	resource.SetName(name)
	resource.SetLabels(resourceLabels)
	return resource
}

func newDogu(name string) *unstructured.Unstructured {
	return newResource(doguGVK, name, map[string]string{"k8s.cloudogu.com/dogu.name": name})
}

// existingNames returns the names of the resources of the given kind that still exist.
func existingNames(t *testing.T, c client.Client, gvk schema.GroupVersionKind) []string {
	t.Helper()
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(gvk)
	require.NoError(t, c.List(t.Context(), list))
	var names []string
	for _, item := range list.Items {
		names = append(names, item.GetName())
	}
	return names
}

// waitFor waits for the wait group and reports whether it finished before the timeout.
func waitFor(wg *sync.WaitGroup, timeout time.Duration) bool {
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return true
	case <-time.After(timeout):
		return false
	}
}

// newTestManager creates a manager that checks for deleted resources quickly.
func newTestManager(k8sClient k8sClient) *DefaultCleanupManager {
	sut := NewManager(k8sClient, testNamespace)
	sut.deleteWaitTime = time.Millisecond
	return sut
}

func TestDefaultCleanupManager_deleteMatching(t *testing.T) {
	selector := labels.SelectorFromSet(labels.Set{"k8s.cloudogu.com/dogu.name": "cas"})

	t.Run("delete only the matching resources of the kind in the namespace", func(t *testing.T) {
		otherNamespace := newDogu("cas")
		otherNamespace.SetNamespace("other")
		casConfigMap := newResource(corev1.SchemeGroupVersion.WithKind("ConfigMap"), "cas", map[string]string{"k8s.cloudogu.com/dogu.name": "cas"})
		k8sClient := newFakeClient(t, interceptor.Funcs{}, newDogu("cas"), newDogu("ldap"), otherNamespace, casConfigMap)
		sut := newTestManager(k8sClient)
		var wg sync.WaitGroup

		err := sut.deleteMatching(t.Context(), &wg, doguGVK, selector)

		require.NoError(t, err)
		assert.True(t, waitFor(&wg, time.Second))
		assert.ElementsMatch(t, []string{"ldap", "cas"}, existingNames(t, k8sClient, doguGVK), "ldap and the cas dogu of the other namespace remain")
		assert.Equal(t, []string{"cas"}, existingNames(t, k8sClient, casConfigMap.GroupVersionKind()))
	})

	t.Run("delete in foreground", func(t *testing.T) {
		var deleteOptions client.DeleteOptions
		k8sClient := newFakeClient(t, interceptor.Funcs{
			Delete: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.DeleteOption) error {
				deleteOptions.ApplyOptions(opts)
				return c.Delete(ctx, obj, opts...)
			},
		}, newDogu("cas"))
		var wg sync.WaitGroup

		require.NoError(t, newTestManager(k8sClient).deleteMatching(t.Context(), &wg, doguGVK, selector))

		require.NotNil(t, deleteOptions.PropagationPolicy)
		assert.Equal(t, "Foreground", string(*deleteOptions.PropagationPolicy))
		assert.True(t, waitFor(&wg, time.Second))
	})

	t.Run("skip an already deleted resource and delete the following ones", func(t *testing.T) {
		everything := labels.Everything()
		k8sClient := newFakeClient(t, interceptor.Funcs{
			Delete: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.DeleteOption) error {
				if obj.GetName() == "cas" {
					// e.g. deleted in the meantime by the foreground deletion of its dogu
					return k8sErr.NewNotFound(schema.GroupResource{}, obj.GetName())
				}
				return c.Delete(ctx, obj, opts...)
			},
		}, newDogu("cas"), newDogu("ldap"))
		var wg sync.WaitGroup

		err := newTestManager(k8sClient).deleteMatching(t.Context(), &wg, doguGVK, everything)

		require.NoError(t, err)
		assert.True(t, waitFor(&wg, time.Second))
		assert.Equal(t, []string{"cas"}, existingNames(t, k8sClient, doguGVK))
	})

	t.Run("fail to list", func(t *testing.T) {
		k8sClient := newFakeClient(t, interceptor.Funcs{
			List: func(context.Context, client.WithWatch, client.ObjectList, ...client.ListOption) error {
				return assert.AnError
			},
		})
		var wg sync.WaitGroup

		err := newTestManager(k8sClient).deleteMatching(t.Context(), &wg, doguGVK, selector)

		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to list Dogu")
	})

	t.Run("fail to delete", func(t *testing.T) {
		k8sClient := newFakeClient(t, interceptor.Funcs{
			Delete: func(context.Context, client.WithWatch, client.Object, ...client.DeleteOption) error {
				return assert.AnError
			},
		}, newDogu("cas"))
		var wg sync.WaitGroup

		err := newTestManager(k8sClient).deleteMatching(t.Context(), &wg, doguGVK, selector)

		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to delete Dogu \"cas\"")
	})
}

func TestDefaultCleanupManager_waitForDeletion(t *testing.T) {
	key := client.ObjectKey{Namespace: testNamespace, Name: "cas"}

	t.Run("return when the resource is gone", func(t *testing.T) {
		var gets atomic.Int32
		k8sClient := newFakeClient(t, interceptor.Funcs{
			Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
				if gets.Add(1) < 3 {
					// the resource is still there, then the client cannot reach the api server
					if gets.Load() == 2 {
						return assert.AnError
					}
					return nil
				}
				return c.Get(ctx, key, obj, opts...)
			},
		})
		var wg sync.WaitGroup

		wg.Go(func() { newTestManager(k8sClient).waitForDeletion(t.Context(), doguGVK, key) })

		assert.True(t, waitFor(&wg, time.Second))
		assert.Equal(t, int32(3), gets.Load())
	})

	t.Run("keep waiting while the resource exists until the context ends", func(t *testing.T) {
		k8sClient := newFakeClient(t, interceptor.Funcs{}, newDogu("cas"))
		ctx, cancel := context.WithCancel(t.Context())
		var wg sync.WaitGroup

		wg.Go(func() { newTestManager(k8sClient).waitForDeletion(ctx, doguGVK, key) })

		assert.False(t, waitFor(&wg, 50*time.Millisecond), "must wait while the dogu exists")
		cancel()
		assert.True(t, waitFor(&wg, time.Second), "must stop waiting when the context ends")
	})
}
