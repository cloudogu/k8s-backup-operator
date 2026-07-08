package backup

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestService(t *testing.T) {
	var testCtx = context.TODO()

	t.Run("It should add default labels", func(t *testing.T) {
		expectedBackup := Backup{
			Name: "backup1",
			Labels: map[string]string{
				"app":                      "ces",
				"k8s.cloudogu.com/part-of": "backup",
			},
		}

		backupRepositoryMock := newMockBackupRepository(t)
		backupRepositoryMock.EXPECT().save(expectedBackup).Return(nil)

		providerBackupRepositoryMock := newMockProviderBackupRepository(t)
		providerBackupRepositoryMock.EXPECT().save(expectedBackup).Return(nil)

		configGatewayMock := newMockConfigGateway(t)

		backup := Backup{
			Name: "backup1",
		}
		service := NewService(backupRepositoryMock, providerBackupRepositoryMock, configGatewayMock)

		err := service.createBackup(testCtx, backup)

		assert.NoError(t, err)
	})

	t.Run("It should keep existing label", func(t *testing.T) {
		expectedBackup := Backup{
			Name: "backup2",
			Labels: map[string]string{
				"app":                      "ces",
				"k8s.cloudogu.com/part-of": "backup",
				"example.com/key1":         "value1",
				"example.com/key2":         "value2",
			},
		}
		backupRepositoryMock := newMockBackupRepository(t)
		backupRepositoryMock.EXPECT().save(expectedBackup).Return(nil)

		providerBackupRepositoryMock := newMockProviderBackupRepository(t)
		providerBackupRepositoryMock.EXPECT().save(expectedBackup).Return(nil)

		configGatewayMock := newMockConfigGateway(t)

		backup := Backup{
			Name: "backup2",
			Labels: map[string]string{
				"example.com/key1": "value1",
				"example.com/key2": "value2",
			},
		}
		service := NewService(backupRepositoryMock, providerBackupRepositoryMock, configGatewayMock)

		err := service.createBackup(testCtx, backup)

		assert.NoError(t, err)
	})

	t.Run("It should throw an error if the backup repository throws an error", func(t *testing.T) {
		backupRepositoryMock := newMockBackupRepository(t)
		backupRepositoryMock.EXPECT().save(mock.Anything).Return(assert.AnError)

		providerBackupRepositoryMock := newMockProviderBackupRepository(t)
		configGatewayMock := newMockConfigGateway(t)

		backup := Backup{
			Name: "backup3",
		}
		service := NewService(backupRepositoryMock, providerBackupRepositoryMock, configGatewayMock)

		err := service.createBackup(testCtx, backup)

		assert.Error(t, err)
	})

	t.Run("It should throw an error if the provider backup repository throws an error", func(t *testing.T) {
		backupRepositoryMock := newMockBackupRepository(t)
		backupRepositoryMock.EXPECT().save(mock.Anything).Return(nil)

		providerBackupRepositoryMock := newMockProviderBackupRepository(t)
		providerBackupRepositoryMock.EXPECT().save(mock.Anything).Return(assert.AnError)

		configGatewayMock := newMockConfigGateway(t)

		backup := Backup{
			Name: "backup3",
		}
		service := NewService(backupRepositoryMock, providerBackupRepositoryMock, configGatewayMock)

		err := service.createBackup(testCtx, backup)

		assert.Error(t, err)
	})

	t.Run("It should add an annotation that contains information about the blueprint if it exists", func(t *testing.T) {
		t.Skip("TODO")
	})

	t.Run("It should add a finalizer", func(t *testing.T) {
		t.Skip("TODO")
	})

}
