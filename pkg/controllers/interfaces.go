package controller

import (
	"k8s.io/client-go/tools/record"

	"github.com/cloudogu/k8s-backup-operator/pkg/api/ecosystem"
)

type ecosystemInterface interface {
	ecosystem.Interface
}

type eventRecorder interface {
	record.EventRecorder
}
