package velero

type defaultManager struct {
	backupManager
	restoreManager
	syncManager
}

// NewDefaultManager creates a new instance of defaultManager.
func NewDefaultManager(veleroClientSet veleroClientSet, ecosystemClientSet ecosystemClientSet, recorder eventRecorder, namespace string) *defaultManager {
	return &defaultManager{
		backupManager:  NewDefaultBackupManager(veleroClientSet, recorder),
		restoreManager: NewDefaultRestoreManager(veleroClientSet, recorder),
		syncManager:    NewDefaultSyncManager(veleroClientSet, ecosystemClientSet, recorder, namespace),
	}
}
