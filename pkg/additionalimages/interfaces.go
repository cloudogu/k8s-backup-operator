package additionalimages

import (
	"context"
	typedbatchv1 "k8s.io/client-go/kubernetes/typed/batch/v1"

	"github.com/cloudogu/k8s-backup-operator/pkg/api/ecosystem"
)

type ecosystemClientSet interface {
	ecosystem.Interface
}

type backupScheduleClient interface {
	ecosystem.BackupScheduleInterface
}

type Getter interface {
	// ImageForKey returns a container image reference as found in OperatorAdditionalImagesConfigmapName.
	ImageForKey(ctx context.Context, key string) (string, error)
}

type Updater interface {
	// Update sets the newest additional images wherever they are needed.
	// E.g., the kubectl image used in the CronJob of a BackupSchedule.
	Update(ctx context.Context) error
}

//nolint:unused
//goland:noinspection GoUnusedType
type ecosystemV1Alpha1Interface interface {
	ecosystem.V1Alpha1Interface
}

//nolint:unused
//goland:noinspection GoUnusedType
type batchV1Interface interface {
	typedbatchv1.BatchV1Interface
}

//nolint:unused
//goland:noinspection GoUnusedType
type cronJobInterface interface {
	typedbatchv1.CronJobInterface
}
