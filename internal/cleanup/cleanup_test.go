package cleanup

import (
	"context"
	"sync"
	"testing"
	"time"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/k8s-backup-operator/internal/controller/restore/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

var configMapGVK = corev1.SchemeGroupVersion.WithKind("ConfigMap")

func componentResource(name, component string) *unstructured.Unstructured {
	return newResource(configMapGVK, name, map[string]string{
		backupScopeLabelKey:               "component",
		"k8s.cloudogu.com/component.name": component,
	})
}

func doguPVC(name, dogu string) *unstructured.Unstructured {
	return newResource(pvcGVK, name, map[string]string{"k8s.cloudogu.com/dogu.name": dogu})
}

func shortCleanupTimeout(t *testing.T) {
	previous := cleanupTimeout
	cleanupTimeout = 50 * time.Millisecond
	t.Cleanup(func() { cleanupTimeout = previous })
}

func TestNewManager(t *testing.T) {
	k8sClient := newMockK8sClient(t)

	actual := NewManager(k8sClient, testNamespace)

	assert.Same(t, k8sClient, actual.client)
	assert.Equal(t, testNamespace, actual.namespace)
	assert.Equal(t, defaultWaitTime, actual.deleteWaitTime)
}

func TestDefaultCleanupManager_Cleanup(t *testing.T) {
	existingResources := func() []client.Object {
		return []client.Object{
			newDogu("cas"), newDogu("postfix"), newDogu("redmine"),
			// a dogu without the dogu label is only selected by a legacy cleanup
			newResource(doguGVK, "unlabeled", nil),
			doguPVC("cas-data", "cas"), doguPVC("postfix-data", "postfix"),
			componentResource("velero-config", "k8s-velero"), componentResource("loki-config", "k8s-loki"),
			// resources without a backup scope are never deleted
			newResource(configMapGVK, "unscoped", map[string]string{"k8s.cloudogu.com/component.name": "k8s-velero"}),
		}
	}
	units := state.RestoreState{
		Kind:       state.KindUnits,
		V2Dogus:    []string{"cas", "unlabeled"},
		V3Dogus:    []string{"postfix", "redmine"},
		Components: []string{"k8s-loki", "k8s-velero"},
	}

	tests := []struct {
		name               string
		restoreState       state.RestoreState
		spec               backupv1.RestoreSpec
		wantDogus          []string
		wantPVCs           []string
		wantConfigMaps     []string
		wantErrMsg         string
		wantNothingDeleted bool
	}{
		{
			name:           "legacy deletes all dogus and all resources with a backup scope",
			restoreState:   state.RestoreState{Kind: state.KindLegacy, Units: []state.UnitBackup{{}}},
			wantDogus:      nil,
			wantPVCs:       nil,
			wantConfigMaps: []string{"unscoped"},
		},
		{
			name:           "units delete only the restored dogus and components",
			restoreState:   units,
			spec:           backupv1.RestoreSpec{ExcludedDogus: []string{"postfix"}, ExcludedComponents: []string{"k8s-loki"}},
			wantDogus:      []string{"postfix", "unlabeled"},
			wantPVCs:       []string{"postfix-data"},
			wantConfigMaps: []string{"loki-config", "unscoped"},
		},
		{
			name:           "units without restored dogus skip the dogu cleanup",
			restoreState:   units,
			spec:           backupv1.RestoreSpec{ExcludedDogus: []string{"postfix", "redmine"}, Options: &backupv1.UnitOptions{IgnoreV2Dogus: true}},
			wantDogus:      []string{"cas", "postfix", "redmine", "unlabeled"},
			wantPVCs:       []string{"cas-data", "postfix-data"},
			wantConfigMaps: []string{"unscoped"},
		},
		{
			name:           "units without restored components skip the component resource cleanup",
			restoreState:   units,
			spec:           backupv1.RestoreSpec{ExcludedComponents: []string{"k8s-loki", "k8s-velero"}},
			wantDogus:      []string{"unlabeled"},
			wantPVCs:       nil,
			wantConfigMaps: []string{"loki-config", "unscoped", "velero-config"},
		},
		{
			name:               "unknown backup kind fails without deleting anything",
			restoreState:       state.RestoreState{Kind: state.KindNotFound},
			wantErrMsg:         "cannot clean up for a backup of kind \"NotFound\"",
			wantNothingDeleted: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k8sClient := newFakeClient(t, interceptor.Funcs{
				Delete: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.DeleteOption) error {
					if tt.wantNothingDeleted {
						t.Errorf("unexpected deletion of %s", obj.GetName())
					}
					return c.Delete(ctx, obj, opts...)
				},
			}, existingResources()...)

			err := newTestManager(k8sClient).Cleanup(t.Context(), tt.restoreState, tt.spec)

			if tt.wantErrMsg != "" {
				assert.ErrorContains(t, err, tt.wantErrMsg)
				return
			}
			require.NoError(t, err)
			assert.ElementsMatch(t, tt.wantDogus, existingNames(t, k8sClient, doguGVK), "dogus")
			assert.ElementsMatch(t, tt.wantPVCs, existingNames(t, k8sClient, pvcGVK), "pvcs")
			assert.ElementsMatch(t, tt.wantConfigMaps, existingNames(t, k8sClient, configMapGVK), "configmaps")
		})
	}
}

func TestDefaultCleanupManager_Cleanup_errors(t *testing.T) {
	restoreState := state.RestoreState{Kind: state.KindUnits, V3Dogus: []string{"cas"}, Components: []string{"k8s-velero"}}
	failListOf := func(kind string) interceptor.Funcs {
		return interceptor.Funcs{
			List: func(ctx context.Context, c client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
				if list.GetObjectKind().GroupVersionKind().Kind == kind {
					return assert.AnError
				}
				return c.List(ctx, list, opts...)
			},
		}
	}

	tests := []struct {
		name       string
		funcs      interceptor.Funcs
		wantErrMsg string
	}{
		{name: "fail to cleanup dogus", funcs: failListOf("Dogu"), wantErrMsg: "failed to cleanup dogus"},
		{name: "fail to cleanup dogu pvcs", funcs: failListOf("PersistentVolumeClaim"), wantErrMsg: "failed to cleanup dogu pvcs"},
		{name: "fail to cleanup component resources", funcs: failListOf("Secret"), wantErrMsg: "failed to cleanup component resources"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k8sClient := newFakeClient(t, tt.funcs)

			err := newTestManager(k8sClient).Cleanup(t.Context(), restoreState, backupv1.RestoreSpec{})

			assert.ErrorIs(t, err, assert.AnError)
			assert.ErrorContains(t, err, tt.wantErrMsg)
		})
	}

	t.Run("fail with timeout while a resource is not deleted", func(t *testing.T) {
		shortCleanupTimeout(t)
		k8sClient := newFakeClient(t, interceptor.Funcs{
			// the deletion is accepted, but the dogu stays, e.g. because of a finalizer
			Delete: func(context.Context, client.WithWatch, client.Object, ...client.DeleteOption) error { return nil },
		}, newDogu("cas"))

		err := newTestManager(k8sClient).Cleanup(t.Context(), restoreState, backupv1.RestoreSpec{})

		assert.ErrorContains(t, err, "cleanup timed out")
	})
}

// The cleanup only succeeds after all deleted resources are gone.
func TestDefaultCleanupManager_Cleanup_waitsForAllDeletions(t *testing.T) {
	var mu sync.Mutex
	pending := map[string]int{"cas": 3, "velero-config": 2}
	k8sClient := newFakeClient(t, interceptor.Funcs{
		Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
			mu.Lock()
			defer mu.Unlock()
			if pending[key.Name] > 0 {
				// the resource takes a few polls to disappear
				pending[key.Name]--
				return nil
			}
			return c.Get(ctx, key, obj, opts...)
		},
	}, newDogu("cas"), componentResource("velero-config", "k8s-velero"))
	restoreState := state.RestoreState{Kind: state.KindUnits, V3Dogus: []string{"cas"}, Components: []string{"k8s-velero"}}

	err := newTestManager(k8sClient).Cleanup(t.Context(), restoreState, backupv1.RestoreSpec{})

	require.NoError(t, err)
	assert.Equal(t, map[string]int{"cas": 0, "velero-config": 0}, pending)
}
