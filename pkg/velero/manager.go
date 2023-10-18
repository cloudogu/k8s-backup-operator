package velero

type defaultManager struct {
	backupManager
	restoreManager
}

// NewDefaultManager creates a new instance of defaultManager.
func NewDefaultManager(veleroClientSet veleroClientSet, recorder eventRecorder) *defaultManager {
	return &defaultManager{
		backupManager:  NewDefaultBackupManager(veleroClientSet, recorder),
		restoreManager: NewDefaultRestoreManager(veleroClientSet, recorder),
	}
}
