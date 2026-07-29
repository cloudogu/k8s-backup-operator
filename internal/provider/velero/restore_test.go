package velero

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
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

	return fake.NewClientBuilder().
		WithScheme(testScheme).
		WithObjects(objects...).
		WithInterceptorFuncs(writes.interceptor()).
		Build()
}

func newParentRestore() *k8sv1.Restore {
	return &k8sv1.Restore{
		ObjectMeta: metav1.ObjectMeta{Name: testRestore, Namespace: testNamespace, UID: testRestoreUID},
		Spec:       k8sv1.RestoreSpec{BackupName: testBackup, Provider: k8sv1.ProviderVelero},
	}
}

func getPersistedRestore(t *testing.T, k8sClient client.WithWatch) *velerov1.Restore {
	t.Helper()

	persisted := &velerov1.Restore{}
	require.NoError(t, k8sClient.Get(testCtx, types.NamespacedName{Namespace: testNamespace, Name: testRestore}, persisted))

	return persisted
}

func TestBuildRestore(t *testing.T) {
	child := BuildRestore(newParentRestore())

	assert.Equal(t, testRestore, child.Name)
	assert.Equal(t, testNamespace, child.Namespace)
	assert.Equal(t, map[string]string{
		RestoreSourceNameLabel: testRestore,
		RestoreSourceUIDLabel:  string(testRestoreUID),
	}, child.Labels)

	controller := metav1.GetControllerOf(child)
	require.NotNil(t, controller)
	assert.Equal(t, "k8s.cloudogu.com/v1", controller.APIVersion)
	assert.Equal(t, restoreKind, controller.Kind)
	assert.Equal(t, testRestore, controller.Name)
	assert.Equal(t, testRestoreUID, controller.UID)
	require.NotNil(t, controller.BlockOwnerDeletion)
	assert.True(t, *controller.BlockOwnerDeletion)
	assert.Equal(t, testBackup, child.Spec.BackupName)
	assert.Equal(t, velerov1.PolicyTypeUpdate, child.Spec.ExistingResourcePolicy)
	require.NotNil(t, child.Spec.ResourceModifier)
	require.NotNil(t, child.Spec.ResourceModifier.APIGroup)
	assert.Empty(t, *child.Spec.ResourceModifier.APIGroup)
	assert.Equal(t, "ConfigMap", child.Spec.ResourceModifier.Kind)
	assert.Equal(t, restoreDoguModifierConfigMapName, child.Spec.ResourceModifier.Name)
}

func TestEnsureRestoreCreatesTheChild(t *testing.T) {
	parent := newParentRestore()
	writes := &writeCounter{}
	k8sClient := newTestClient(t, writes, parent)

	created, err := EnsureRestore(testCtx, k8sClient, parent)

	require.NoError(t, err)
	require.NotNil(t, created)
	assert.Equal(t, 1, writes.creates)
	assert.Equal(t, 1, writes.total())

	persisted := getPersistedRestore(t, k8sClient)
	assert.True(t, IsOwnedRestore(parent, persisted))
	assert.Equal(t, testBackup, persisted.Spec.BackupName)
	assert.Equal(t, string(testRestoreUID), persisted.Labels[RestoreSourceUIDLabel])
}

func TestEnsureRestoreReusesItsOwnChildWithoutWriting(t *testing.T) {
	parent := newParentRestore()
	writes := &writeCounter{}
	k8sClient := newTestClient(t, writes, parent)

	first, err := EnsureRestore(testCtx, k8sClient, parent)
	require.NoError(t, err)
	firstResourceVersion := getPersistedRestore(t, k8sClient).ResourceVersion

	second, err := EnsureRestore(testCtx, k8sClient, parent)

	require.NoError(t, err)
	require.NotNil(t, second)
	assert.Equal(t, first.Name, second.Name)
	assert.Equal(t, 1, writes.creates, "the child must be created exactly once across both attempts")
	assert.Equal(t, 1, writes.total(), "the second attempt must perform no write at all")
	assert.Equal(t, firstResourceVersion, getPersistedRestore(t, k8sClient).ResourceVersion)
}

func TestEnsureRestoreReportsConflicts(t *testing.T) {
	tests := map[string]struct {
		existing *velerov1.Restore
		reason   string
	}{
		"a child written by an older operator version carries no owner reference": {
			existing: &velerov1.Restore{
				ObjectMeta: metav1.ObjectMeta{Name: testRestore, Namespace: testNamespace},
				Spec:       velerov1.RestoreSpec{BackupName: testBackup},
			},
			reason: "must not be adopted",
		},
		"a leftover child of a deleted Cloudogu Restore of the same name carries a foreign UID": {
			existing: func() *velerov1.Restore {
				foreignParent := newParentRestore()
				foreignParent.UID = types.UID("22222222-2222-2222-2222-222222222222")

				return BuildRestore(foreignParent)
			}(),
			reason: "must not be adopted",
		},
		"our own child restores a different backup than the parent now expects": {
			existing: func() *velerov1.Restore {
				child := BuildRestore(newParentRestore())
				child.Spec.BackupName = "some-other-backup"

				return child
			}(),
			reason: "expects backup",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			parent := newParentRestore()
			writes := &writeCounter{}
			k8sClient := newTestClient(t, writes, parent, test.existing)
			untouched := getPersistedRestore(t, k8sClient)

			child, err := EnsureRestore(testCtx, k8sClient, parent)

			require.Error(t, err)
			assert.Nil(t, child)
			var conflictErr *ConflictError
			require.ErrorAs(t, err, &conflictErr, "the conflict must be classifiable without matching strings")
			assert.Contains(t, err.Error(), test.reason)
			assert.Equal(t, 0, writes.total(), "a conflicting child must neither be deleted, mutated nor claimed")
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

		child, err := EnsureRestore(testCtx, k8sClient, newParentRestore())

		require.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to get velero restore [test-restore]")
		assert.Nil(t, child)
		assert.Equal(t, 0, writes.total())
	})

	t.Run("create fails", func(t *testing.T) {
		parent := newParentRestore()
		writes := &writeCounter{}
		funcs := writes.interceptor()
		funcs.Create = func(_ context.Context, _ client.WithWatch, _ client.Object, _ ...client.CreateOption) error {
			return assert.AnError
		}
		testScheme := runtime.NewScheme()
		require.NoError(t, velerov1.AddToScheme(testScheme))
		k8sClient := fake.NewClientBuilder().WithScheme(testScheme).WithInterceptorFuncs(funcs).Build()

		child, err := EnsureRestore(testCtx, k8sClient, parent)

		require.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to create velero restore [test-restore]")
		assert.Nil(t, child)
	})
}

func TestGetRestoreTreatsAnAbsentChildAsNoChild(t *testing.T) {
	writes := &writeCounter{}
	k8sClient := newTestClient(t, writes)

	child, err := GetRestore(testCtx, k8sClient, newParentRestore())

	require.NoError(t, err)
	assert.Nil(t, child)
}

func TestDeleteRestore(t *testing.T) {
	t.Run("deletes the child", func(t *testing.T) {
		parent := newParentRestore()
		writes := &writeCounter{}
		k8sClient := newTestClient(t, writes, parent, BuildRestore(parent))

		require.NoError(t, DeleteRestore(testCtx, k8sClient, parent))

		assert.Equal(t, 1, writes.deletes)
		err := k8sClient.Get(testCtx, types.NamespacedName{Namespace: testNamespace, Name: testRestore}, &velerov1.Restore{})
		assert.True(t, apierrors.IsNotFound(err))
	})

	t.Run("tolerates an already absent child", func(t *testing.T) {
		writes := &writeCounter{}
		k8sClient := newTestClient(t, writes)

		require.NoError(t, DeleteRestore(testCtx, k8sClient, newParentRestore()))
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

		err := DeleteRestore(testCtx, k8sClient, newParentRestore())

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to delete velero restore [test-restore]")
	})
}

func TestAMatchingRunningChildIsFoundAndReported(t *testing.T) {
	parent := newParentRestore()
	require.Empty(t, parent.Status.Conditions, "the scenario starts from a parent whose conditions were never persisted")
	running := BuildRestore(parent)
	running.Status.Phase = velerov1.RestorePhaseInProgress
	writes := &writeCounter{}
	k8sClient := newTestClient(t, writes, parent, running)

	found, err := GetRestore(testCtx, k8sClient, parent)

	require.NoError(t, err)
	require.NotNil(t, found)
	assert.True(t, IsOwnedRestore(parent, found))
	require.NoError(t, CheckRestoreForConflicts(parent, found))
	assert.Equal(t, RestoreRunning, ObserveRestorePhase(found.Status.Phase))

	// the running child is reused, never restarted
	ensured, err := EnsureRestore(testCtx, k8sClient, parent)

	require.NoError(t, err)
	require.NotNil(t, ensured)
	assert.Equal(t, velerov1.RestorePhaseInProgress, ensured.Status.Phase)
	assert.Equal(t, 0, writes.total(), "a running own child must be reported, not written to")
}
