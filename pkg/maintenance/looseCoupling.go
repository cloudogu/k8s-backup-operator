package maintenance

import (
	"context"
	"fmt"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var waitForEtcdTimeout = 5 * time.Minute

type looselyCoupledMaintenanceSwitch struct {
	maintenanceModeSwitch
	statefulSetClient statefulSetInterface
	serviceInterface  serviceInterface
}

// NewWithLooseCoupling creates a switch that checks if the configuration registry (e.g., etcd) exists before switching.
// If the registry does not exist, no switch is executed.
func NewWithLooseCoupling(globalConfigRepository globalConfigRepository, clientSet statefulSetInterface, serviceInterface serviceInterface) *looselyCoupledMaintenanceSwitch {
	return &looselyCoupledMaintenanceSwitch{
		maintenanceModeSwitch: New(globalConfigRepository),
		statefulSetClient:     clientSet,
		serviceInterface:      serviceInterface,
	}
}

// ActivateMaintenanceMode activates the maintenance mode if the etcd exists and is ready.
// This loose coupling enables us to perform restores on an empty cluster.
func (lcms *looselyCoupledMaintenanceSwitch) ActivateMaintenanceMode(ctx context.Context, title string, text string) error {
	if etcdReady, err := lcms.isEtcdReady(ctx); err != nil {
		return fmt.Errorf("failed to check if etcd is ready: %w", err)
	} else if etcdReady {
		return lcms.maintenanceModeSwitch.ActivateMaintenanceMode(ctx, title, text)
	}

	return nil
}

func (lcms *looselyCoupledMaintenanceSwitch) isEtcdReady(ctx context.Context) (bool, error) {
	statefulSet, err := lcms.statefulSetClient.Get(ctx, "etcd", metav1.GetOptions{})
	if err != nil {
		return checkReadyWithResourceNotFoundError(err, "etcd", "statefulset")
	}
	_, headLessServiceErr := lcms.serviceInterface.Get(ctx, "etcd-headless", metav1.GetOptions{})
	if headLessServiceErr != nil {
		return checkReadyWithResourceNotFoundError(headLessServiceErr, "etcd-headless", "service")
	}
	_, serviceErr := lcms.serviceInterface.Get(ctx, "etcd", metav1.GetOptions{})
	if serviceErr != nil {
		return checkReadyWithResourceNotFoundError(serviceErr, "etcd", "service")
	}

	if statefulSet.Status.ReadyReplicas >= 1 {
		return true, nil
	}
	return false, nil
}

func checkReadyWithResourceNotFoundError(err error, resource string, resourceType string) (bool, error) {
	if errors.IsNotFound(err) {
		return false, nil
	}
	return false, fmt.Errorf("failed to get %s [%s]: %w", resourceType, resource, err)
}

// DeactivateMaintenanceMode waits until the etcd is ready and then deactivates the maintenance mode.
// While this is not directly loose coupling, we trust that an instance of etcd will be restored.
func (lcms *looselyCoupledMaintenanceSwitch) DeactivateMaintenanceMode(ctx context.Context) error {
	err := lcms.waitForReadyEtcd(ctx)
	if err != nil {
		return fmt.Errorf("failed to wait for ready etcd: %w", err)
	}

	return lcms.maintenanceModeSwitch.DeactivateMaintenanceMode(ctx)
}

func (lcms *looselyCoupledMaintenanceSwitch) waitForReadyEtcd(ctx context.Context) error {
	waitCtx, cancelFunc := context.WithTimeout(ctx, waitForEtcdTimeout)
	defer cancelFunc()
	logger := log.FromContext(ctx)

	watch, err := lcms.statefulSetClient.Watch(ctx, metav1.ListOptions{FieldSelector: "metadata.name=etcd"})
	if err != nil {
		return fmt.Errorf("failed to create watch for StatefulSet etcd: %w", err)
	}

	defer watch.Stop()

	result := make(chan error, 1)
	go func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				result <- fmt.Errorf("waiting for etcd to become ready timed out")
				return
			case event := <-watch.ResultChan():
				statefulSet, ok := event.Object.(*appsv1.StatefulSet)
				if !ok {
					logger.Error(fmt.Errorf("unexpected type %T for watch on StatefulSet etcd; object: %#v", event.Object, event.Object), "wrong object type")
					continue
				}

				if statefulSet.Status.ReadyReplicas >= 1 {
					result <- nil
					return
				}
			}
		}
	}(waitCtx)

	return <-result
}
