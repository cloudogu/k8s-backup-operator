package restore

type defaultManager struct {
	deleteManager
}

// NewRestoreManager creates a new instance of defaultManager.
func NewRestoreManager(
	k8sClient k8sClient,
	namespace string,
	recorder eventRecorder,
) *defaultManager {
	return &defaultManager{
		deleteManager: newDeleteManager(k8sClient, namespace, recorder),
	}
}
