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
	ownerRefRestorer ownerReferenceRestore,
) *defaultManager {
	return &defaultManager{
		createManager: newCreateManager(clientSet, namespace, recorder, cleanup, ownerRefRestorer),
		deleteManager: newDeleteManager(clientSet, namespace, recorder),
	}
}
