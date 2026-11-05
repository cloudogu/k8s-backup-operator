package backup

import (
	"context"
	"maps"
	"reflect"
	"testing"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	blueprintv3 "github.com/cloudogu/k8s-blueprint-lib/v3/api/v3"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestEnsureBackupSetup(t *testing.T) {
	t.Run("It should add default labels", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		blueprintList := &blueprintv3.BlueprintList{Items: make([]blueprintv3.Blueprint, 0)}
		var updateCalled = false
		fakeClient := newFakeClientForEnsureBackupSetupTest(t, backup, blueprintList, &updateCalled)
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default", "velero")

		nextAction, err := reconciler.ensureBackupMetadata(context.Background(), backup)

		expectedLabels := map[string]string{}
		maps.Copy(expectedLabels, defaultLabels)

		assert.NoError(t, err)
		assert.Equal(t, Next, nextAction)
		assert.True(t, reflect.DeepEqual(backup.Labels, expectedLabels))
		assert.True(t, updateCalled)
	})

	t.Run("It should keep existing label while adding the default labels", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		backup.Labels = map[string]string{
			"example.com/key1": "value1",
			"example.com/key2": "value2",
		}
		blueprintList := &blueprintv3.BlueprintList{Items: make([]blueprintv3.Blueprint, 0)}
		var updateCalled = false
		fakeClient := newFakeClientForEnsureBackupSetupTest(t, backup, blueprintList, &updateCalled)
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default", "velero")

		nextAction, err := reconciler.ensureBackupMetadata(context.Background(), backup)

		expectedLabels := map[string]string{}
		maps.Copy(expectedLabels, backup.Labels)
		maps.Copy(expectedLabels, defaultLabels)

		assert.NoError(t, err)
		assert.Equal(t, Next, nextAction)
		assert.True(t, reflect.DeepEqual(backup.Labels, expectedLabels))
		assert.True(t, updateCalled)
	})

	t.Run("It should add the finalizer", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		blueprintList := &blueprintv3.BlueprintList{Items: make([]blueprintv3.Blueprint, 0)}
		var updateCalled = false
		fakeClient := newFakeClientForEnsureBackupSetupTest(t, backup, blueprintList, &updateCalled)
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default", "velero")

		nextAction, err := reconciler.ensureBackupMetadata(context.Background(), backup)

		assert.NoError(t, err)
		assert.Equal(t, Next, nextAction)

		expectedFinalizers := []string{backupv1.BackupFinalizer}
		assert.ElementsMatch(t, backup.Finalizers, expectedFinalizers)
		assert.True(t, updateCalled)
	})

	t.Run("It should keep existing finalizers while adding the finalizer", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		backup.Finalizers = []string{"finalizer01", "finalizer02"}
		blueprintList := &blueprintv3.BlueprintList{
			Items: []blueprintv3.Blueprint{},
		}
		var updateCalled = false
		fakeClient := newFakeClientForEnsureBackupSetupTest(t, backup, blueprintList, &updateCalled)
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default", "velero")

		nextAction, err := reconciler.ensureBackupMetadata(context.Background(), backup)

		assert.NoError(t, err)
		assert.Equal(t, Next, nextAction)

		expectedFinalizers := []string{backupv1.BackupFinalizer, "finalizer01", "finalizer02"}
		assert.ElementsMatch(t, backup.Finalizers, expectedFinalizers)
		assert.True(t, updateCalled)
	})

	t.Run("should not update unchanged metadata", func(t *testing.T) {
		blueprintList := newBlueprintListForEnsureBackupSetupTest(
			"ns",
			"blueprint",
			"blueprint-display-name",
			[]blueprintv3.Dogu{{Name: "dogu01"}, {Name: "dogu02"}},
		)
		backup := newBackupForTest("ns", "backup")
		backup.Labels = maps.Clone(defaultLabels)
		backup.Finalizers = []string{backupv1.BackupFinalizer}
		updateCalled := false
		fakeClient := newFakeClientForEnsureBackupSetupTest(t, backup, blueprintList, &updateCalled)
		reconciler := NewReconciler(fakeClient, newTestEventRecorder(), nil, newRealClock(), "default", "velero")

		nextAction, err := reconciler.ensureBackupMetadata(context.Background(), backup)

		assert.NoError(t, err)
		assert.Equal(t, Next, nextAction)
		assert.False(t, updateCalled)
	})
}

func newFakeClientForEnsureBackupSetupTest(t *testing.T, backup *backupv1.Backup, blueprintList *blueprintv3.BlueprintList, updateCalled *bool) client.WithWatch {
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

func newBlueprintListForEnsureBackupSetupTest(
	namespace string,
	name string,
	displayName string,
	dogus []blueprintv3.Dogu,
) *blueprintv3.BlueprintList {
	blueprint := blueprintv3.Blueprint{
		Namespace: namespace,
		Name:      name,
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
