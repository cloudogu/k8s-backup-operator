package restore

type defaultManager struct {
	createManager
	deleteManager
}

func NewRestoreManager(clientSet ecosystemInterface, recorder eventRecorder) *defaultManager {
	return &defaultManager{
		createManager: newCreateManager(clientSet, recorder),
		deleteManager: newDeleteManager(clientSet, recorder),
	}
}
