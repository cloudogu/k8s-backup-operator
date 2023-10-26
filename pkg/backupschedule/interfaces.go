package backupschedule

import (
	"context"

	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/cloudogu/k8s-backup-operator/pkg/api/ecosystem"
	v1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
)

type ecosystemInterface interface {
	ecosystem.Interface
}

type eventRecorder interface {
	record.EventRecorder
}

type requeueHandler interface {
	Handle(ctx context.Context, contextMessage string, restore v1.RequeuableObject, originalErr error, requeueStatus string) (ctrl.Result, error)
}