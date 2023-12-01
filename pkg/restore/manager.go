package restore

type defaultManager struct {
	createManager
	deleteManager
}

// NewRestoreManager creates a new instance of defaultManager.
func NewRestoreManager(
	clientSet ecosystemInterface,
	namespace string,
	recorder eventRecorder,
	registry cesRegistry,
	cleanup cleanupManager,
) *defaultManager {
	return &defaultManager{
		createManager: newCreateManager(clientSet, namespace, recorder, registry, cleanup),
		deleteManager: newDeleteManager(clientSet, namespace, recorder),
	}
}
