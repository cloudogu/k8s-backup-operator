package cleanup

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"time"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/k8s-backup-operator/internal/controller/restore/state"
	"github.com/cloudogu/k8s-backup-operator/internal/logging"
	componentv1 "github.com/cloudogu/k8s-component-lib/api/v1"
	doguv3 "github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/selection"
)

const defaultWaitTime = time.Second * 3
const defaultCleanupTimeout = time.Minute * 30

var cleanupTimeout = defaultCleanupTimeout

type doguManager interface {
	cleanupDogus(ctx context.Context, wg *sync.WaitGroup, doguSelector, pvcSelector labels.Selector) error
}

type componentResourceManager interface {
	cleanupComponentResources(ctx context.Context, wg *sync.WaitGroup, selector labels.Selector) error
}

type DefaultCleanupManager struct {
	doguManager
	componentResourceManager
}

// NewManager creates a new instance of DefaultCleanupManager.
func NewManager(doguClient doguClient, dynamicClient dynamicClient, namespace string) *DefaultCleanupManager {
	pvcGroupVersionResource := corev1.SchemeGroupVersion.WithResource("persistentvolumeclaims")
	gvrs := []schema.GroupVersionResource{
		corev1.SchemeGroupVersion.WithResource("configmaps"),
		corev1.SchemeGroupVersion.WithResource("secrets"),
		pvcGroupVersionResource,
	}
	clients := make(map[schema.GroupVersionResource]unstructuredClient, len(gvrs))
	for _, gvr := range gvrs {
		clients[gvr] = dynamicClient.Resource(gvr).Namespace(namespace)
	}

	return &DefaultCleanupManager{
		doguManager:              newDoguManager(doguClient, clients[pvcGroupVersionResource]),
		componentResourceManager: newComponentResourceManager(clients),
	}
}

// Cleanup deletes all resources that need to be deleted before restoring the backup.
// It waits for the deletion of the resources to complete.
// If the deletion fails, the cleanup will fail.
// For a unit backup only the dogus and components which are restored are deleted; ignored or excluded ones are kept.
// For a legacy backup all dogus and all resources with a backup scope are deleted, because the whole backup is restored.
func (c *DefaultCleanupManager) Cleanup(ctx context.Context, restoreState state.RestoreState, spec backupv1.RestoreSpec) error {
	logging.Info(ctx, "starting cleanup before restore...", "kind", restoreState.Kind)

	selectors, err := newCleanupSelectors(restoreState, spec)
	if err != nil {
		return fmt.Errorf("failed to determine the resources to clean up: %w", err)
	}

	// Create a timeout context for the entire cleanup
	ctx, cancel := context.WithTimeout(ctx, cleanupTimeout)
	defer cancel()

	var wg sync.WaitGroup

	if selectors.dogus == nil {
		logging.Info(ctx, "no dogus to clean up")
	} else if err := c.cleanupDogus(ctx, &wg, selectors.dogus, selectors.doguPVCs); err != nil {
		return fmt.Errorf("failed to cleanup dogus: %w", err)
	}

	if selectors.componentResources == nil {
		logging.Info(ctx, "no components to clean up component resources for")
	} else if err := c.cleanupComponentResources(ctx, &wg, selectors.componentResources); err != nil {
		return fmt.Errorf("failed to cleanup component resources: %w", err)
	}

	// Wait for all goroutines OR timeout
	allDeletesDone := make(chan struct{})
	go func() {
		wg.Wait()
		close(allDeletesDone)
	}()

	select {
	case <-allDeletesDone:
		logging.Info(ctx, "... cleanup finished. All resources were deleted successfully.")
		return nil
	case <-ctx.Done():
		return fmt.Errorf("cleanup timed out: %w", ctx.Err())
	}
}

// cleanupSelectors select the resources to delete before the restore; a nil selector means nothing to delete.
type cleanupSelectors struct {
	dogus              labels.Selector
	doguPVCs           labels.Selector
	componentResources labels.Selector
}

func newCleanupSelectors(restoreState state.RestoreState, spec backupv1.RestoreSpec) (cleanupSelectors, error) {
	switch restoreState.Kind {
	case state.KindLegacy:
		return legacyCleanupSelectors()
	case state.KindUnits:
		v2Dogus, v3Dogus := restoreState.RestoredDogus(spec)
		return unitCleanupSelectors(slices.Concat(v2Dogus, v3Dogus), restoreState.RestoredComponents(spec))
	default:
		return cleanupSelectors{}, fmt.Errorf("cannot clean up for a backup of kind %q", restoreState.Kind)
	}
}

// legacyCleanupSelectors select all dogus and all resources with a backup scope.
// The pvcs of v2 dogus are deleted with their dogu; those of v3 dogus are selected by their dogu label.
func legacyCleanupSelectors() (cleanupSelectors, error) {
	doguPVCs, err := selectorOf(doguv3.DoguLabelName, selection.Exists)
	if err != nil {
		return cleanupSelectors{}, err
	}
	componentResources, err := selectorOf(backupScopeLabelKey, selection.Exists)
	if err != nil {
		return cleanupSelectors{}, err
	}

	return cleanupSelectors{dogus: labels.Everything(), doguPVCs: doguPVCs, componentResources: componentResources}, nil
}

// unitCleanupSelectors select the given dogus and the resources with a backup scope of the given components.
// A label selector with the operator "in" requires at least one value, so a selector without values stays nil.
func unitCleanupSelectors(dogus, components []string) (cleanupSelectors, error) {
	var selectors cleanupSelectors

	if len(dogus) > 0 {
		doguSelector, err := selectorOf(doguv3.DoguLabelName, selection.In, dogus...)
		if err != nil {
			return cleanupSelectors{}, err
		}
		selectors.dogus = doguSelector
		selectors.doguPVCs = doguSelector
	}

	if len(components) > 0 {
		componentResources, err := selectorOf(backupScopeLabelKey, selection.Exists)
		if err != nil {
			return cleanupSelectors{}, err
		}
		componentRequirement, err := labels.NewRequirement(componentv1.ComponentNameLabelKey, selection.In, components)
		if err != nil {
			return cleanupSelectors{}, fmt.Errorf("failed to create label selector: %w", err)
		}
		selectors.componentResources = componentResources.Add(*componentRequirement)
	}

	return selectors, nil
}

func selectorOf(key string, operator selection.Operator, values ...string) (labels.Selector, error) {
	requirement, err := labels.NewRequirement(key, operator, values)
	if err != nil {
		return nil, fmt.Errorf("failed to create label selector: %w", err)
	}
	return labels.NewSelector().Add(*requirement), nil
}
