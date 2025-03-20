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
	cleanup cleanupManager,
	recreator ownerReferenceRestore,
) *defaultManager {
	return &defaultManager{
		createManager: newCreateManager(clientSet, namespace, recorder, cleanup, recreator),
		deleteManager: newDeleteManager(clientSet, namespace, recorder),
	}
}
