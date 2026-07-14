package backup

import (
	"context"
	"maps"
	"reflect"
	"slices"
	"testing"

	backupV1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	blueprintv3 "github.com/cloudogu/k8s-blueprint-lib/v3/api/v3"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestServiceConfigureBackup(t *testing.T) {
	t.Run("It should add default labels", func(t *testing.T) {
		srv := newTestFixture(t)
		backup := newBackupForServiceTest()

		srv.configureBackup(context.Background(), backup)

		expectedLabels := map[string]string{}
		maps.Copy(expectedLabels, defaultLabels)

		assert.True(t, reflect.DeepEqual(backup.Labels, expectedLabels))
	})

	t.Run("It should keep existing label while adding the default labels", func(t *testing.T) {
		srv := newTestFixture(t)
		backup := newBackupForServiceTest()
		backup.Labels = map[string]string{
			"example.com/key1": "value1",
			"example.com/key2": "value2",
		}

		srv.configureBackup(context.Background(), backup)

		expectedLabels := map[string]string{}
		maps.Copy(expectedLabels, backup.Labels)
		maps.Copy(expectedLabels, defaultLabels)

		assert.True(t, reflect.DeepEqual(backup.Labels, expectedLabels))
	})

	t.Run("It should add the finalizer", func(t *testing.T) {
		srv := newTestFixture(t)
		backup := newBackupForServiceTest()

		srv.configureBackup(context.Background(), backup)

		expectedFinalizers := []string{backupV1.BackupFinalizer}
		assert.ElementsMatch(t, backup.Finalizers, expectedFinalizers)
	})

	t.Run("It should keep existing finalizers while adding the finalizer", func(t *testing.T) {
		srv := newTestFixture(t)
		backup := newBackupForServiceTest()
		backup.Finalizers = []string{}

		existingFinalizers := []string{"finalizer01", "finalizer02"}
		backup.Finalizers = append(backup.Finalizers, existingFinalizers...)

		srv.configureBackup(context.Background(), backup)

		expectedFinalizers := []string{backupV1.BackupFinalizer}
		expectedFinalizers = append(expectedFinalizers, existingFinalizers...)

		assert.ElementsMatch(t, backup.Finalizers, expectedFinalizers)
	})

}

func TestServiceAddBlueprintAnnotations(t *testing.T) {
	t.Run("should add an annotation with the blueprint's display name and dogus", func(t *testing.T) {
		srv := newTestFixture(t)
		backup := newBackupForServiceTest()

		err := srv.addBlueprintAnnotation(
			context.Background(),
			backup,
			"blueprint-display-name",
			[]blueprintv3.Dogu{{Name: "dogu01"}, {Name: "dogu02"}},
		)

		assert.NoError(t, err)
		assert.Equal(t, "blueprint-display-name", backup.Annotations[blueprintIdAnnotation])
		assert.JSONEq(t, `[{"name": "dogu01"}, {"name": "dogu02"}]`, backup.Annotations[blueprintDogusAnnotation])
	})

	t.Run("should keep existing annotations while adding the annotations for the blueprint infos", func(t *testing.T) {
		srv := newTestFixture(t)
		backup := newBackupForServiceTest()
		backup.Annotations = map[string]string{
			"example.com/anno1": "annoVal1",
			"example.com/anno2": "annoVal2",
		}

		err := srv.addBlueprintAnnotation(
			context.Background(),
			backup,
			"blueprint-display-name",
			[]blueprintv3.Dogu{{Name: "dogu01"}, {Name: "dogu02"}},
		)

		assert.NoError(t, err)
		assert.ElementsMatch(t,
			[]string{"example.com/anno1", "example.com/anno2", blueprintIdAnnotation, blueprintDogusAnnotation},
			slices.Collect(maps.Keys(backup.Annotations)),
		)
	})
}

func TestServiceDeleteBackup(t *testing.T) {
	t.Run("If the backup is running don't delete it", func(t *testing.T) {
		t.Skip("TODO")
	})
}

func TestServiceReconcileBackup(t *testing.T) {
	t.Run("It should activate the maintenance mode", func(t *testing.T) {
		t.Skip("TODO")
	})

	t.Run("It should set start time", func(t *testing.T) {
		t.Skip("TODO")
	})

	t.Run("It should not set start time if it is already set", func(t *testing.T) {
		t.Skip("TODO")
	})

	t.Run("It should set condition for state in progress", func(t *testing.T) {
		t.Skip("TODO")
	})

	t.Run("should create velero backup", func(t *testing.T) {
		t.Skip("TODO")
	})

}

func newTestFixture(t *testing.T) *ServiceImpl {
	fakeClient := fake.NewClientBuilder().Build()
	clockMock := NewMockClock(t)
	serviceImpl := NewService(fakeClient, clockMock)
	return serviceImpl
}

func newBackupForServiceTest() *backupV1.Backup {
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
