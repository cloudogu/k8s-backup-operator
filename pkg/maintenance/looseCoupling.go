package maintenance

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type looselyCoupledMaintenanceSwitch struct {
	maintenanceModeSwitch
	namespace string
	clientSet ecosystemInterface
}

// NewWithLooseCoupling creates a switch that checks if the configuration registry (e.g., etcd) exists before switching.
// If the registry does not exist, no switch is executed.
func NewWithLooseCoupling(globalConfig globalConfig, namespace string, clientSet ecosystemInterface) *looselyCoupledMaintenanceSwitch {
	return &looselyCoupledMaintenanceSwitch{
		maintenanceModeSwitch: New(globalConfig),
		namespace:             namespace,
		clientSet:             clientSet,
	}
}

// ActivateMaintenanceMode activates the maintenance mode if the etcd exists and is ready.
// This loose coupling enables us to perform restores on an empty cluster.
func (lcms *looselyCoupledMaintenanceSwitch) ActivateMaintenanceMode(title string, text string) error {
	if etcdReady, err := lcms.isEtcdReady(); err != nil {
		return err
	} else if etcdReady {
		return lcms.maintenanceModeSwitch.ActivateMaintenanceMode(title, text)
	}

	return nil
}

func (lcms *looselyCoupledMaintenanceSwitch) isEtcdReady() (bool, error) {
	statefulSet, err := lcms.clientSet.AppsV1().StatefulSets(lcms.namespace).Get(context.Background(), "etcd", metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to get StatefulSet etcd: %w", err)
	}

	if statefulSet.Status.ReadyReplicas >= 1 {
		return true, nil
	}
	return false, nil
}

// DeactivateMaintenanceMode waits until the etcd is ready and then deactivates the maintenance mode.
// While this is not directly loose coupling, we trust that an instance of etcd will be restored.
func (lcms *looselyCoupledMaintenanceSwitch) DeactivateMaintenanceMode() error {
	err := lcms.waitForReadyEtcd()
	if err != nil {
		return fmt.Errorf("failed to wait for ready etcd: %w", err)
	}

	return lcms.maintenanceModeSwitch.DeactivateMaintenanceMode()
}

func (lcms *looselyCoupledMaintenanceSwitch) waitForReadyEtcd() error {
	ctx, cancelFunc := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancelFunc()

	watch, err := lcms.clientSet.AppsV1().StatefulSets(lcms.namespace).Watch(context.Background(), metav1.ListOptions{
		FieldSelector: "metadata.name=etcd",
	})
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
					result <- fmt.Errorf("unexpected type %T for watch on StatefulSet etcd; object: %#v", event.Object, event.Object)
					return
				}

				if statefulSet.Status.ReadyReplicas >= 1 {
					result <- nil
					return
				}
			}
		}
	}(ctx)

	return <-result
}
