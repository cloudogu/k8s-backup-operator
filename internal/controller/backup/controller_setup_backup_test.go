package backup

import (
	"context"
	"maps"
	"reflect"
	"slices"
	"testing"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	blueprintv3 "github.com/cloudogu/k8s-blueprint-lib/v3/api/v3"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestControllerSetupBackup(t *testing.T) {
	t.Run("It should add default labels", func(t *testing.T) {
		backup := newBackupForControllerTest("ns", "backup")
		blueprintList := &blueprintv3.BlueprintList{Items: make([]blueprintv3.Blueprint, 0)}
		var updateCalled = false
		fakeClient := newFakeClientForControllerSetupBackupTest(t, backup, blueprintList, &updateCalled)
		controller := NewController(fakeClient, nil)

		err := controller.setupBackup(context.Background(), backup, "ns", logr.Discard())

		expectedLabels := map[string]string{}
		maps.Copy(expectedLabels, defaultLabels)

		assert.NoError(t, err)
		assert.True(t, reflect.DeepEqual(backup.Labels, expectedLabels))
		assert.True(t, updateCalled)
	})

	t.Run("It should keep existing label while adding the default labels", func(t *testing.T) {
		backup := newBackupForControllerTest("ns", "backup")
		backup.Labels = map[string]string{
			"example.com/key1": "value1",
			"example.com/key2": "value2",
		}
		blueprintList := &blueprintv3.BlueprintList{Items: make([]blueprintv3.Blueprint, 0)}
		var updateCalled = false
		fakeClient := newFakeClientForControllerSetupBackupTest(t, backup, blueprintList, &updateCalled)
		controller := NewController(fakeClient, nil)

		err := controller.setupBackup(context.Background(), backup, "ns", logr.Discard())

		expectedLabels := map[string]string{}
		maps.Copy(expectedLabels, backup.Labels)
		maps.Copy(expectedLabels, defaultLabels)

		assert.NoError(t, err)
		assert.True(t, reflect.DeepEqual(backup.Labels, expectedLabels))
		assert.True(t, updateCalled)
	})

	t.Run("It should add the finalizer", func(t *testing.T) {
		backup := newBackupForControllerTest("ns", "backup")
		blueprintList := &blueprintv3.BlueprintList{Items: make([]blueprintv3.Blueprint, 0)}
		var updateCalled = false
		fakeClient := newFakeClientForControllerSetupBackupTest(t, backup, blueprintList, &updateCalled)
		controller := NewController(fakeClient, nil)

		err := controller.setupBackup(context.Background(), backup, "ns", logr.Discard())

		assert.NoError(t, err)

		expectedFinalizers := []string{backupv1.BackupFinalizer}
		assert.ElementsMatch(t, backup.Finalizers, expectedFinalizers)
		assert.True(t, updateCalled)
	})

	t.Run("It should keep existing finalizers while adding the finalizer", func(t *testing.T) {
		backup := newBackupForControllerTest("ns", "backup")
		backup.Finalizers = []string{"finalizer01", "finalizer02"}
		blueprintList := &blueprintv3.BlueprintList{
			Items: []blueprintv3.Blueprint{},
		}
		var updateCalled = false
		fakeClient := newFakeClientForControllerSetupBackupTest(t, backup, blueprintList, &updateCalled)
		controller := NewController(fakeClient, nil)

		err := controller.setupBackup(context.Background(), backup, "ns", logr.Discard())

		assert.NoError(t, err)

		expectedFinalizers := []string{backupv1.BackupFinalizer, "finalizer01", "finalizer02"}
		assert.ElementsMatch(t, backup.Finalizers, expectedFinalizers)
		assert.True(t, updateCalled)
	})

	t.Run("should add an annotation with the blueprint's display name and dogus", func(t *testing.T) {
		backup := newBackupForControllerTest("ns", "backup")
		blueprintList := newBlueprintListForControllerSetupBackupTest(
			"ns",
			"blueprint",
			"blueprint-display-name",
			[]blueprintv3.Dogu{
				{Name: "dogu01"},
				{Name: "dogu02"},
			},
		)
		var updateCalled = false
		fakeClient := newFakeClientForControllerSetupBackupTest(t, backup, blueprintList, &updateCalled)
		controller := NewController(fakeClient, nil)

		err := controller.setupBackup(context.Background(), backup, "ns", logr.Discard())

		assert.NoError(t, err)
		assert.Equal(t, "blueprint-display-name", backup.Annotations[blueprintIdAnnotation])
		assert.JSONEq(t, `[{"name": "dogu01"}, {"name": "dogu02"}]`, backup.Annotations[blueprintDogusAnnotation])
		assert.True(t, updateCalled)
	})

	t.Run("should keep existing annotations while adding the annotations for the blueprint infos", func(t *testing.T) {
		backup := newBackupForControllerTest("ns", "backup")
		backup.Annotations = map[string]string{
			"example.com/anno1": "annoVal1",
			"example.com/anno2": "annoVal2",
		}
		blueprintList := newBlueprintListForControllerSetupBackupTest(
			"ns",
			"blueprint",
			"blueprint-display-name",
			[]blueprintv3.Dogu{
				{Name: "dogu01"},
				{Name: "dogu02"},
			},
		)
		var updateCalled = false
		fakeClient := newFakeClientForControllerSetupBackupTest(t, backup, blueprintList, &updateCalled)
		controller := NewController(fakeClient, nil)

		err := controller.setupBackup(context.Background(), backup, "ns", logr.Discard())

		assert.NoError(t, err)
		assert.ElementsMatch(t,
			[]string{"example.com/anno1", "example.com/anno2", blueprintIdAnnotation, blueprintDogusAnnotation},
			slices.Collect(maps.Keys(backup.Annotations)),
		)
		assert.True(t, updateCalled)
	})

}

func newFakeClientForControllerSetupBackupTest(t *testing.T, backup *backupv1.Backup, blueprintList *blueprintv3.BlueprintList, updateCalled *bool) client.WithWatch {
	return newFakeClientBuilder(t).
		WithObjects(backup).
		WithLists(blueprintList).
		WithInterceptorFuncs(interceptor.Funcs{
			Update: func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
				*updateCalled = true
				return client.Update(ctx, obj, opts...)
			},
		}).
		Build()
}

func newBlueprintListForControllerSetupBackupTest(
	namespace string,
	name string,
	displayName string,
	dogus []blueprintv3.Dogu,
) *blueprintv3.BlueprintList {
	blueprint := blueprintv3.Blueprint{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Spec: blueprintv3.BlueprintSpec{
			DisplayName: displayName,
			Blueprint: blueprintv3.BlueprintManifest{
				Dogus: dogus,
			},
		},
	}
	return &blueprintv3.BlueprintList{
		Items: []blueprintv3.Blueprint{blueprint},
	}
}
