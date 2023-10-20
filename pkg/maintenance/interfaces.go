package maintenance

import (
	"github.com/cloudogu/cesapp-lib/registry"
	"k8s.io/apimachinery/pkg/watch"
	appsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

type maintenanceModeSwitch interface {
	// ActivateMaintenanceMode activates the maintenance mode.
	ActivateMaintenanceMode(title string, text string) error
	// DeactivateMaintenanceMode deactivates the maintenance mode.
	DeactivateMaintenanceMode() error
}

type statefulSetInterface interface {
	appsv1.StatefulSetInterface
}

type serviceInterface interface {
	corev1.ServiceInterface
}

type globalConfig interface {
	registry.ConfigurationContext
}

// used for mocks

//nolint:unused
//goland:noinspection GoUnusedType
type watcher interface {
	watch.Interface
}
