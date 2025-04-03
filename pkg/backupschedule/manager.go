package backupschedule

import "github.com/cloudogu/k8s-backup-operator/pkg/additionalimages"

// maxTries controls the maximum number of waiting intervals between tries when getting an error that is recoverable
// during k8s operations.
var maxTries = 5

type defaultManager struct {
	createManager
	updateManager
	deleteManager
}

func NewManager(clientSet ecosystemInterface, recorder eventRecorder, namespace string, imageConfig additionalimages.ImageConfig) *defaultManager {
	return &defaultManager{
		createManager: newCreateManager(clientSet, recorder, namespace, imageConfig),
		updateManager: newUpdateManager(clientSet, recorder, namespace, imageConfig),
		deleteManager: newDeleteManager(clientSet, recorder, namespace),
	}
}
