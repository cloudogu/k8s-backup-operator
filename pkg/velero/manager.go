package velero

type defaultManager struct {
	backupManager
	restoreManager
	syncManager
}

// NewDefaultManager creates a new instance of defaultManager.
func NewDefaultManager(k8sClient k8sWatchClient, discoveryClient discoveryClient, recorder eventRecorder, namespace string) *defaultManager {
	return &defaultManager{
		backupManager:  newDefaultBackupManager(k8sClient, recorder),
		restoreManager: newDefaultRestoreManager(k8sClient, discoveryClient, recorder),
		syncManager:    newDefaultSyncManager(k8sClient, recorder, namespace),
	}
}
