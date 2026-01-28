package restore

type defaultManager struct {
	createManager
	deleteManager
}

// NewRestoreManager creates a new instance of defaultManager.
func NewRestoreManager(
	k8sClient k8sClient,
	clientSet ecosystemInterface,
	namespace string,
	recorder eventRecorder,
	cleanup cleanupManager,
) *defaultManager {
	return &defaultManager{
		createManager: newCreateManager(k8sClient, clientSet, namespace, recorder, cleanup),
		deleteManager: newDeleteManager(k8sClient, clientSet, namespace, recorder),
	}
}
