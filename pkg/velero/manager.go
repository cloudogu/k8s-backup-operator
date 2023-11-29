package velero

type defaultManager struct {
	backupManager
	restoreManager
	syncManager
}

// NewDefaultManager creates a new instance of defaultManager.
func NewDefaultManager(veleroClientSet veleroClientSet, ecosystemClientSet ecosystemClientSet, recorder eventRecorder, namespace string) *defaultManager {
	return &defaultManager{
		backupManager:  newDefaultBackupManager(veleroClientSet, recorder),
		restoreManager: newDefaultRestoreManager(veleroClientSet, recorder),
		syncManager:    newDefaultSyncManager(veleroClientSet, ecosystemClientSet, recorder, namespace),
	}
}
