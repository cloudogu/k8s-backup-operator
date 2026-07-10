package backup

import (
	"context"
)

type backupRepositoryImpl struct {
	backupInterface ecosystemBackupInterface
}

func NewBackupRespository(backupInterface ecosystemBackupInterface) *backupRepositoryImpl {
	return &backupRepositoryImpl{
		backupInterface: backupInterface,
	}
}

func (b backupRepositoryImpl) save(context context.Context, backup Backup) error {
	//backupCr, err := b.backupInterface.Get(context, backup.Name, v1.GetOptions{})
	//TODO implement me
	panic("implement me")
}
