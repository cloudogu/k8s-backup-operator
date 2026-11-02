package scheduledbackup

import (
	"context"

	operatortime "github.com/cloudogu/k8s-backup-operator/internal/time"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Manager interface {
	ScheduleBackup(ctx context.Context) error
}

type k8sClient interface {
	client.Client
}

type timeProvider interface {
	operatortime.TimeProvider
}
