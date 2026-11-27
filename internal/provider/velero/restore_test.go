package velero

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	k8sv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
)

var testCtx = context.TODO()

const (
	testNamespace  = "ecosystem-test"
	testRestore    = "test-restore"
	testRestoreUID = types.UID("11111111-1111-1111-1111-111111111111")
	testBackup     = "test-backup"
)

// writeCounter counts every mutating call a test makes through the fake client
type writeCounter struct {
	creates int
	updates int
	patches int
	deletes int
}

func (c *writeCounter) total() int {
	return c.creates + c.updates + c.patches + c.deletes
}

func (c *writeCounter) interceptor() interceptor.Funcs {
	return interceptor.Funcs{
		Create: func(ctx context.Context, wrapped client.WithWatch, object client.Object, opts ...client.CreateOption) error {
			c.creates++

			return wrapped.Create(ctx, object, opts...)
		},
		Update: func(ctx context.Context, wrapped client.WithWatch, object client.Object, opts ...client.UpdateOption) error {
			c.updates++

			return wrapped.Update(ctx, object, opts...)
		},
		Patch: func(ctx context.Context, wrapped client.WithWatch, object client.Object, patch client.Patch, opts ...client.PatchOption) error {
			c.patches++

			return wrapped.Patch(ctx, object, patch, opts...)
		},
		Delete: func(ctx context.Context, wrapped client.WithWatch, object client.Object, opts ...client.DeleteOption) error {
			c.deletes++

			return wrapped.Delete(ctx, object, opts...)
		},
	}
}

func newTestClient(t *testing.T, writes *writeCounter, objects ...client.Object) client.WithWatch {
	t.Helper()

	testScheme := runtime.NewScheme()
	require.NoError(t, k8sv1.AddToScheme(testScheme))
	require.NoError(t, velerov1.AddToScheme(testScheme))
	require.NoError(t, appsv1.AddToScheme(testScheme))

	return fake.NewClientBuilder().
		WithScheme(testScheme).
		WithObjects(objects...).
		WithInterceptorFuncs(writes.interceptor()).
		Build()
}

func newCesRestore() *k8sv1.Restore {
	return &k8sv1.Restore{
		Name: testRestore, Namespace: testNamespace, UID: testRestoreUID,
		Spec: k8sv1.RestoreSpec{BackupName: testBackup, Provider: k8sv1.ProviderVelero},
	}
}

func getPersistedRestore(t *testing.T, k8sClient client.WithWatch) *velerov1.Restore {
	t.Helper()

	persisted := &velerov1.Restore{}
	require.NoError(t, k8sClient.Get(testCtx, types.NamespacedName{Namespace: testNamespace, Name: testRestore}, persisted))

	return persisted
}

func TestBuildRestore(t *testing.T) {
	veleroRestore := BuildRestore(newCesRestore())

	assert.Equal(t, testRestore, veleroRestore.Name)
	assert.Equal(t, testNamespace, veleroRestore.Namespace)
	assert.Equal(t, map[string]string{
		RestoreSourceNameLabel: testRestore,
		RestoreSourceUIDLabel:  string(testRestoreUID),
	}, veleroRestore.Labels)

	controller := metav1.GetControllerOf(veleroRestore)
	require.NotNil(t, controller)
	assert.Equal(t, "k8s.cloudogu.com/v1", controller.APIVersion)
	assert.Equal(t, restoreKind, controller.Kind)
	assert.Equal(t, testRestore, controller.Name)
	assert.Equal(t, testRestoreUID, controller.UID)
	require.NotNil(t, controller.BlockOwnerDeletion)
	assert.True(t, *controller.BlockOwnerDeletion)
	assert.Equal(t, testBackup, veleroRestore.Spec.BackupName)
	assert.Equal(t, velerov1.PolicyTypeUpdate, veleroRestore.Spec.ExistingResourcePolicy)
	require.NotNil(t, veleroRestore.Spec.ResourceModifier)
	require.NotNil(t, veleroRestore.Spec.ResourceModifier.APIGroup)
	assert.Empty(t, *veleroRestore.Spec.ResourceModifier.APIGroup)
	assert.Equal(t, "ConfigMap", veleroRestore.Spec.ResourceModifier.Kind)
	assert.Equal(t, restoreDoguModifierConfigMapName, veleroRestore.Spec.ResourceModifier.Name)
}

func TestEnsureRestoreCreatesTheChild(t *testing.T) {
	cesRestore := newCesRestore()
	writes := &writeCounter{}
	k8sClient := newTestClient(t, writes, cesRestore)

	created, err := EnsureRestore(testCtx, k8sClient, cesRestore)

	require.NoError(t, err)
	require.NotNil(t, created)
	assert.Equal(t, 1, writes.creates)
	assert.Equal(t, 1, writes.total())

	persisted := getPersistedRestore(t, k8sClient)
	assert.True(t, IsOwnedRestore(cesRestore, persisted))
	assert.Equal(t, testBackup, persisted.Spec.BackupName)
	assert.Equal(t, string(testRestoreUID), persisted.Labels[RestoreSourceUIDLabel])
}

func TestEnsureRestoreReusesItsOwnChildWithoutWriting(t *testing.T) {
	cesRestore := newCesRestore()
	writes := &writeCounter{}
	k8sClient := newTestClient(t, writes, cesRestore)

	first, err := EnsureRestore(testCtx, k8sClient, cesRestore)
	require.NoError(t, err)
	firstResourceVersion := getPersistedRestore(t, k8sClient).ResourceVersion

	second, err := EnsureRestore(testCtx, k8sClient, cesRestore)

	require.NoError(t, err)
	require.NotNil(t, second)
	assert.Equal(t, first.Name, second.Name)
	assert.Equal(t, 1, writes.creates, "the velero Restore must be created exactly once across both attempts")
	assert.Equal(t, 1, writes.total(), "the second attempt must perform no write at all")
	assert.Equal(t, firstResourceVersion, getPersistedRestore(t, k8sClient).ResourceVersion)
}

func TestCheckRestoreForConflictsComparesTheExpectedBackupName(t *testing.T) {
	cesRestore := newCesRestore()
	veleroRestore := BuildRestore(cesRestore)
	veleroRestore.Spec.BackupName = "test-backup-global-config"

	require.NoError(t, CheckRestoreForConflicts(cesRestore, veleroRestore, "test-backup-global-config"))

	err := CheckRestoreForConflicts(cesRestore, veleroRestore, testBackup)
	var conflictErr *ConflictError
	require.ErrorAs(t, err, &conflictErr)
	assert.EqualError(t, err, "velero restore [test-restore] conflicts with the expected child: it restores backup [test-backup-global-config] but restore [test-restore] expects backup [test-backup]")
}

func TestEnsureRestoreReportsConflicts(t *testing.T) {
	tests := map[string]struct {
		existing *velerov1.Restore
		reason   string
	}{
		"a velero restore written by an older operator version carries no owner reference": {
			existing: &velerov1.Restore{
				Name: testRestore, Namespace: testNamespace,
				Spec: velerov1.RestoreSpec{BackupName: testBackup},
			},
			reason: "must not be adopted",
		},
		"a leftover velero restore of a deleted Cloudogu Restore of the same name carries a foreign UID": {
			existing: func() *velerov1.Restore {
				foreignCesRestore := newCesRestore()
				foreignCesRestore.UID = types.UID("22222222-2222-2222-2222-222222222222")

				return BuildRestore(foreignCesRestore)
			}(),
			reason: "must not be adopted",
		},
		"our own velero restore restores a different backup than the cesRestore now expects": {
			existing: func() *velerov1.Restore {
				veleroRestore := BuildRestore(newCesRestore())
				veleroRestore.Spec.BackupName = "some-other-backup"

				return veleroRestore
			}(),
			reason: "expects backup",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			cesRestore := newCesRestore()
			writes := &writeCounter{}
			k8sClient := newTestClient(t, writes, cesRestore, test.existing)
			untouched := getPersistedRestore(t, k8sClient)

			veleroRestore, err := EnsureRestore(testCtx, k8sClient, cesRestore)

			require.Error(t, err)
			assert.Nil(t, veleroRestore)
			var conflictErr *ConflictError
			require.ErrorAs(t, err, &conflictErr, "the conflict must be classifiable without matching strings")
			assert.Contains(t, err.Error(), test.reason)
			assert.Equal(t, 0, writes.total(), "a conflicting velero restore must neither be deleted, mutated nor claimed")
			assert.Equal(t, untouched, getPersistedRestore(t, k8sClient))
		})
	}
}

func TestEnsureRestorePropagatesClientErrors(t *testing.T) {
	t.Run("get fails", func(t *testing.T) {
		writes := &writeCounter{}
		funcs := writes.interceptor()
		funcs.Get = func(_ context.Context, _ client.WithWatch, _ client.ObjectKey, _ client.Object, _ ...client.GetOption) error {
			return assert.AnError
		}
		testScheme := runtime.NewScheme()
		require.NoError(t, velerov1.AddToScheme(testScheme))
		k8sClient := fake.NewClientBuilder().WithScheme(testScheme).WithInterceptorFuncs(funcs).Build()

		veleroRestore, err := EnsureRestore(testCtx, k8sClient, newCesRestore())

		require.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to get velero restore [test-restore]")
		assert.Nil(t, veleroRestore)
		assert.Equal(t, 0, writes.total())
	})

	t.Run("create fails", func(t *testing.T) {
		cesRestore := newCesRestore()
		writes := &writeCounter{}
		funcs := writes.interceptor()
		funcs.Create = func(_ context.Context, _ client.WithWatch, _ client.Object, _ ...client.CreateOption) error {
			return assert.AnError
		}
		testScheme := runtime.NewScheme()
		require.NoError(t, velerov1.AddToScheme(testScheme))
		k8sClient := fake.NewClientBuilder().WithScheme(testScheme).WithInterceptorFuncs(funcs).Build()

		veleroRestore, err := EnsureRestore(testCtx, k8sClient, cesRestore)

		require.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to create velero restore [test-restore]")
		assert.Nil(t, veleroRestore)
	})
}

func TestHasOwnedRestoreRequiresTheOwnerReference(t *testing.T) {
	cesRestore := newCesRestore()
	unownedRestore := BuildRestore(cesRestore)
	unownedRestore.Name = "a-unowned-with-uid-label"
	unownedRestore.OwnerReferences = nil
	owned := BuildRestore(cesRestore)
	owned.Name = "b-owned"

	tests := []struct {
		name           string
		veleroRestores []client.Object
		want           bool
	}{
		{name: "a UID label without owner reference is not owned", veleroRestores: []client.Object{unownedRestore}, want: false},
		{name: "an owned velero restore behind an unowned one is found", veleroRestores: []client.Object{unownedRestore, owned}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := HasOwnedRestore(testCtx, newTestClient(t, &writeCounter{}, tt.veleroRestores...), cesRestore)

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetLegacyRestoreSucceedsEmptyOnAbsentVeleroRetore(t *testing.T) {
	writes := &writeCounter{}
	k8sClient := newTestClient(t, writes)

	veleroRestore, err := GetLegacyRestore(testCtx, k8sClient, newCesRestore())

	require.NoError(t, err)
	assert.Nil(t, veleroRestore)
}

func TestDeleteRestore(t *testing.T) {
	t.Run("deletes the velero restore", func(t *testing.T) {
		cesRestore := newCesRestore()
		veleroRestore := BuildRestore(cesRestore)
		veleroRestore.UID = "22222222-2222-2222-2222-222222222222"
		writes := &writeCounter{}
		k8sClient := newTestClient(t, writes, cesRestore, veleroRestore)

		require.NoError(t, DeleteRestore(testCtx, k8sClient, veleroRestore))

		assert.Equal(t, 1, writes.deletes)
		err := k8sClient.Get(testCtx, types.NamespacedName{Namespace: testNamespace, Name: testRestore}, &velerov1.Restore{})
		assert.True(t, apierrors.IsNotFound(err))
	})

	t.Run("tolerates an already absent velero restore", func(t *testing.T) {
		veleroRestore := BuildRestore(newCesRestore())
		veleroRestore.UID = "22222222-2222-2222-2222-222222222222"
		writes := &writeCounter{}
		k8sClient := newTestClient(t, writes)

		require.NoError(t, DeleteRestore(testCtx, k8sClient, veleroRestore))
	})

	t.Run("propagates other errors", func(t *testing.T) {
		writes := &writeCounter{}
		funcs := writes.interceptor()
		funcs.Delete = func(_ context.Context, _ client.WithWatch, _ client.Object, _ ...client.DeleteOption) error {
			return apierrors.NewForbidden(schema.GroupResource{Group: velerov1.SchemeGroupVersion.Group, Resource: "restores"}, testRestore, assert.AnError)
		}
		testScheme := runtime.NewScheme()
		require.NoError(t, velerov1.AddToScheme(testScheme))
		k8sClient := fake.NewClientBuilder().WithScheme(testScheme).WithInterceptorFuncs(funcs).Build()
		veleroRestore := BuildRestore(newCesRestore())
		veleroRestore.UID = "22222222-2222-2222-2222-222222222222"

		err := DeleteRestore(testCtx, k8sClient, veleroRestore)

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to delete velero restore [test-restore]")
	})
}

func TestAMatchingRunningVeleroRestoreIsFoundAndReported(t *testing.T) {
	cesRestore := newCesRestore()
	require.Empty(t, cesRestore.Status.Conditions, "the scenario starts from a cesRestore whose conditions were never persisted")
	running := BuildRestore(cesRestore)
	running.Status.Phase = velerov1.RestorePhaseInProgress
	writes := &writeCounter{}
	k8sClient := newTestClient(t, writes, cesRestore, running)

	found, err := GetLegacyRestore(testCtx, k8sClient, cesRestore)

	require.NoError(t, err)
	require.NotNil(t, found)
	assert.True(t, IsOwnedRestore(cesRestore, found))
	require.NoError(t, CheckRestoreForConflicts(cesRestore, found, testBackup))
	assert.Equal(t, RestoreRunning, ObserveRestorePhase(found.Status.Phase))

	// the running velero restore is reused, never restarted
	ensured, err := EnsureRestore(testCtx, k8sClient, cesRestore)

	require.NoError(t, err)
	require.NotNil(t, ensured)
	assert.Equal(t, velerov1.RestorePhaseInProgress, ensured.Status.Phase)
	assert.Equal(t, 0, writes.total(), "a running own velero restore must be reported, not written to")
}
