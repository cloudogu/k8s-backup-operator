package velero

import (
	"k8s.io/client-go/tools/record"

	"github.com/vmware-tanzu/velero/pkg/generated/clientset/versioned"
	velerov1 "github.com/vmware-tanzu/velero/pkg/generated/clientset/versioned/typed/velero/v1"
)

type eventRecorder interface {
	record.EventRecorder
}

type veleroClientSet interface {
	versioned.Interface
}

// The following interfaces are here to generate mocks.

type veleroInterface interface {
	velerov1.VeleroV1Interface
}

type veleroBackupInterface interface {
	velerov1.BackupInterface
}
