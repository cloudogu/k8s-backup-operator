package backupschedule

type defaultManager struct {
	createManager
	updateManager
	deleteManager
}

func NewManager(clientSet ecosystemInterface, recorder eventRecorder, namespace string) *defaultManager {
	return &defaultManager{
		createManager: newCreateManager(clientSet, recorder, namespace),
		updateManager: newUpdateManager(clientSet, recorder, namespace),
		deleteManager: newDeleteManager(clientSet, recorder, namespace),
	}
}
