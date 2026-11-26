package cleanup

import (
	"context"
	"sync"
	"testing"
	"time"

	doguv2 "github.com/cloudogu/k8s-dogu-lib/v3/api/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
)

// doguListOptions select the resources of the v2 dogu cas and the v3 dogu test.
var doguListOptions = metav1.ListOptions{LabelSelector: "k8s.cloudogu.com/dogu.name in (cas,test)"}

func Test_defaultDoguManager_cleanupDogus(t *testing.T) {
	cancelCtx, cancel := context.WithCancel(t.Context())
	notFound := &errors.StatusError{ErrStatus: metav1.Status{Reason: metav1.StatusReasonNotFound}}
	pvc := func(name string) unstructured.Unstructured {
		return unstructured.Unstructured{Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "PersistentVolumeClaim",
			"metadata":   map[string]any{"name": name},
		}}
	}
	tests := []struct {
		name         string
		doguClientFn func(t *testing.T) doguClient
		// pvcClientFn is optional; by default there are no dogu pvcs.
		pvcClientFn   func(t *testing.T) unstructuredClient
		ctxFn         func(t *testing.T) context.Context
		wantErr       assert.ErrorAssertionFunc
		shouldTimeout bool
	}{
		{
			name: "fail to list dogus",
			doguClientFn: func(t *testing.T) doguClient {
				m := newMockDoguClient(t)
				m.EXPECT().List(t.Context(), doguListOptions).Return(nil, assert.AnError)
				return m
			},
			ctxFn: func(t *testing.T) context.Context {
				return t.Context()
			},
			wantErr: func(t assert.TestingT, err error, i ...any) bool {
				return assert.ErrorIs(t, err, assert.AnError, i) &&
					assert.ErrorContains(t, err, "failed to list dogus", i)
			},
		},
		{
			name: "fail to delete dogu",
			doguClientFn: func(t *testing.T) doguClient {
				m := newMockDoguClient(t)
				m.EXPECT().List(t.Context(), doguListOptions).Return(&doguv2.DoguList{
					Items: []doguv2.Dogu{
						{Name: "test"},
					},
				}, nil)
				propagationPolicyForeground := metav1.DeletePropagationForeground
				m.EXPECT().Delete(t.Context(), "test", metav1.DeleteOptions{PropagationPolicy: &propagationPolicyForeground}).Return(assert.AnError)
				return m
			},
			ctxFn: func(t *testing.T) context.Context {
				return t.Context()
			},
			wantErr: func(t assert.TestingT, err error, i ...any) bool {
				return assert.ErrorIs(t, err, assert.AnError, i) &&
					assert.ErrorContains(t, err, "failed to delete dogu \"test\"", i)
			},
		},
		{
			name: "timeout with fail to get dogu",
			doguClientFn: func(t *testing.T) doguClient {
				m := newMockDoguClient(t)
				m.EXPECT().List(t.Context(), doguListOptions).Return(&doguv2.DoguList{
					Items: []doguv2.Dogu{
						{Name: "test"},
					},
				}, nil)
				propagationPolicyForeground := metav1.DeletePropagationForeground
				m.EXPECT().Delete(t.Context(), "test", metav1.DeleteOptions{PropagationPolicy: &propagationPolicyForeground}).Return(nil)
				m.EXPECT().Get(t.Context(), "test", metav1.GetOptions{}).Return(nil, assert.AnError)
				return m
			},
			ctxFn: func(t *testing.T) context.Context {
				return t.Context()
			},
			wantErr:       assert.NoError,
			shouldTimeout: true,
		},
		{
			name: "timeout with success to get dogu",
			doguClientFn: func(t *testing.T) doguClient {
				m := newMockDoguClient(t)
				m.EXPECT().List(t.Context(), doguListOptions).Return(&doguv2.DoguList{
					Items: []doguv2.Dogu{
						{Name: "test"},
					},
				}, nil)
				propagationPolicyForeground := metav1.DeletePropagationForeground
				m.EXPECT().Delete(t.Context(), "test", metav1.DeleteOptions{PropagationPolicy: &propagationPolicyForeground}).Return(nil)
				m.EXPECT().Get(t.Context(), "test", metav1.GetOptions{}).Return(&doguv2.Dogu{}, nil)
				return m
			},
			ctxFn: func(t *testing.T) context.Context {
				return t.Context()
			},
			wantErr:       assert.NoError,
			shouldTimeout: true,
		},
		{
			name: "abort on cancelled context",
			doguClientFn: func(t *testing.T) doguClient {
				m := newMockDoguClient(t)
				m.EXPECT().List(cancelCtx, doguListOptions).Return(&doguv2.DoguList{
					Items: []doguv2.Dogu{
						{Name: "test"},
					},
				}, nil)
				propagationPolicyForeground := metav1.DeletePropagationForeground
				m.EXPECT().Delete(cancelCtx, "test", metav1.DeleteOptions{PropagationPolicy: &propagationPolicyForeground}).Return(nil)
				m.EXPECT().Get(cancelCtx, "test", metav1.GetOptions{}).RunAndReturn(
					func(ctx context.Context, _ string, _ metav1.GetOptions) (*doguv2.Dogu, error) {
						cancel()
						return nil, ctx.Err()
					})
				return m
			},
			ctxFn: func(t *testing.T) context.Context {
				return cancelCtx
			},
			wantErr:       assert.NoError,
			shouldTimeout: false,
		},
		{
			name: "succeed without timeout on not found",
			doguClientFn: func(t *testing.T) doguClient {
				m := newMockDoguClient(t)
				m.EXPECT().List(t.Context(), doguListOptions).Return(&doguv2.DoguList{
					Items: []doguv2.Dogu{
						{Name: "test"},
					},
				}, nil)
				propagationPolicyForeground := metav1.DeletePropagationForeground
				m.EXPECT().Delete(t.Context(), "test", metav1.DeleteOptions{PropagationPolicy: &propagationPolicyForeground}).Return(nil)
				m.EXPECT().Get(t.Context(), "test", metav1.GetOptions{}).
					Return(nil, &errors.StatusError{ErrStatus: metav1.Status{Reason: metav1.StatusReasonNotFound}})
				return m
			},
			ctxFn: func(t *testing.T) context.Context {
				return t.Context()
			},
			wantErr:       assert.NoError,
			shouldTimeout: false,
		},
		{
			name: "skip already deleted dogu and delete the following ones",
			doguClientFn: func(t *testing.T) doguClient {
				m := newMockDoguClient(t)
				m.EXPECT().List(t.Context(), doguListOptions).Return(&doguv2.DoguList{
					Items: []doguv2.Dogu{{Name: "cas"}, {Name: "test"}},
				}, nil)
				propagationPolicyForeground := metav1.DeletePropagationForeground
				m.EXPECT().Delete(t.Context(), "cas", metav1.DeleteOptions{PropagationPolicy: &propagationPolicyForeground}).Return(notFound)
				m.EXPECT().Delete(t.Context(), "test", metav1.DeleteOptions{PropagationPolicy: &propagationPolicyForeground}).Return(nil)
				m.EXPECT().Get(t.Context(), "test", metav1.GetOptions{}).Return(nil, notFound)
				return m
			},
			ctxFn: func(t *testing.T) context.Context {
				return t.Context()
			},
			wantErr:       assert.NoError,
			shouldTimeout: false,
		},
		{
			name: "fail to list dogu pvcs",
			doguClientFn: func(t *testing.T) doguClient {
				m := newMockDoguClient(t)
				m.EXPECT().List(t.Context(), doguListOptions).Return(&doguv2.DoguList{}, nil)
				return m
			},
			pvcClientFn: func(t *testing.T) unstructuredClient {
				m := newMockUnstructuredClient(t)
				m.EXPECT().List(t.Context(), doguListOptions).Return(nil, assert.AnError)
				return m
			},
			ctxFn: func(t *testing.T) context.Context {
				return t.Context()
			},
			wantErr: func(t assert.TestingT, err error, i ...any) bool {
				return assert.ErrorIs(t, err, assert.AnError, i) &&
					assert.ErrorContains(t, err, "failed to list dogu pvcs", i)
			},
		},
		{
			name: "fail to delete dogu pvc",
			doguClientFn: func(t *testing.T) doguClient {
				m := newMockDoguClient(t)
				m.EXPECT().List(t.Context(), doguListOptions).Return(&doguv2.DoguList{}, nil)
				return m
			},
			pvcClientFn: func(t *testing.T) unstructuredClient {
				m := newMockUnstructuredClient(t)
				m.EXPECT().List(t.Context(), doguListOptions).Return(&unstructured.UnstructuredList{Items: []unstructured.Unstructured{pvc("test-data")}}, nil)
				propagationPolicyForeground := metav1.DeletePropagationForeground
				m.EXPECT().Delete(t.Context(), "test-data", metav1.DeleteOptions{PropagationPolicy: &propagationPolicyForeground}).Return(assert.AnError)
				return m
			},
			ctxFn: func(t *testing.T) context.Context {
				return t.Context()
			},
			wantErr: func(t assert.TestingT, err error, i ...any) bool {
				return assert.ErrorIs(t, err, assert.AnError, i) &&
					assert.ErrorContains(t, err, "failed to delete dogu pvc \"test-data\"", i)
			},
		},
		{
			name: "succeed deleting dogu pvcs",
			doguClientFn: func(t *testing.T) doguClient {
				m := newMockDoguClient(t)
				m.EXPECT().List(t.Context(), doguListOptions).Return(&doguv2.DoguList{}, nil)
				return m
			},
			pvcClientFn: func(t *testing.T) unstructuredClient {
				m := newMockUnstructuredClient(t)
				m.EXPECT().List(t.Context(), doguListOptions).Return(&unstructured.UnstructuredList{Items: []unstructured.Unstructured{pvc("cas-data"), pvc("test-data")}}, nil)
				propagationPolicyForeground := metav1.DeletePropagationForeground
				// a pvc deleted in the meantime, e.g. by the foreground deletion of its dogu, is no error
				// and must neither be awaited nor stop the deletion of the following pvcs
				m.EXPECT().Delete(t.Context(), "cas-data", metav1.DeleteOptions{PropagationPolicy: &propagationPolicyForeground}).Return(notFound)
				m.EXPECT().Delete(t.Context(), "test-data", metav1.DeleteOptions{PropagationPolicy: &propagationPolicyForeground}).Return(nil)
				m.EXPECT().Get(t.Context(), "test-data", metav1.GetOptions{}).Return(nil, notFound)
				return m
			},
			ctxFn: func(t *testing.T) context.Context {
				return t.Context()
			},
			wantErr:       assert.NoError,
			shouldTimeout: false,
		},
		{
			name: "timeout waiting for dogu pvc deletion",
			doguClientFn: func(t *testing.T) doguClient {
				m := newMockDoguClient(t)
				m.EXPECT().List(t.Context(), doguListOptions).Return(&doguv2.DoguList{}, nil)
				return m
			},
			pvcClientFn: func(t *testing.T) unstructuredClient {
				m := newMockUnstructuredClient(t)
				m.EXPECT().List(t.Context(), doguListOptions).Return(&unstructured.UnstructuredList{Items: []unstructured.Unstructured{pvc("test-data")}}, nil)
				propagationPolicyForeground := metav1.DeletePropagationForeground
				m.EXPECT().Delete(t.Context(), "test-data", metav1.DeleteOptions{PropagationPolicy: &propagationPolicyForeground}).Return(nil)
				m.EXPECT().Get(t.Context(), "test-data", metav1.GetOptions{}).Return(&unstructured.Unstructured{}, nil)
				return m
			},
			ctxFn: func(t *testing.T) context.Context {
				return t.Context()
			},
			wantErr:       assert.NoError,
			shouldTimeout: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doguSelector := mustParseSelector(t, doguListOptions.LabelSelector)
			previousWaitTime := doguDeleteWaitTime
			doguDeleteWaitTime = 10 * time.Millisecond
			defer func() { doguDeleteWaitTime = previousWaitTime }()
			previousResourceWaitTime := componentResourceDeleteWaitTime
			componentResourceDeleteWaitTime = 10 * time.Millisecond
			defer func() { componentResourceDeleteWaitTime = previousResourceWaitTime }()

			pvcClient := newMockUnstructuredClient(t)
			if tt.pvcClientFn != nil {
				pvcClient = tt.pvcClientFn(t).(*mockUnstructuredClient)
			} else {
				// not reached if the dogus cannot be listed or deleted
				pvcClient.EXPECT().List(mock.Anything, doguListOptions).Return(&unstructured.UnstructuredList{}, nil).Maybe()
			}

			c := &defaultDoguManager{
				doguClient: tt.doguClientFn(t),
				pvcClient:  pvcClient,
			}

			timer := time.NewTimer(100 * time.Millisecond)
			defer cancel()
			var wg sync.WaitGroup

			tt.wantErr(t, c.cleanupDogus(tt.ctxFn(t), &wg, doguSelector, doguSelector))

			done := make(chan struct{})
			go func() {
				wg.Wait()
				close(done)
			}()
			select {
			case <-done:
				if tt.shouldTimeout {
					assert.Fail(t, "cleanup should timeout")
				}
			case <-timer.C:
				if !tt.shouldTimeout {
					assert.Fail(t, "cleanup timed out")
				}
			}
		})
	}
}

func Test_defaultDoguManager_cleanupDogus_withDifferentSelectors(t *testing.T) {
	// a legacy cleanup lists all dogus but only the pvcs with a dogu label
	doguClient := newMockDoguClient(t)
	doguClient.EXPECT().List(t.Context(), metav1.ListOptions{}).Return(&doguv2.DoguList{}, nil)
	pvcClient := newMockUnstructuredClient(t)
	pvcClient.EXPECT().List(t.Context(), metav1.ListOptions{LabelSelector: "k8s.cloudogu.com/dogu.name"}).Return(&unstructured.UnstructuredList{}, nil)
	c := &defaultDoguManager{doguClient: doguClient, pvcClient: pvcClient}
	var wg sync.WaitGroup

	err := c.cleanupDogus(t.Context(), &wg, labels.Everything(), mustParseSelector(t, "k8s.cloudogu.com/dogu.name"))

	assert.NoError(t, err)
}
