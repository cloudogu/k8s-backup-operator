package cleanup

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cloudogu/k8s-backup-operator/internal/controller/restore/state"
	"github.com/cloudogu/k8s-backup-operator/internal/logging"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const defaultWaitTime = time.Second * 3
const defaultCleanupTimeout = time.Minute * 30

var cleanupTimeout = defaultCleanupTimeout

type doguManager interface {
	cleanupDogus(ctx context.Context, wg *sync.WaitGroup, v2Dogus, v3Dogus []string) error
}

type additionalResourceManager interface {
	cleanupAdditionalResources(ctx context.Context, wg *sync.WaitGroup) error
}

type DefaultCleanupManager struct {
	doguManager
	additionalResourceManager
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
		doguManager:               newDoguManager(doguClient, clients[pvcGroupVersionResource]),
		additionalResourceManager: newAdditionalResourceManager(clients),
	}
}

// Was passiert, wenn ein legacy Backup eingespielt wird?
// Muss ebenfalls Liste von Dogus liefern -> Diese müssen gelöscht werden inkl. pvcs und additional resources.
// Das legacy Backups muss preisgeben, ob legacy resources enthalten sind.

// Cleanup deletes all resources that need to be deleted before restoring the backup.
// It waits for the deletion of the resources to complete.
// If the deletion fails, the cleanup will fail.
func (c *DefaultCleanupManager) Cleanup(ctx context.Context, state state.RestoreState) error {
	logging.Info(ctx, "starting cleanup before restore...")

	// List of Dogus and Components (additional resources)

	// Create a timeout context for the entire cleanup
	ctx, cancel := context.WithTimeout(ctx, cleanupTimeout)
	defer cancel()

	var wg sync.WaitGroup

	// Pass DoguLists to the function

	err := c.cleanupDogus(ctx, &wg, state.V2Dogus, state.V3Dogus)
	if err != nil {
		return fmt.Errorf("failed to cleanup dogus: %w", err)
	}

	// Pass ComponentList to the function and use component.name label selector with backup-scope together
	// Option: We could pass the dogu v3 list too and delete orphaned resources from them

	err = c.cleanupAdditionalResources(ctx, &wg)
	if err != nil {
		return fmt.Errorf("failed to cleanup additional resources: %w", err)
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
