package backup

type backupManager struct {
	createManager
	deleteManager
}

// NewBackupManager creates a new instance of backupManager containing a createManager and deleteManager.
func NewBackupManager(backupClient ecosystemBackupInterface, recorder eventRecorder, registry etcdRegistry) *backupManager {
	creator := NewBackupCreateManager(backupClient, recorder, registry)
	remover := NewBackupDeleteManager(backupClient, recorder)
	return &backupManager{createManager: creator, deleteManager: remover}
}
