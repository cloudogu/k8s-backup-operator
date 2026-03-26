package backupschedule

import (
	"github.com/cloudogu/k8s-backup-operator/pkg/additionalimages"
	corev1 "k8s.io/api/core/v1"
)

// maxTries controls the maximum number of waiting intervals between tries when getting an error that is recoverable
// during k8s operations.
var maxTries = 5

type defaultManager struct {
	createManager
	updateManager
	deleteManager
}

func NewManager(clientSet ecosystemInterface, recorder eventRecorder, namespace string, imageConfig additionalimages.ImageConfig, imagePullSecrets []corev1.LocalObjectReference) *defaultManager {
	return &defaultManager{
		createManager: newCreateManager(clientSet, recorder, namespace, imageConfig, imagePullSecrets),
		updateManager: newUpdateManager(clientSet, recorder, namespace, imageConfig, imagePullSecrets),
		deleteManager: newDeleteManager(clientSet, recorder, namespace),
	}
}
