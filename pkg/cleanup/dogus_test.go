package cleanup

import (
	"context"
	"sync"
	"testing"
	"time"

	doguv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_defaultDoguManager_cleanupDogus(t *testing.T) {
	cancelCtx, cancel := context.WithCancel(t.Context())
	tests := []struct {
		name          string
		doguClientFn  func(t *testing.T) doguClient
		ctxFn         func(t *testing.T) context.Context
		wantErr       assert.ErrorAssertionFunc
		shouldTimeout bool
	}{
		{
			name: "fail to list dogus",
			doguClientFn: func(t *testing.T) doguClient {
				m := newMockDoguClient(t)
				m.EXPECT().List(t.Context(), metav1.ListOptions{}).Return(nil, assert.AnError)
				return m
			},
			ctxFn: func(t *testing.T) context.Context {
				return t.Context()
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, assert.AnError, i) &&
					assert.ErrorContains(t, err, "failed to list dogus", i)
			},
		},
		{
			name: "fail to delete dogu",
			doguClientFn: func(t *testing.T) doguClient {
				m := newMockDoguClient(t)
				m.EXPECT().List(t.Context(), metav1.ListOptions{}).Return(&doguv2.DoguList{
					Items: []doguv2.Dogu{
						{ObjectMeta: metav1.ObjectMeta{Name: "test"}},
					},
				}, nil)
				propagationPolicyForeground := metav1.DeletePropagationForeground
				m.EXPECT().Delete(t.Context(), "test", metav1.DeleteOptions{PropagationPolicy: &propagationPolicyForeground}).Return(assert.AnError)
				return m
			},
			ctxFn: func(t *testing.T) context.Context {
				return t.Context()
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, assert.AnError, i) &&
					assert.ErrorContains(t, err, "failed to delete dogu \"test\"", i)
			},
		},
		{
			name: "timeout with fail to get dogu",
			doguClientFn: func(t *testing.T) doguClient {
				m := newMockDoguClient(t)
				m.EXPECT().List(t.Context(), metav1.ListOptions{}).Return(&doguv2.DoguList{
					Items: []doguv2.Dogu{
						{ObjectMeta: metav1.ObjectMeta{Name: "test"}},
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
				m.EXPECT().List(t.Context(), metav1.ListOptions{}).Return(&doguv2.DoguList{
					Items: []doguv2.Dogu{
						{ObjectMeta: metav1.ObjectMeta{Name: "test"}},
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
				m.EXPECT().List(cancelCtx, metav1.ListOptions{}).Return(&doguv2.DoguList{
					Items: []doguv2.Dogu{
						{ObjectMeta: metav1.ObjectMeta{Name: "test"}},
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
				m.EXPECT().List(t.Context(), metav1.ListOptions{}).Return(&doguv2.DoguList{
					Items: []doguv2.Dogu{
						{ObjectMeta: metav1.ObjectMeta{Name: "test"}},
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			previousWaitTime := doguDeleteWaitTime
			doguDeleteWaitTime = 10 * time.Millisecond
			defer func() { doguDeleteWaitTime = previousWaitTime }()

			c := &defaultDoguManager{
				doguClient: tt.doguClientFn(t),
			}

			timer := time.NewTimer(100 * time.Millisecond)
			defer cancel()
			var wg sync.WaitGroup

			tt.wantErr(t, c.cleanupDogus(tt.ctxFn(t), &wg))

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
