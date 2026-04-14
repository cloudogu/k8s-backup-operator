package backup

type backupManager struct {
	createManager
	deleteManager
	statusSyncManager
	timeoutManager
}

// NewBackupManager creates a new instance of backupManager containing a createManager, deleteManager and statusSyncManager.
func NewBackupManager(k8sClient k8sClient, clientSet ecosystemInterface, blueprintClient blueprintInterface, namespace string, recorder eventRecorder, backupTimeout int) *backupManager {
	creator := newBackupCreateManager(k8sClient, clientSet, blueprintClient, namespace, recorder, backupTimeout)
	remover := newBackupDeleteManager(k8sClient, clientSet, namespace, recorder)
	statusSyncManager := newBackupStatusSyncManager(k8sClient, namespace, recorder)
	timeouter := newBackupTimeoutManager(k8sClient, clientSet, namespace, recorder, backupTimeout)
	return &backupManager{createManager: creator, deleteManager: remover, statusSyncManager: statusSyncManager, timeoutManager: timeouter}
}
