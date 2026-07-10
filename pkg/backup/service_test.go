package backup

import (
	"context"
	"maps"
	"reflect"
	"testing"
	"time"

	backupV1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/k8s-backup-operator/pkg/annotations"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestService(t *testing.T) {
	var testCtx = context.TODO()

	var backupRepositoryMock *mockBackupRepository
	var veleroBackupRepositoryMock *mockVeleroBackupRepository
	var configGatewayMock *mockConfigGateway
	var blueprintGatewayMock *mockBlueprintGateway
	var clockMock *MockClock
	var maintenanceGatewayMock *mockMaintenanceGateway
	var service *ServiceImpl

	run := func(name string, fn func(t *testing.T)) {
		backupRepositoryMock = newMockBackupRepository(t)
		veleroBackupRepositoryMock = newMockVeleroBackupRepository(t)
		configGatewayMock = newMockConfigGateway(t)
		blueprintGatewayMock = newMockBlueprintGateway(t)
		clockMock = NewMockClock(t)
		maintenanceGatewayMock = newMockMaintenanceGateway(t)
		service = NewService(
			backupRepositoryMock,
			veleroBackupRepositoryMock,
			configGatewayMock,
			blueprintGatewayMock,
			clockMock,
			maintenanceGatewayMock,
		)
		t.Run(name, fn)
	}

	run("It should add default labels", func(t *testing.T) {
		backupRepositoryMock.EXPECT().save(testCtx, mock.Anything).Run(func(context context.Context, backup Backup) {
			assert.Equal(t, "backup1", backup.Name)
			assert.True(t, reflect.DeepEqual(backup.Labels, defaultLabels))
		}).Return(nil)

		veleroBackupRepositoryMock.EXPECT().save(testCtx, mock.Anything).Run(func(context context.Context, backup Backup) {
			assert.Equal(t, "backup1", backup.Name)
			assert.True(t, reflect.DeepEqual(backup.Labels, defaultLabels))
		}).Return(nil)

		clockMock.EXPECT().Now().Return(time.Now())
		blueprintGatewayMock.EXPECT().find(testCtx).Return(nil, nil)
		maintenanceGatewayMock.EXPECT().ActivateMaintenance(testCtx, mock.Anything, mock.Anything).Return(nil)

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

		veleroBackupRepositoryMock.EXPECT().save(testCtx, mock.Anything).Run(func(_a0 context.Context, backup Backup) {
			assert.Equal(t, "backup2", backup.Name)
			assert.True(t, reflect.DeepEqual(backup.Labels, expectedLabels))
		}).Return(nil)

		clockMock.EXPECT().Now().Return(time.Now())
		blueprintGatewayMock.EXPECT().find(testCtx).Return(nil, nil)
		maintenanceGatewayMock.EXPECT().ActivateMaintenance(testCtx, mock.Anything, mock.Anything).Return(nil)

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

		veleroBackupRepositoryMock.EXPECT().save(testCtx, mock.Anything).Run(func(context context.Context, backup Backup) {
			assert.Equal(t, "backup3", backup.Name)
			assert.True(t, reflect.DeepEqual(backup.Annotations, expectedAnnotations))
		}).Return(nil)

		clockMock.EXPECT().Now().Return(time.Now())
		maintenanceGatewayMock.EXPECT().ActivateMaintenance(testCtx, mock.Anything, mock.Anything).Return(nil)

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

		veleroBackupRepositoryMock.EXPECT().save(testCtx, mock.Anything).Run(func(context context.Context, backup Backup) {
			assert.Equal(t, "backup4", backup.Name)
			assert.True(t, reflect.DeepEqual(backup.Annotations, expectedAnnotations))
		}).Return(nil)

		clockMock.EXPECT().Now().Return(time.Now())
		maintenanceGatewayMock.EXPECT().ActivateMaintenance(testCtx, mock.Anything, mock.Anything).Return(nil)

		blueprint := Blueprint{
			DisplayName:       "Blueprint",
			DogusAsJsonString: "[{}]",
		}
		blueprintGatewayMock.EXPECT().find(testCtx).Return(&blueprint, nil)

		err := service.createBackup(testCtx, backup)

		assert.NoError(t, err)
	})

	run("It should not add an annotation with blueprint infos if there is no blueprint", func(t *testing.T) {
		backupRepositoryMock.EXPECT().save(testCtx, mock.Anything).Run(func(context context.Context, backup Backup) {
			assert.Equal(t, "backup5", backup.Name)
			assert.Equal(t, 0, len(backup.Annotations))
		}).Return(nil)

		veleroBackupRepositoryMock.EXPECT().save(testCtx, mock.Anything).Run(func(context context.Context, backup Backup) {
			assert.Equal(t, "backup5", backup.Name)
			assert.Equal(t, 0, len(backup.Annotations))
		}).Return(nil)

		clockMock.EXPECT().Now().Return(time.Now())
		blueprintGatewayMock.EXPECT().find(testCtx).Return(nil, nil)
		maintenanceGatewayMock.EXPECT().ActivateMaintenance(testCtx, mock.Anything, mock.Anything).Return(nil)

		backup := Backup{Name: "backup5"}
		err := service.createBackup(testCtx, backup)

		assert.NoError(t, err)
	})

	run("It should throw an error if the backup repository throws an error", func(t *testing.T) {
		clockMock.EXPECT().Now().Return(time.Now())
		blueprintGatewayMock.EXPECT().find(testCtx).Return(nil, nil)
		maintenanceGatewayMock.EXPECT().ActivateMaintenance(testCtx, mock.Anything, mock.Anything).Return(nil)

		backupRepositoryMock.EXPECT().save(testCtx, mock.Anything).Return(assert.AnError)

		backup := Backup{Name: "backup6"}
		err := service.createBackup(testCtx, backup)

		assert.Error(t, err)
	})

	run("It should throw an error if the provider backup repository throws an error", func(t *testing.T) {
		backupRepositoryMock.EXPECT().save(testCtx, mock.Anything).Return(nil)
		clockMock.EXPECT().Now().Return(time.Now())
		blueprintGatewayMock.EXPECT().find(testCtx).Return(nil, nil)
		maintenanceGatewayMock.EXPECT().ActivateMaintenance(testCtx, mock.Anything, mock.Anything).Return(nil)

		veleroBackupRepositoryMock.EXPECT().save(testCtx, mock.Anything).Return(assert.AnError)

		backup := Backup{Name: "backup7"}
		err := service.createBackup(testCtx, backup)

		assert.Error(t, err)
	})

	run("It should throw an error if the blueprint gateway throws an error", func(t *testing.T) {
		clockMock.EXPECT().Now().Return(time.Now())
		maintenanceGatewayMock.EXPECT().ActivateMaintenance(testCtx, mock.Anything, mock.Anything).Return(nil)

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

		veleroBackupRepositoryMock.EXPECT().save(testCtx, mock.Anything).Run(func(context context.Context, backup Backup) {
			assert.Equal(t, "backup6", backup.Name)
			assert.ElementsMatch(t, backup.Finalizers, expectedFinalizers)
		}).Return(nil)

		clockMock.EXPECT().Now().Return(time.Now())
		blueprintGatewayMock.EXPECT().find(testCtx).Return(nil, nil)
		maintenanceGatewayMock.EXPECT().ActivateMaintenance(testCtx, mock.Anything, mock.Anything).Return(nil)

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

		veleroBackupRepositoryMock.EXPECT().save(testCtx, mock.Anything).Run(func(context context.Context, backup Backup) {
			assert.Equal(t, "backup7", backup.Name)
			assert.ElementsMatch(t, backup.Finalizers, expectedFinalizers)
		}).Return(nil)

		clockMock.EXPECT().Now().Return(time.Now())
		blueprintGatewayMock.EXPECT().find(testCtx).Return(nil, nil)
		maintenanceGatewayMock.EXPECT().ActivateMaintenance(testCtx, mock.Anything, mock.Anything).Return(nil)

		err := service.createBackup(testCtx, backup)

		assert.NoError(t, err)
	})

	run("It should set start time", func(t *testing.T) {
		timeNow := time.Now()
		clockMock.EXPECT().Now().Return(timeNow)

		backupRepositoryMock.EXPECT().save(testCtx, mock.Anything).Run(func(context context.Context, backup Backup) {
			assert.Equal(t, "backup8", backup.Name)
			assert.Equal(t, &timeNow, backup.StartTime)
		}).Return(nil)

		veleroBackupRepositoryMock.EXPECT().save(testCtx, mock.Anything).Run(func(context context.Context, backup Backup) {
			assert.Equal(t, "backup8", backup.Name)
			assert.Equal(t, &timeNow, backup.StartTime)
		}).Return(nil)

		blueprintGatewayMock.EXPECT().find(testCtx).Return(nil, nil)
		maintenanceGatewayMock.EXPECT().ActivateMaintenance(testCtx, mock.Anything, mock.Anything).Return(nil)

		backup := Backup{
			Name:      "backup8",
			StartTime: nil,
		}
		err := service.createBackup(testCtx, backup)

		assert.NoError(t, err)
	})

	run("It should not set start time if it is already set", func(t *testing.T) {
		startTime := time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC)

		backupRepositoryMock.EXPECT().save(testCtx, mock.Anything).Run(func(context context.Context, backup Backup) {
			assert.Equal(t, "backup9", backup.Name)
			assert.Equal(t, &startTime, backup.StartTime)
		}).Return(nil)

		veleroBackupRepositoryMock.EXPECT().save(testCtx, mock.Anything).Run(func(context context.Context, backup Backup) {
			assert.Equal(t, "backup9", backup.Name)
			assert.Equal(t, &startTime, backup.StartTime)
		}).Return(nil)

		blueprintGatewayMock.EXPECT().find(testCtx).Return(nil, nil)
		maintenanceGatewayMock.EXPECT().ActivateMaintenance(testCtx, mock.Anything, mock.Anything).Return(nil)

		backup := Backup{
			Name:      "backup9",
			StartTime: &startTime,
		}
		err := service.createBackup(testCtx, backup)

		assert.NoError(t, err)
	})

	run("It should activate the maintenance mode", func(t *testing.T) {
		blueprintGatewayMock.EXPECT().find(testCtx).Return(nil, nil)
		clockMock.EXPECT().Now().Return(time.Now())

		mock.InOrder(
			backupRepositoryMock.On("save", testCtx, mock.Anything).Return(nil),
			maintenanceGatewayMock.On("ActivateMaintenance", testCtx, maintenanceModeTitle, maintenanceModeText).Return(nil),
			veleroBackupRepositoryMock.On("save", testCtx, mock.Anything).Return(nil),
		)

		backup := Backup{Name: "backup10"}
		err := service.createBackup(testCtx, backup)

		assert.NoError(t, err)
	})

	t.Run("It should set condition for state in progress", func(t *testing.T) {
		t.Skip("TODO")
	})

}
