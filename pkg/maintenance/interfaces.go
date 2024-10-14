package maintenance

import (
	"context"
	"github.com/cloudogu/k8s-registry-lib/config"
	"k8s.io/apimachinery/pkg/watch"
	appsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

type maintenanceModeSwitch interface {
	// ActivateMaintenanceMode activates the maintenance mode.
	ActivateMaintenanceMode(ctx context.Context, title string, text string) error
	// DeactivateMaintenanceMode deactivates the maintenance mode.
	DeactivateMaintenanceMode(ctx context.Context) error
}

type statefulSetInterface interface {
	appsv1.StatefulSetInterface
}

type serviceInterface interface {
	corev1.ServiceInterface
}

// used for mocks

//nolint:unused
//goland:noinspection GoUnusedType
type watcher interface {
	watch.Interface
}

type globalConfigRepository interface {
	Get(ctx context.Context) (config.GlobalConfig, error)
	Update(ctx context.Context, globalConfig config.GlobalConfig) (config.GlobalConfig, error)
}
