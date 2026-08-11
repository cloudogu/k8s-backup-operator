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
	apiMeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestEnsureBackupSetup(t *testing.T) {
	t.Run("It should add default labels", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		blueprintList := &blueprintv3.BlueprintList{Items: make([]blueprintv3.Blueprint, 0)}
		var updateCalled = false
		fakeClient := newFakeClientForEnsureBackupSetupTest(t, backup, blueprintList, &updateCalled)
		reconciler := NewReconciler(fakeClient, nil, newRealClock(), "default")

		nextAction, err := reconciler.ensureBackupSetup(context.Background(), backup, logr.Discard())

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
		reconciler := NewReconciler(fakeClient, nil, newRealClock(), "default")

		nextAction, err := reconciler.ensureBackupSetup(context.Background(), backup, logr.Discard())

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
		reconciler := NewReconciler(fakeClient, nil, newRealClock(), "default")

		nextAction, err := reconciler.ensureBackupSetup(context.Background(), backup, logr.Discard())

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
		reconciler := NewReconciler(fakeClient, nil, newRealClock(), "default")

		nextAction, err := reconciler.ensureBackupSetup(context.Background(), backup, logr.Discard())

		assert.NoError(t, err)
		assert.Equal(t, Next, nextAction)

		expectedFinalizers := []string{backupv1.BackupFinalizer, "finalizer01", "finalizer02"}
		assert.ElementsMatch(t, backup.Finalizers, expectedFinalizers)
		assert.True(t, updateCalled)
	})

	t.Run("should add an annotation with the blueprint's display name and dogus", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		blueprintList := newBlueprintListForEnsureBackupSetupTest(
			"ns",
			"blueprint",
			"blueprint-display-name",
			[]blueprintv3.Dogu{
				{Name: "dogu01"},
				{Name: "dogu02"},
			},
		)
		var updateCalled = false
		fakeClient := newFakeClientForEnsureBackupSetupTest(t, backup, blueprintList, &updateCalled)
		reconciler := NewReconciler(fakeClient, nil, newRealClock(), "default")

		nextAction, err := reconciler.ensureBackupSetup(context.Background(), backup, logr.Discard())

		assert.NoError(t, err)
		assert.Equal(t, Next, nextAction)
		assert.Equal(t, "blueprint-display-name", backup.Annotations[blueprintIdAnnotation])
		assert.JSONEq(t, `[{"name": "dogu01"}, {"name": "dogu02"}]`, backup.Annotations[blueprintDogusAnnotation])
		assert.True(t, updateCalled)
	})

	t.Run("should keep existing annotations while adding the annotations for the blueprint infos", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		backup.Annotations = map[string]string{
			"example.com/anno1": "annoVal1",
			"example.com/anno2": "annoVal2",
		}
		blueprintList := newBlueprintListForEnsureBackupSetupTest(
			"ns",
			"blueprint",
			"blueprint-display-name",
			[]blueprintv3.Dogu{
				{Name: "dogu01"},
				{Name: "dogu02"},
			},
		)
		var updateCalled = false
		fakeClient := newFakeClientForEnsureBackupSetupTest(t, backup, blueprintList, &updateCalled)
		reconciler := NewReconciler(fakeClient, nil, newRealClock(), "default")

		nextAction, err := reconciler.ensureBackupSetup(context.Background(), backup, logr.Discard())

		assert.NoError(t, err)
		assert.Equal(t, Next, nextAction)
		assert.ElementsMatch(t,
			[]string{"example.com/anno1", "example.com/anno2", blueprintIdAnnotation, blueprintDogusAnnotation},
			slices.Collect(maps.Keys(backup.Annotations)),
		)
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
		backup.Annotations = map[string]string{
			blueprintIdAnnotation:    "blueprint-display-name",
			blueprintDogusAnnotation: `[{"name":"dogu01"},{"name":"dogu02"}]`,
		}
		backup.Finalizers = []string{backupv1.BackupFinalizer}
		updateCalled := false
		fakeClient := newFakeClientForEnsureBackupSetupTest(t, backup, blueprintList, &updateCalled)
		reconciler := NewReconciler(fakeClient, nil, newRealClock(), "default")

		nextAction, err := reconciler.ensureBackupSetup(context.Background(), backup, logr.Discard())

		assert.NoError(t, err)
		assert.Equal(t, Next, nextAction)
		assert.False(t, updateCalled)
	})

	t.Run("should update changed managed metadata", func(t *testing.T) {
		blueprintList := newBlueprintListForEnsureBackupSetupTest(
			"ns",
			"blueprint",
			"blueprint-display-name",
			[]blueprintv3.Dogu{{Name: "dogu01"}},
		)
		backup := newBackupForTest("ns", "backup")
		backup.Labels = maps.Clone(defaultLabels)
		backup.Labels["app"] = "outdated"
		backup.Annotations = map[string]string{
			blueprintIdAnnotation:    "outdated",
			blueprintDogusAnnotation: `[{"name":"outdated"}]`,
		}
		backup.Finalizers = []string{backupv1.BackupFinalizer}
		updateCalled := false
		fakeClient := newFakeClientForEnsureBackupSetupTest(t, backup, blueprintList, &updateCalled)
		reconciler := NewReconciler(fakeClient, nil, newRealClock(), "default")

		nextAction, err := reconciler.ensureBackupSetup(context.Background(), backup, logr.Discard())

		assert.NoError(t, err)
		assert.Equal(t, Next, nextAction)
		assert.True(t, updateCalled)
		assert.Equal(t, defaultLabels["app"], backup.Labels["app"])
		assert.Equal(t, "blueprint-display-name", backup.Annotations[blueprintIdAnnotation])
		assert.JSONEq(t, `[{"name":"dogu01"}]`, backup.Annotations[blueprintDogusAnnotation])
	})

	t.Run("should ignore a missing blueprint CRD", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		updateCalled := false
		fakeClient := newFakeClientBuilder(t).
			WithObjects(backup).
			WithInterceptorFuncs(interceptor.Funcs{
				List: func(context.Context, client.WithWatch, client.ObjectList, ...client.ListOption) error {
					return &apiMeta.NoKindMatchError{
						GroupKind:        schema.GroupKind{Group: "k8s.cloudogu.com", Kind: "Blueprint"},
						SearchedVersions: []string{"v3"},
					}
				},
				Update: func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
					updateCalled = true
					return client.Update(ctx, obj, opts...)
				},
			}).
			Build()
		reconciler := NewReconciler(fakeClient, nil, newRealClock(), "default")

		nextAction, err := reconciler.ensureBackupSetup(context.Background(), backup, logr.Discard())

		assert.NoError(t, err)
		assert.Equal(t, Next, nextAction)
		assert.True(t, updateCalled)
		assert.Contains(t, backup.Finalizers, backupv1.BackupFinalizer)
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
