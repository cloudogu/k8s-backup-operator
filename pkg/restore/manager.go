package restore

type defaultManager struct {
	createManager
	deleteManager
}

// NewRestoreManager creates a new instance of defaultManager.
func NewRestoreManager(restoreClient ecosystemRestoreInterface, backupClient ecosystemBackupInterface, recorder eventRecorder, registry cesRegistry) *defaultManager {
	return &defaultManager{
		createManager: newCreateManager(restoreClient, backupClient, recorder, registry),
		deleteManager: newDeleteManager(restoreClient, recorder),
	}
}
