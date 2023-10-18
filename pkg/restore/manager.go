package restore

type defaultManager struct {
	createManager
	deleteManager
}

// NewRestoreManager creates a new instance of defaultManager.
func NewRestoreManager(clientSet ecosystemInterface, recorder eventRecorder, registry cesRegistry) *defaultManager {
	return &defaultManager{
		createManager: newCreateManager(clientSet, recorder, registry),
		deleteManager: newDeleteManager(clientSet, recorder),
	}
}
