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
	scaleManager scaleManager,
) *defaultManager {
	return &defaultManager{
		createManager: newCreateManager(k8sClient, clientSet, namespace, recorder, cleanup, scaleManager),
		deleteManager: newDeleteManager(k8sClient, clientSet, namespace, recorder),
	}
}
