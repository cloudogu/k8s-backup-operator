package backup

type backupManager struct {
	createManager
	deleteManager
	statusSyncManager
}

// NewBackupManager creates a new instance of backupManager containing a createManager, deleteManager and statusSyncManager.
func NewBackupManager(clientSet ecosystemInterface, namespace string, recorder eventRecorder, globalConfigRepository globalConfigRepository, ownerRefBackuper ownerReferenceBackup) *backupManager {
	creator := newBackupCreateManager(clientSet, namespace, recorder, globalConfigRepository, ownerRefBackuper)
	remover := newBackupDeleteManager(clientSet, namespace, recorder)
	statusSyncManager := newBackupStatusSyncManager(clientSet, namespace, recorder)
	return &backupManager{createManager: creator, deleteManager: remover, statusSyncManager: statusSyncManager}
}
