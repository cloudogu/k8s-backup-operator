package backup

type backupManager struct {
	createManager
	deleteManager
}

// NewBackupManager creates a new instance of backupManager containing a createManager and deleteManager.
func NewBackupManager(clientSet ecosystemInterface, namespace string, recorder eventRecorder, registry etcdRegistry) *backupManager {
	creator := NewBackupCreateManager(clientSet, namespace, recorder, registry)
	remover := NewBackupDeleteManager(clientSet, namespace, recorder)
	return &backupManager{createManager: creator, deleteManager: remover}
}
