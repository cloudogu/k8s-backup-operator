package additionalimages

import (
	"context"

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
