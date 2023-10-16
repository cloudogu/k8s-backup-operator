package velero

type defaultRestoreManager struct {
}

// NewDefaultRestoreManager creates a new instance of defaultRestoreManager.
func NewDefaultRestoreManager() *defaultRestoreManager {
	return &defaultRestoreManager{}
}
