package cleanup

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestDefaultCleanupManager_Cleanup(t *testing.T) {
	type fields struct {
		doguManagerFn               func(t *testing.T) doguManager
		additionalResourceManagerFn func(t *testing.T) additionalResourceManager
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "fail to cleanup dogus",
			fields: fields{
				doguManagerFn: func(t *testing.T) doguManager {
					m := newMockDoguManager(t)
					m.EXPECT().cleanupDogus(mock.Anything, &sync.WaitGroup{}).Return(assert.AnError)
					return m
				},
				additionalResourceManagerFn: func(t *testing.T) additionalResourceManager {
					m := newMockAdditionalResourceManager(t)
					return m
				},
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, assert.AnError, i) &&
					assert.ErrorContains(t, err, "failed to cleanup dogus", i)
			},
		},
		{
			name: "fail to cleanup additional resources",
			fields: fields{
				doguManagerFn: func(t *testing.T) doguManager {
					m := newMockDoguManager(t)
					m.EXPECT().cleanupDogus(mock.Anything, &sync.WaitGroup{}).Return(nil)
					return m
				},
				additionalResourceManagerFn: func(t *testing.T) additionalResourceManager {
					m := newMockAdditionalResourceManager(t)
					m.EXPECT().cleanupAdditionalResources(mock.Anything, &sync.WaitGroup{}).Return(assert.AnError)
					return m
				},
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, assert.AnError, i) &&
					assert.ErrorContains(t, err, "failed to cleanup additional resources", i)
			},
		},
		{
			name: "fail with timeout",
			fields: fields{
				doguManagerFn: func(t *testing.T) doguManager {
					m := newMockDoguManager(t)
					m.EXPECT().cleanupDogus(mock.Anything, &sync.WaitGroup{}).Return(nil)
					return m
				},
				additionalResourceManagerFn: func(t *testing.T) additionalResourceManager {
					m := newMockAdditionalResourceManager(t)
					m.EXPECT().cleanupAdditionalResources(mock.Anything, &sync.WaitGroup{}).Run(
						func(ctx context.Context, wg *sync.WaitGroup) {
							wg.Add(1)
						}).Return(nil)
					return m
				},
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorContains(t, err, "cleanup timed out", i)
			},
		},
		{
			name: "succeed",
			fields: fields{
				doguManagerFn: func(t *testing.T) doguManager {
					m := newMockDoguManager(t)
					m.EXPECT().cleanupDogus(mock.Anything, &sync.WaitGroup{}).Run(
						func(ctx context.Context, wg *sync.WaitGroup) {
							wg.Add(1)
							go func() {
								wg.Done()
							}()
						}).Return(nil)
					return m
				},
				additionalResourceManagerFn: func(t *testing.T) additionalResourceManager {
					m := newMockAdditionalResourceManager(t)
					m.EXPECT().cleanupAdditionalResources(mock.Anything, mock.Anything).Run(
						func(ctx context.Context, wg *sync.WaitGroup) {
							wg.Add(1)
							go func() {
								wg.Done()
							}()
						}).Return(nil)
					return m
				},
			},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			previousCleanupTimeout := cleanupTimeout
			defer func() { cleanupTimeout = previousCleanupTimeout }()
			cleanupTimeout = 10 * time.Millisecond

			c := &DefaultCleanupManager{
				doguManager:               tt.fields.doguManagerFn(t),
				additionalResourceManager: tt.fields.additionalResourceManagerFn(t),
			}
			tt.wantErr(t, c.Cleanup(t.Context()))
		})
	}
}
