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
	tests := []struct {
		name          string
		doguClientFn  func(t *testing.T) doguClient
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
			wantErr:       assert.NoError,
			shouldTimeout: true,
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

			var ctx, cancel = context.WithTimeout(t.Context(), 100*time.Millisecond)
			defer cancel()
			var wg sync.WaitGroup

			tt.wantErr(t, c.cleanupDogus(t.Context(), &wg))

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
