package backup

import (
	"context"
	"maps"
	"reflect"
	"testing"

	backupV1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestServiceConfigureBackup(t *testing.T) {
	var testCtx = context.TODO()

	t.Run("It should add default labels", func(t *testing.T) {
		_, _, srv := newTestFixture(t)
		backup := newBackupForControllerConfigureBackupTest()

		srv.configureBackup(testCtx, backup)

		expectedLabels := map[string]string{}
		maps.Copy(expectedLabels, defaultLabels)

		assert.True(t, reflect.DeepEqual(backup.Labels, expectedLabels))
	})

	t.Run("It should keep existing label while adding the default labels", func(t *testing.T) {
		_, _, srv := newTestFixture(t)
		backup := newBackupForControllerConfigureBackupTest()
		backup.Labels = map[string]string{
			"example.com/key1": "value1",
			"example.com/key2": "value2",
		}

		srv.configureBackup(testCtx, backup)

		expectedLabels := map[string]string{}
		maps.Copy(expectedLabels, backup.Labels)
		maps.Copy(expectedLabels, defaultLabels)

		assert.True(t, reflect.DeepEqual(backup.Labels, expectedLabels))
	})

	t.Run("It should add the finalizer", func(t *testing.T) {
		_, _, srv := newTestFixture(t)
		backup := newBackupForControllerConfigureBackupTest()

		srv.configureBackup(testCtx, backup)

		expectedFinalizers := []string{backupV1.BackupFinalizer}
		assert.ElementsMatch(t, backup.Finalizers, expectedFinalizers)
	})

	t.Run("It should keep existing finalizers while adding the finalizer", func(t *testing.T) {
		_, _, srv := newTestFixture(t)
		backup := newBackupForControllerConfigureBackupTest()
		backup.Finalizers = []string{}

		existingFinalizers := []string{"finalizer01", "finalizer02"}
		backup.Finalizers = append(backup.Finalizers, existingFinalizers...)

		srv.configureBackup(testCtx, backup)

		expectedFinalizers := []string{backupV1.BackupFinalizer}
		expectedFinalizers = append(expectedFinalizers, existingFinalizers...)

		assert.ElementsMatch(t, backup.Finalizers, expectedFinalizers)
	})

}

func TestServiceAddBlueprintAnnotations(t *testing.T) {

	t.Run("should add an annotation with the blueprint's display name and dogus", func(t *testing.T) {
		t.Skip("TODO")
	})

	t.Run("should keep existing annotations while adding the annotations for the bluprint infos", func(t *testing.T) {
		t.Skip("TODO")
	})
}

func TestServiceReconcileBackup(t *testing.T) {

	t.Run("It should set start time", func(t *testing.T) {
		t.Skip("TODO")
		/*
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

		*/
	})

	t.Run("It should not set start time if it is already set", func(t *testing.T) {
		t.Skip("TODO")

		/*
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

		*/
	})

	t.Run("It should activate the maintenance mode", func(t *testing.T) {
		t.Skip("TODO")

		/*
			blueprintGatewayMock.EXPECT().find(testCtx).Return(nil, nil)
			clockMock.EXPECT().Now().Return(time.Now())

			mock.InOrder(
				backupRepositoryMock.On("save", testCtx, mock.Anything).Return(nil),
				maintenanceGatewayMock.On("ActivateMaintenance", testCtx, backup2.maintenanceModeTitle, backup2.maintenanceModeText).Return(nil),
				veleroBackupRepositoryMock.On("save", testCtx, mock.Anything).Return(nil),
			)

			backup := Backup{Name: "backup10"}
			err := service.createBackup(testCtx, backup)

			assert.NoError(t, err)

		*/
	})

	t.Run("It should set condition for state in progress", func(t *testing.T) {
		t.Skip("TODO")
	})

}

func newTestFixture(t *testing.T) (client.Client, Clock, *ServiceImpl) {
	fakeClient := fake.NewClientBuilder().Build()
	clockMock := NewMockClock(t)
	serviceImpl := NewService(fakeClient, clockMock)
	return fakeClient, clockMock, serviceImpl
}

func newBackupForControllerConfigureBackupTest() *backupV1.Backup {
	return &backupV1.Backup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ns",
			Namespace: "backup",
		},
		Spec: backupV1.BackupSpec{
			Provider: "velero",
		},
	}
}
