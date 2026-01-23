package backup

type backupManager struct {
	createManager
	deleteManager
	statusSyncManager
}

// NewBackupManager creates a new instance of backupManager containing a createManager, deleteManager and statusSyncManager.
func NewBackupManager(k8sClient k8sClient, clientSet ecosystemInterface, blueprintClient blueprintInterface, namespace string, recorder eventRecorder, globalConfigRepository globalConfigRepository) *backupManager {
	creator := newBackupCreateManager(k8sClient, clientSet, blueprintClient, namespace, recorder, globalConfigRepository)
	remover := newBackupDeleteManager(k8sClient, clientSet, namespace, recorder)
	statusSyncManager := newBackupStatusSyncManager(k8sClient, namespace, recorder)
	return &backupManager{createManager: creator, deleteManager: remover, statusSyncManager: statusSyncManager}
}
