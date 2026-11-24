package cleanup

import (
	"context"
	"sync"
	"testing"
	"time"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/k8s-backup-operator/internal/controller/restore/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestDefaultCleanupManager_Cleanup(t *testing.T) {
	restoreState := state.RestoreState{
		Kind:       state.KindUnits,
		V2Dogus:    []string{"cas"},
		V3Dogus:    []string{"postfix", "redmine"},
		Components: []string{"k8s-loki", "k8s-velero"},
	}
	// the excluded dogus and components must be kept
	spec := backupv1.RestoreSpec{ExcludedDogus: []string{"postfix"}, ExcludedComponents: []string{"k8s-loki"}}

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
					m.EXPECT().cleanupDogus(mock.Anything, &sync.WaitGroup{}, []string{"cas"}, []string{"redmine"}).Return(assert.AnError)
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
					m.EXPECT().cleanupDogus(mock.Anything, &sync.WaitGroup{}, []string{"cas"}, []string{"redmine"}).Return(nil)
					return m
				},
				additionalResourceManagerFn: func(t *testing.T) additionalResourceManager {
					m := newMockAdditionalResourceManager(t)
					m.EXPECT().cleanupAdditionalResources(mock.Anything, &sync.WaitGroup{}, []string{"k8s-velero"}).Return(assert.AnError)
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
					m.EXPECT().cleanupDogus(mock.Anything, &sync.WaitGroup{}, []string{"cas"}, []string{"redmine"}).Return(nil)
					return m
				},
				additionalResourceManagerFn: func(t *testing.T) additionalResourceManager {
					m := newMockAdditionalResourceManager(t)
					m.EXPECT().cleanupAdditionalResources(mock.Anything, &sync.WaitGroup{}, []string{"k8s-velero"}).Run(
						func(ctx context.Context, wg *sync.WaitGroup, _ []string) {
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
					m.EXPECT().cleanupDogus(mock.Anything, &sync.WaitGroup{}, []string{"cas"}, []string{"redmine"}).Run(
						func(ctx context.Context, wg *sync.WaitGroup, _ []string, _ []string) {
							wg.Add(1)
							go func() {
								wg.Done()
							}()
						}).Return(nil)
					return m
				},
				additionalResourceManagerFn: func(t *testing.T) additionalResourceManager {
					m := newMockAdditionalResourceManager(t)
					m.EXPECT().cleanupAdditionalResources(mock.Anything, mock.Anything, []string{"k8s-velero"}).Run(
						func(ctx context.Context, wg *sync.WaitGroup, _ []string) {
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
			tt.wantErr(t, c.Cleanup(t.Context(), restoreState, spec))
		})
	}
}
