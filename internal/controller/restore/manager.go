package restore

type defaultManager struct {
	createManager
	deleteManager
}

// NewRestoreManager creates a new instance of defaultManager.
func NewRestoreManager(
	k8sClient k8sClient,
	namespace string,
	recorder eventRecorder,
	scaleManager scaleManager,
) *defaultManager {
	return &defaultManager{
		createManager: newCreateManager(k8sClient, namespace, recorder, scaleManager),
		deleteManager: newDeleteManager(k8sClient, namespace, recorder),
	}
}
