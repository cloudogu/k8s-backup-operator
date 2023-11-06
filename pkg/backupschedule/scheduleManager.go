package backupschedule

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
