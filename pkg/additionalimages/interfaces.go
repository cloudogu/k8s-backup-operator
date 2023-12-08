package additionalimages

import (
	"context"
	typedbatchv1 "k8s.io/client-go/kubernetes/typed/batch/v1"
	"k8s.io/client-go/tools/record"

	"github.com/cloudogu/k8s-backup-operator/pkg/api/ecosystem"
)

type ecosystemClientSet interface {
	ecosystem.Interface
}

type backupScheduleClient interface {
	ecosystem.BackupScheduleInterface
}

type eventRecorder interface {
	record.EventRecorder
}

type Getter interface {
	// ImageForKey returns a container image reference as found in OperatorAdditionalImagesConfigmapName.
	ImageForKey(ctx context.Context, key string) (string, error)
}

type Updater interface {
	// Update sets these images wherever they are needed.
	// E.g., the image used in the CronJob of a BackupSchedule.
	Update(ctx context.Context, config ImageConfig) error
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
