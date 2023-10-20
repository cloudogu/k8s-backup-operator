package restore

type defaultManager struct {
	createManager
	deleteManager
}

// NewRestoreManager creates a new instance of defaultManager.
func NewRestoreManager(
	restoreClient ecosystemRestoreInterface,
	recorder eventRecorder,
	registry cesRegistry,
	statefulSetClient statefulSetInterface,
	serviceInterface serviceInterface,
	cleanup cleanupManager,
) *defaultManager {
	return &defaultManager{
		createManager: newCreateManager(restoreClient, recorder, registry, statefulSetClient, serviceInterface, cleanup),
		deleteManager: newDeleteManager(restoreClient, recorder),
	}
}
