package cleanup

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"
)

func Test_defaultAdditionalResourceManager_cleanupAdditionalResources(t *testing.T) {
	tests := []struct {
		name              string
		configMapClientFn func(t *testing.T) unstructuredClient
		pvcClientFn       func(t *testing.T) unstructuredClient
		wantErr           assert.ErrorAssertionFunc
		shouldTimeout     bool
	}{
		{
			name: "fail to list configmaps",
			configMapClientFn: func(t *testing.T) unstructuredClient {
				m := newMockUnstructuredClient(t)
				m.EXPECT().List(t.Context(), metav1.ListOptions{LabelSelector: "k8s.cloudogu.com/backup-scope"}).Return(nil, assert.AnError)
				return m
			},
			pvcClientFn: func(t *testing.T) unstructuredClient {
				return newMockUnstructuredClient(t)
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, assert.AnError, i) &&
					assert.ErrorContains(t, err, "failed to list configmaps", i)
			},
		},
		{
			name: "fail to delete configmap",
			configMapClientFn: func(t *testing.T) unstructuredClient {
				m := newMockUnstructuredClient(t)
				m.EXPECT().List(t.Context(), metav1.ListOptions{LabelSelector: "k8s.cloudogu.com/backup-scope"}).Return(&unstructured.UnstructuredList{
					Items: []unstructured.Unstructured{
						{Object: map[string]interface{}{
							"apiVersion": "v1",
							"kind":       "ConfigMap",
							"metadata":   map[string]interface{}{"name": "test"},
						}},
					},
				}, nil)
				propagationPolicyForeground := metav1.DeletePropagationForeground
				m.EXPECT().Delete(t.Context(), "test", metav1.DeleteOptions{PropagationPolicy: &propagationPolicyForeground}).Return(assert.AnError)
				return m
			},
			pvcClientFn: func(t *testing.T) unstructuredClient {
				return newMockUnstructuredClient(t)
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, assert.AnError, i) &&
					assert.ErrorContains(t, err, "failed to delete ConfigMap \"test\"", i)
			},
		},
		{
			name: "timeout with fail to get configmap",
			configMapClientFn: func(t *testing.T) unstructuredClient {
				m := newMockUnstructuredClient(t)
				m.EXPECT().List(t.Context(), metav1.ListOptions{LabelSelector: "k8s.cloudogu.com/backup-scope"}).Return(&unstructured.UnstructuredList{
					Items: []unstructured.Unstructured{
						{Object: map[string]interface{}{
							"apiVersion": "v1",
							"kind":       "ConfigMap",
							"metadata":   map[string]interface{}{"name": "test"},
						}},
					},
				}, nil)
				propagationPolicyForeground := metav1.DeletePropagationForeground
				m.EXPECT().Delete(t.Context(), "test", metav1.DeleteOptions{PropagationPolicy: &propagationPolicyForeground}).Return(nil)
				m.EXPECT().Get(t.Context(), "test", metav1.GetOptions{}).Return(nil, assert.AnError)
				return m
			},
			pvcClientFn: func(t *testing.T) unstructuredClient {
				m := newMockUnstructuredClient(t)
				m.EXPECT().List(t.Context(), metav1.ListOptions{LabelSelector: "k8s.cloudogu.com/backup-scope"}).Return(&unstructured.UnstructuredList{
					Items: []unstructured.Unstructured{
						{Object: map[string]interface{}{
							"apiVersion": "v1",
							"kind":       "PersistentVolumeClaim",
							"metadata":   map[string]interface{}{"name": "test"},
						}},
					},
				}, nil)
				propagationPolicyForeground := metav1.DeletePropagationForeground
				m.EXPECT().Delete(t.Context(), "test", metav1.DeleteOptions{PropagationPolicy: &propagationPolicyForeground}).Return(nil)
				m.EXPECT().Get(t.Context(), "test", metav1.GetOptions{}).Return(nil, assert.AnError)
				return m
			},
			wantErr:       assert.NoError,
			shouldTimeout: true,
		},
		{
			name: "timeout with success to get configmap",
			configMapClientFn: func(t *testing.T) unstructuredClient {
				m := newMockUnstructuredClient(t)
				m.EXPECT().List(t.Context(), metav1.ListOptions{LabelSelector: "k8s.cloudogu.com/backup-scope"}).Return(&unstructured.UnstructuredList{
					Items: []unstructured.Unstructured{
						{Object: map[string]interface{}{
							"apiVersion": "v1",
							"kind":       "ConfigMap",
							"metadata":   map[string]interface{}{"name": "test"},
						}},
					},
				}, nil)
				propagationPolicyForeground := metav1.DeletePropagationForeground
				m.EXPECT().Delete(t.Context(), "test", metav1.DeleteOptions{PropagationPolicy: &propagationPolicyForeground}).Return(nil)
				m.EXPECT().Get(t.Context(), "test", metav1.GetOptions{}).Return(&unstructured.Unstructured{}, nil)
				return m
			},
			pvcClientFn: func(t *testing.T) unstructuredClient {
				m := newMockUnstructuredClient(t)
				m.EXPECT().List(t.Context(), metav1.ListOptions{LabelSelector: "k8s.cloudogu.com/backup-scope"}).Return(&unstructured.UnstructuredList{
					Items: []unstructured.Unstructured{
						{Object: map[string]interface{}{
							"apiVersion": "v1",
							"kind":       "PersistentVolumeClaim",
							"metadata":   map[string]interface{}{"name": "test"},
						}},
					},
				}, nil)
				propagationPolicyForeground := metav1.DeletePropagationForeground
				m.EXPECT().Delete(t.Context(), "test", metav1.DeleteOptions{PropagationPolicy: &propagationPolicyForeground}).Return(nil)
				m.EXPECT().Get(t.Context(), "test", metav1.GetOptions{}).Return(&unstructured.Unstructured{}, nil)
				return m
			},
			wantErr:       assert.NoError,
			shouldTimeout: true,
		},
		{
			name: "succeed without timeout on not found",
			configMapClientFn: func(t *testing.T) unstructuredClient {
				m := newMockUnstructuredClient(t)
				m.EXPECT().List(t.Context(), metav1.ListOptions{LabelSelector: "k8s.cloudogu.com/backup-scope"}).Return(&unstructured.UnstructuredList{
					Items: []unstructured.Unstructured{
						{Object: map[string]interface{}{
							"apiVersion": "v1",
							"kind":       "ConfigMap",
							"metadata":   map[string]interface{}{"name": "test"},
						}},
					},
				}, nil)
				propagationPolicyForeground := metav1.DeletePropagationForeground
				m.EXPECT().Delete(t.Context(), "test", metav1.DeleteOptions{PropagationPolicy: &propagationPolicyForeground}).Return(nil)
				m.EXPECT().Get(t.Context(), "test", metav1.GetOptions{}).
					Return(nil, &errors.StatusError{ErrStatus: metav1.Status{Reason: metav1.StatusReasonNotFound}})
				return m
			},
			pvcClientFn: func(t *testing.T) unstructuredClient {
				m := newMockUnstructuredClient(t)
				m.EXPECT().List(t.Context(), metav1.ListOptions{LabelSelector: "k8s.cloudogu.com/backup-scope"}).Return(&unstructured.UnstructuredList{
					Items: []unstructured.Unstructured{
						{Object: map[string]interface{}{
							"apiVersion": "v1",
							"kind":       "PersistentVolumeClaim",
							"metadata":   map[string]interface{}{"name": "test"},
						}},
					},
				}, nil)
				propagationPolicyForeground := metav1.DeletePropagationForeground
				m.EXPECT().Delete(t.Context(), "test", metav1.DeleteOptions{PropagationPolicy: &propagationPolicyForeground}).Return(nil)
				m.EXPECT().Get(t.Context(), "test", metav1.GetOptions{}).
					Return(nil, &errors.StatusError{ErrStatus: metav1.Status{Reason: metav1.StatusReasonNotFound}})
				return m
			},
			wantErr:       assert.NoError,
			shouldTimeout: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			previousWaitTime := doguDeleteWaitTime
			additionalResourceDeleteWaitTime = 10 * time.Millisecond
			defer func() { additionalResourceDeleteWaitTime = previousWaitTime }()

			c := &defaultAdditionalResourceManager{
				clients: map[schema.GroupVersionResource]unstructuredClient{
					corev1.SchemeGroupVersion.WithResource("configmaps"):             tt.configMapClientFn(t),
					corev1.SchemeGroupVersion.WithResource("persistentvolumeclaims"): tt.pvcClientFn(t),
				},
			}

			var ctx, cancel = context.WithTimeout(t.Context(), 100*time.Millisecond)
			defer cancel()
			var wg sync.WaitGroup

			tt.wantErr(t, c.cleanupAdditionalResources(t.Context(), &wg))

			done := make(chan struct{})
			go func() {
				wg.Wait()
				close(done)
			}()
			select {
			case <-done:
			case <-ctx.Done():
				if !tt.shouldTimeout {
					assert.Fail(t, "cleanup timed out")
				}
			}
		})
	}
}

func Test_newAdditionalResourceManager(t *testing.T) {
	scheme := runtime.NewScheme()
	err := corev1.AddToScheme(scheme)
	assert.NoError(t, err)
	dynamicClient := fake.NewSimpleDynamicClient(scheme)

	actual := newAdditionalResourceManager(dynamicClient, "test-namespace")

	assert.NotEmpty(t, actual)
}
