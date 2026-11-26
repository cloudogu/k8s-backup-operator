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
	"k8s.io/apimachinery/pkg/labels"
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
	restoredDogus := matchesSelector("k8s.cloudogu.com/dogu.name in (cas,redmine)")
	restoredComponents := matchesSelector("k8s.cloudogu.com/backup-scope,k8s.cloudogu.com/component.name in (k8s-velero)")

	type fields struct {
		doguManagerFn              func(t *testing.T) doguManager
		componentResourceManagerFn func(t *testing.T) componentResourceManager
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
					m.EXPECT().cleanupDogus(mock.Anything, &sync.WaitGroup{}, restoredDogus, restoredDogus).Return(assert.AnError)
					return m
				},
				componentResourceManagerFn: func(t *testing.T) componentResourceManager {
					m := newMockComponentResourceManager(t)
					return m
				},
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, assert.AnError, i) &&
					assert.ErrorContains(t, err, "failed to cleanup dogus", i)
			},
		},
		{
			name: "fail to cleanup component resources",
			fields: fields{
				doguManagerFn: func(t *testing.T) doguManager {
					m := newMockDoguManager(t)
					m.EXPECT().cleanupDogus(mock.Anything, &sync.WaitGroup{}, restoredDogus, restoredDogus).Return(nil)
					return m
				},
				componentResourceManagerFn: func(t *testing.T) componentResourceManager {
					m := newMockComponentResourceManager(t)
					m.EXPECT().cleanupComponentResources(mock.Anything, &sync.WaitGroup{}, restoredComponents).Return(assert.AnError)
					return m
				},
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, assert.AnError, i) &&
					assert.ErrorContains(t, err, "failed to cleanup component resources", i)
			},
		},
		{
			name: "fail with timeout",
			fields: fields{
				doguManagerFn: func(t *testing.T) doguManager {
					m := newMockDoguManager(t)
					m.EXPECT().cleanupDogus(mock.Anything, &sync.WaitGroup{}, restoredDogus, restoredDogus).Return(nil)
					return m
				},
				componentResourceManagerFn: func(t *testing.T) componentResourceManager {
					m := newMockComponentResourceManager(t)
					m.EXPECT().cleanupComponentResources(mock.Anything, &sync.WaitGroup{}, restoredComponents).Run(
						func(ctx context.Context, wg *sync.WaitGroup, _ labels.Selector) {
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
					m.EXPECT().cleanupDogus(mock.Anything, &sync.WaitGroup{}, restoredDogus, restoredDogus).Run(
						func(ctx context.Context, wg *sync.WaitGroup, _ labels.Selector, _ labels.Selector) {
							wg.Add(1)
							go func() {
								wg.Done()
							}()
						}).Return(nil)
					return m
				},
				componentResourceManagerFn: func(t *testing.T) componentResourceManager {
					m := newMockComponentResourceManager(t)
					m.EXPECT().cleanupComponentResources(mock.Anything, mock.Anything, restoredComponents).Run(
						func(ctx context.Context, wg *sync.WaitGroup, _ labels.Selector) {
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
				doguManager:              tt.fields.doguManagerFn(t),
				componentResourceManager: tt.fields.componentResourceManagerFn(t),
			}
			tt.wantErr(t, c.Cleanup(t.Context(), restoreState, spec))
		})
	}
}

func TestDefaultCleanupManager_Cleanup_selectsResourcesByKind(t *testing.T) {
	units := state.RestoreState{
		Kind:       state.KindUnits,
		V2Dogus:    []string{"cas"},
		V3Dogus:    []string{"redmine"},
		Components: []string{"k8s-velero"},
	}
	tests := []struct {
		name                   string
		restoreState           state.RestoreState
		spec                   backupv1.RestoreSpec
		wantDogus              string
		wantDoguPVCs           string
		wantComponentResources string
		skipDogus              bool
		skipComponentResources bool
		wantErrMsg             string
	}{
		{
			name:                   "legacy deletes all dogus and all resources with a backup scope",
			restoreState:           state.RestoreState{Kind: state.KindLegacy, Units: []state.UnitBackup{{}}},
			wantDogus:              "",
			wantDoguPVCs:           "k8s.cloudogu.com/dogu.name",
			wantComponentResources: "k8s.cloudogu.com/backup-scope",
		},
		{
			name:                   "units delete only the restored dogus and components",
			restoreState:           units,
			wantDogus:              "k8s.cloudogu.com/dogu.name in (cas,redmine)",
			wantDoguPVCs:           "k8s.cloudogu.com/dogu.name in (cas,redmine)",
			wantComponentResources: "k8s.cloudogu.com/backup-scope,k8s.cloudogu.com/component.name in (k8s-velero)",
		},
		{
			name:                   "units without restored dogus skip the dogu cleanup",
			restoreState:           units,
			spec:                   backupv1.RestoreSpec{ExcludedDogus: []string{"redmine"}, Options: &backupv1.UnitOptions{IgnoreV2Dogus: true}},
			skipDogus:              true,
			wantComponentResources: "k8s.cloudogu.com/backup-scope,k8s.cloudogu.com/component.name in (k8s-velero)",
		},
		{
			name:                   "units without restored components skip the component resource cleanup",
			restoreState:           units,
			spec:                   backupv1.RestoreSpec{ExcludedComponents: []string{"k8s-velero"}},
			wantDogus:              "k8s.cloudogu.com/dogu.name in (cas,redmine)",
			wantDoguPVCs:           "k8s.cloudogu.com/dogu.name in (cas,redmine)",
			skipComponentResources: true,
		},
		{
			name:                   "unknown backup kind fails without deleting anything",
			restoreState:           state.RestoreState{Kind: state.KindNotFound},
			skipDogus:              true,
			skipComponentResources: true,
			wantErrMsg:             "cannot clean up for a backup of kind \"NotFound\"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// mocks without expectations fail on any call
			doguMock := newMockDoguManager(t)
			if !tt.skipDogus {
				doguMock.EXPECT().cleanupDogus(mock.Anything, mock.Anything, matchesSelector(tt.wantDogus), matchesSelector(tt.wantDoguPVCs)).Return(nil)
			}
			componentResourceMock := newMockComponentResourceManager(t)
			if !tt.skipComponentResources {
				componentResourceMock.EXPECT().cleanupComponentResources(mock.Anything, mock.Anything, matchesSelector(tt.wantComponentResources)).Return(nil)
			}
			c := &DefaultCleanupManager{doguManager: doguMock, componentResourceManager: componentResourceMock}

			err := c.Cleanup(t.Context(), tt.restoreState, tt.spec)

			if tt.wantErrMsg != "" {
				assert.ErrorContains(t, err, tt.wantErrMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func matchesSelector(want string) any {
	return mock.MatchedBy(func(selector labels.Selector) bool { return selector.String() == want })
}
