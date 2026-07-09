package backup

import (
	"context"
	"maps"
	"reflect"
	"testing"

	backupV1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/k8s-backup-operator/pkg/annotations"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestService(t *testing.T) {
	var testCtx = context.TODO()

	var backupRepositoryMock *mockBackupRepository
	var providerServiceMock *MockService
	var configGatewayMock *mockConfigGateway
	var blueprintGatewayMock *mockBlueprintGateway
	var service *ServiceImpl

	run := func(name string, fn func(t *testing.T)) {
		backupRepositoryMock = newMockBackupRepository(t)
		providerServiceMock = NewMockService(t)
		configGatewayMock = newMockConfigGateway(t)
		blueprintGatewayMock = newMockBlueprintGateway(t)
		service = NewService(backupRepositoryMock, providerServiceMock, configGatewayMock, blueprintGatewayMock)
		t.Run(name, fn)
	}

	run("It should add default labels", func(t *testing.T) {
		backupRepositoryMock.EXPECT().save(testCtx, mock.Anything).Run(func(context context.Context, backup Backup) {
			assert.Equal(t, "backup1", backup.Name)
			assert.True(t, reflect.DeepEqual(backup.Labels, defaultLabels))
		}).Return(nil)

		providerServiceMock.EXPECT().createBackup(testCtx, mock.Anything).Run(func(context context.Context, backup Backup) {
			assert.Equal(t, "backup1", backup.Name)
			assert.True(t, reflect.DeepEqual(backup.Labels, defaultLabels))
		}).Return(nil)

		blueprintGatewayMock.EXPECT().find(testCtx).Return(nil, nil)

		backup := Backup{Name: "backup1"}
		err := service.createBackup(testCtx, backup)

		assert.NoError(t, err)
	})

	run("It should keep existing label while adding default labels", func(t *testing.T) {
		expectedLabels := map[string]string{
			"example.com/key1": "value1",
			"example.com/key2": "value2",
		}
		maps.Copy(expectedLabels, defaultLabels)

		backupRepositoryMock.EXPECT().save(testCtx, mock.Anything).Run(func(context context.Context, backup Backup) {
			assert.Equal(t, "backup2", backup.Name)
			assert.True(t, reflect.DeepEqual(backup.Labels, expectedLabels))
		}).Return(nil)

		providerServiceMock.EXPECT().createBackup(testCtx, mock.Anything).Run(func(_a0 context.Context, backup Backup) {
			assert.Equal(t, "backup2", backup.Name)
			assert.True(t, reflect.DeepEqual(backup.Labels, expectedLabels))
		}).Return(nil)

		blueprintGatewayMock.EXPECT().find(testCtx).Return(nil, nil)

		backup := Backup{
			Name: "backup2",
			Labels: map[string]string{
				"example.com/key1": "value1",
				"example.com/key2": "value2",
			},
		}
		err := service.createBackup(testCtx, backup)

		assert.NoError(t, err)
	})

	run("It should add an annotation with blueprint infos if a blueprint exists", func(t *testing.T) {
		expectedAnnotations := map[string]string{
			annotations.BlueprintIdAnnotation: "Blueprint",
			annotations.DogusAnnotation:       "[{}]",
		}

		backupRepositoryMock.EXPECT().save(testCtx, mock.Anything).Run(func(context context.Context, backup Backup) {
			assert.Equal(t, "backup3", backup.Name)
			assert.True(t, reflect.DeepEqual(backup.Annotations, expectedAnnotations))
		}).Return(nil)

		providerServiceMock.EXPECT().createBackup(testCtx, mock.Anything).Run(func(context context.Context, backup Backup) {
			assert.Equal(t, "backup3", backup.Name)
			assert.True(t, reflect.DeepEqual(backup.Annotations, expectedAnnotations))
		}).Return(nil)

		blueprint := Blueprint{
			DisplayName:       "Blueprint",
			DogusAsJsonString: "[{}]",
		}
		blueprintGatewayMock.EXPECT().find(testCtx).Return(&blueprint, nil)

		backup := Backup{Name: "backup3"}
		err := service.createBackup(testCtx, backup)

		assert.NoError(t, err)
	})

	run("It should keep existing annotations while adding annotations for blueprint infos", func(t *testing.T) {
		backup := Backup{
			Name: "backup4",
			Annotations: map[string]string{
				"example.com/annoKey1": "annoVal1",
				"example.com/annoKey2": "annoVal2",
			},
		}

		expectedAnnotations := map[string]string{
			annotations.BlueprintIdAnnotation: "Blueprint",
			annotations.DogusAnnotation:       "[{}]",
		}
		maps.Copy(expectedAnnotations, backup.Annotations)

		backupRepositoryMock.EXPECT().save(testCtx, mock.Anything).Run(func(context context.Context, backup Backup) {
			assert.Equal(t, "backup4", backup.Name)
			assert.True(t, reflect.DeepEqual(backup.Annotations, expectedAnnotations))
		}).Return(nil)

		providerServiceMock.EXPECT().createBackup(testCtx, mock.Anything).Run(func(context context.Context, backup Backup) {
			assert.Equal(t, "backup4", backup.Name)
			assert.True(t, reflect.DeepEqual(backup.Annotations, expectedAnnotations))
		}).Return(nil)

		blueprint := Blueprint{
			DisplayName:       "Blueprint",
			DogusAsJsonString: "[{}]",
		}
		blueprintGatewayMock.EXPECT().find(testCtx).Return(&blueprint, nil)

		err := service.createBackup(testCtx, backup)

		assert.NoError(t, err)
	})

	run("It should not add an annotation with blueprint infos if there is not blueprint", func(t *testing.T) {
		backupRepositoryMock.EXPECT().save(testCtx, mock.Anything).Run(func(context context.Context, backup Backup) {
			assert.Equal(t, "backup5", backup.Name)
			assert.Equal(t, 0, len(backup.Annotations))
		}).Return(nil)

		providerServiceMock.EXPECT().createBackup(testCtx, mock.Anything).Run(func(context context.Context, backup Backup) {
			assert.Equal(t, "backup5", backup.Name)
			assert.Equal(t, 0, len(backup.Annotations))
		}).Return(nil)

		blueprintGatewayMock.EXPECT().find(testCtx).Return(nil, nil)

		backup := Backup{Name: "backup5"}
		err := service.createBackup(testCtx, backup)

		assert.NoError(t, err)
	})

	run("It should throw an error if the backup repository throws an error", func(t *testing.T) {
		backupRepositoryMock.EXPECT().save(testCtx, mock.Anything).Return(assert.AnError)
		blueprintGatewayMock.EXPECT().find(testCtx).Return(nil, nil)

		backup := Backup{Name: "backup6"}
		err := service.createBackup(testCtx, backup)

		assert.Error(t, err)
	})

	run("It should throw an error if the provider backup repository throws an error", func(t *testing.T) {
		backupRepositoryMock.EXPECT().save(testCtx, mock.Anything).Return(nil)
		providerServiceMock.EXPECT().createBackup(testCtx, mock.Anything).Return(assert.AnError)
		blueprintGatewayMock.EXPECT().find(testCtx).Return(nil, nil)

		backup := Backup{Name: "backup7"}
		err := service.createBackup(testCtx, backup)

		assert.Error(t, err)
	})

	run("It should throw an error if the blueprint gateway throws an error", func(t *testing.T) {
		blueprintGatewayMock.EXPECT().find(testCtx).Return(nil, assert.AnError)

		backup := Backup{Name: "backup8"}
		err := service.createBackup(testCtx, backup)

		assert.Error(t, err)
	})

	run("It should add the finalizer", func(t *testing.T) {
		expectedFinalizers := []string{backupV1.BackupFinalizer}

		backupRepositoryMock.EXPECT().save(testCtx, mock.Anything).Run(func(context context.Context, backup Backup) {
			assert.Equal(t, "backup6", backup.Name)
			assert.ElementsMatch(t, backup.Finalizers, expectedFinalizers)
		}).Return(nil)

		providerServiceMock.EXPECT().createBackup(testCtx, mock.Anything).Run(func(context context.Context, backup Backup) {
			assert.Equal(t, "backup6", backup.Name)
			assert.ElementsMatch(t, backup.Finalizers, expectedFinalizers)
		}).Return(nil)

		blueprintGatewayMock.EXPECT().find(testCtx).Return(nil, nil)

		backup := Backup{Name: "backup6"}
		err := service.createBackup(testCtx, backup)

		assert.NoError(t, err)
	})

	run("It should keep existing finalizers while adding the finalizer", func(t *testing.T) {
		backup := Backup{
			Name:       "backup7",
			Finalizers: []string{"finalizer01", "finalizer02"},
		}
		expectedFinalizers := []string{backupV1.BackupFinalizer}
		expectedFinalizers = append(expectedFinalizers, backup.Finalizers...)

		backupRepositoryMock.EXPECT().save(testCtx, mock.Anything).Run(func(context context.Context, backup Backup) {
			assert.Equal(t, "backup7", backup.Name)
			assert.ElementsMatch(t, backup.Finalizers, expectedFinalizers)
		}).Return(nil)

		providerServiceMock.EXPECT().createBackup(testCtx, mock.Anything).Run(func(context context.Context, backup Backup) {
			assert.Equal(t, "backup7", backup.Name)
			assert.ElementsMatch(t, backup.Finalizers, expectedFinalizers)
		}).Return(nil)

		blueprintGatewayMock.EXPECT().find(testCtx).Return(nil, nil)

		err := service.createBackup(testCtx, backup)

		assert.NoError(t, err)
	})

	t.Run("It should activate the maintenance mode", func(t *testing.T) {
		t.Skip("TODO")
	})

}
