package backupschedule

// maxTries controls the maximum number of waiting intervals between tries when getting an error that is recoverable
// during k8s operations.
var maxTries = 5

type defaultManager struct {
	createManager
	updateManager
	deleteManager
}

func NewManager(clientSet ecosystemInterface, recorder eventRecorder, namespace string) *defaultManager {
	return &defaultManager{
		createManager: newScheduleCreateManager(clientSet, recorder, namespace),
		updateManager: newScheduleUpdateManager(clientSet, recorder, namespace),
		deleteManager: newScheduleDeleteManager(clientSet, recorder, namespace),
	}
}
