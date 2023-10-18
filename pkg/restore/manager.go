package restore

type defaultManager struct {
	createManager
	deleteManager
}

// NewRestoreManager creates a new instance of defaultManager.
func NewRestoreManager(restoreClient ecosystemRestoreInterface, recorder eventRecorder, registry cesRegistry) *defaultManager {
	return &defaultManager{
		createManager: newCreateManager(restoreClient, recorder, registry),
		deleteManager: newDeleteManager(restoreClient, recorder),
	}
}
