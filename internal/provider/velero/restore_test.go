package velero

import (
	"context"
	"strings"
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

func getPersistedRestore(t *testing.T, k8sClient client.WithWatch, name string) *velerov1.Restore {
	t.Helper()

	persisted := &velerov1.Restore{}
	require.NoError(t, k8sClient.Get(testCtx, types.NamespacedName{Namespace: testNamespace, Name: name}, persisted))

	return persisted
}

// testRequest is the request of a unit restore, for tests that do not depend on the kind of request.
func testRequest() RestoreRequest {
	return RestoreRequest{Name: "test-restore-global-config", BackupName: "test-backup-global-config"}
}

// legacyRequest is the request of a legacy restore: named like the parent, whole backup, no extra labels.
func legacyRequest() RestoreRequest {
	return RestoreRequest{Name: testRestore, BackupName: testBackup}
}

func TestBuildRestore(t *testing.T) {
	// a components unit restore with a label selector for an excluded component
	selector := &metav1.LabelSelector{MatchExpressions: []metav1.LabelSelectorRequirement{
		{Key: "k8s.cloudogu.com/component.name", Operator: metav1.LabelSelectorOpNotIn, Values: []string{"k8s-loki"}},
	}}
	tests := []struct {
		name       string
		request    RestoreRequest
		wantLabels map[string]string
	}{
		{
			name: "a unit request adds its labels and selector",
			request: RestoreRequest{
				Name:          "test-restore-components",
				BackupName:    "test-backup-components",
				Labels:        map[string]string{"k8s.cloudogu.com/backup": testBackup, "k8s.cloudogu.com/backup-type": "components"},
				LabelSelector: selector,
			},
			wantLabels: map[string]string{
				RestoreSourceNameLabel:         testRestore,
				RestoreSourceUIDLabel:          string(testRestoreUID),
				"k8s.cloudogu.com/backup":      testBackup,
				"k8s.cloudogu.com/backup-type": "components",
			},
		},
		{
			name:    "a legacy request carries only the source labels and restores the whole backup",
			request: legacyRequest(),
			wantLabels: map[string]string{
				RestoreSourceNameLabel: testRestore,
				RestoreSourceUIDLabel:  string(testRestoreUID),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			veleroRestore := BuildRestore(newCesRestore(), tt.request)

			assert.Equal(t, tt.request.Name, veleroRestore.Name)
			assert.Equal(t, testNamespace, veleroRestore.Namespace)
			assert.Equal(t, tt.wantLabels, veleroRestore.Labels)

			controller := metav1.GetControllerOf(veleroRestore)
			require.NotNil(t, controller)
			assert.Equal(t, "k8s.cloudogu.com/v1", controller.APIVersion)
			assert.Equal(t, restoreKind, controller.Kind)
			assert.Equal(t, testRestore, controller.Name)
			assert.Equal(t, testRestoreUID, controller.UID)
			require.NotNil(t, controller.BlockOwnerDeletion)
			assert.True(t, *controller.BlockOwnerDeletion)
			assert.Equal(t, tt.request.BackupName, veleroRestore.Spec.BackupName)
			assert.Equal(t, tt.request.LabelSelector, veleroRestore.Spec.LabelSelector)
			assert.Equal(t, velerov1.PolicyTypeUpdate, veleroRestore.Spec.ExistingResourcePolicy)
			require.NotNil(t, veleroRestore.Spec.ResourceModifier)
			require.NotNil(t, veleroRestore.Spec.ResourceModifier.APIGroup)
			assert.Empty(t, *veleroRestore.Spec.ResourceModifier.APIGroup)
			assert.Equal(t, "ConfigMap", veleroRestore.Spec.ResourceModifier.Kind)
			assert.Equal(t, restoreDoguModifierConfigMapName, veleroRestore.Spec.ResourceModifier.Name)
		})
	}
}

func TestBuildRestoreTruncatesTheSourceNameLabel(t *testing.T) {
	cesRestore := newCesRestore()
	cesRestore.Name = strings.Repeat("r", 70)

	veleroRestore := BuildRestore(cesRestore, testRequest())

	assert.Equal(t, k8sv1.GetTruncatedName(cesRestore.Name), veleroRestore.Labels[RestoreSourceNameLabel])
	assert.Len(t, veleroRestore.Labels[RestoreSourceNameLabel], 63)
}

func TestEnsureRestoreCreatesTheVeleroRestoreAtTheRequestedName(t *testing.T) {
	cesRestore := newCesRestore()
	writes := &writeCounter{}
	k8sClient := newTestClient(t, writes, cesRestore)
	request := testRequest()

	created, err := EnsureRestore(testCtx, k8sClient, cesRestore, request)

	require.NoError(t, err)
	require.NotNil(t, created)
	assert.Equal(t, 1, writes.creates)
	assert.Equal(t, 1, writes.total())

	persisted := getPersistedRestore(t, k8sClient, request.Name)
	assert.True(t, IsOwnedRestore(cesRestore, persisted))
	assert.Equal(t, request.BackupName, persisted.Spec.BackupName)
	assert.Equal(t, string(testRestoreUID), persisted.Labels[RestoreSourceUIDLabel])
}

func TestEnsureRestoreReusesItsOwnChildWithoutWriting(t *testing.T) {
	cesRestore := newCesRestore()
	writes := &writeCounter{}
	k8sClient := newTestClient(t, writes, cesRestore)

	first, err := EnsureRestore(testCtx, k8sClient, cesRestore, testRequest())
	require.NoError(t, err)
	firstResourceVersion := getPersistedRestore(t, k8sClient, testRequest().Name).ResourceVersion

	second, err := EnsureRestore(testCtx, k8sClient, cesRestore, testRequest())

	require.NoError(t, err)
	require.NotNil(t, second)
	assert.Equal(t, first.Name, second.Name)
	assert.Equal(t, 1, writes.creates, "the velero Restore must be created exactly once across both attempts")
	assert.Equal(t, 1, writes.total(), "the second attempt must perform no write at all")
	assert.Equal(t, firstResourceVersion, getPersistedRestore(t, k8sClient, testRequest().Name).ResourceVersion)
}

func TestCheckRestoreForConflictsComparesTheExpectedBackupName(t *testing.T) {
	cesRestore := newCesRestore()
	veleroRestore := BuildRestore(cesRestore, testRequest())

	require.NoError(t, CheckRestoreForConflicts(cesRestore, veleroRestore, testRequest().BackupName))

	err := CheckRestoreForConflicts(cesRestore, veleroRestore, testBackup)
	var conflictErr *ConflictError
	require.ErrorAs(t, err, &conflictErr)
	assert.EqualError(t, err, "velero restore [test-restore-global-config] conflicts with the expected child: it restores backup [test-backup-global-config] but restore [test-restore] expects backup [test-backup]")
}

func TestEnsureRestoreReportsConflicts(t *testing.T) {
	tests := map[string]struct {
		request  RestoreRequest
		existing *velerov1.Restore
		reason   string
	}{
		"a velero restore written by an older operator version carries no owner reference": {
			request: legacyRequest(),
			existing: &velerov1.Restore{
				Name: testRestore, Namespace: testNamespace,
				Spec: velerov1.RestoreSpec{BackupName: testBackup},
			},
			reason: "must not be adopted",
		},
		"a leftover velero restore of a deleted Cloudogu Restore of the same name carries a foreign UID": {
			request: testRequest(),
			existing: func() *velerov1.Restore {
				foreignCesRestore := newCesRestore()
				foreignCesRestore.UID = types.UID("22222222-2222-2222-2222-222222222222")

				return BuildRestore(foreignCesRestore, testRequest())
			}(),
			reason: "must not be adopted",
		},
		"our own velero restore restores a different backup than the cesRestore now expects": {
			request: testRequest(),
			existing: func() *velerov1.Restore {
				veleroRestore := BuildRestore(newCesRestore(), testRequest())
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
			untouched := getPersistedRestore(t, k8sClient, test.request.Name)

			veleroRestore, err := EnsureRestore(testCtx, k8sClient, cesRestore, test.request)

			require.Error(t, err)
			assert.Nil(t, veleroRestore)
			var conflictErr *ConflictError
			require.ErrorAs(t, err, &conflictErr, "the conflict must be classifiable without matching strings")
			assert.Contains(t, err.Error(), test.reason)
			assert.Equal(t, 0, writes.total(), "a conflicting velero restore must neither be deleted, mutated nor claimed")
			assert.Equal(t, untouched, getPersistedRestore(t, k8sClient, test.request.Name))
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

		veleroRestore, err := EnsureRestore(testCtx, k8sClient, newCesRestore(), testRequest())

		require.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to get velero restore [test-restore-global-config]")
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

		veleroRestore, err := EnsureRestore(testCtx, k8sClient, cesRestore, testRequest())

		require.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to create velero restore [test-restore-global-config]")
		assert.Nil(t, veleroRestore)
	})
}

func TestHasOwnedRestoreRequiresTheOwnerReference(t *testing.T) {
	cesRestore := newCesRestore()
	unownedRestore := BuildRestore(cesRestore, testRequest())
	unownedRestore.Name = "a-unowned-with-uid-label"
	unownedRestore.OwnerReferences = nil
	owned := BuildRestore(cesRestore, testRequest())
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

func TestGetRestoreByNameSucceedsEmptyOnAbsentVeleroRestore(t *testing.T) {
	writes := &writeCounter{}
	k8sClient := newTestClient(t, writes)

	veleroRestore, err := GetRestoreByName(testCtx, k8sClient, testNamespace, testRequest().Name)

	require.NoError(t, err)
	assert.Nil(t, veleroRestore)
}

func TestDeleteRestore(t *testing.T) {
	t.Run("deletes the velero restore", func(t *testing.T) {
		cesRestore := newCesRestore()
		veleroRestore := BuildRestore(cesRestore, testRequest())
		veleroRestore.UID = "22222222-2222-2222-2222-222222222222"
		writes := &writeCounter{}
		k8sClient := newTestClient(t, writes, cesRestore, veleroRestore)

		require.NoError(t, DeleteRestore(testCtx, k8sClient, veleroRestore))

		assert.Equal(t, 1, writes.deletes)
		err := k8sClient.Get(testCtx, types.NamespacedName{Namespace: testNamespace, Name: testRequest().Name}, &velerov1.Restore{})
		assert.True(t, apierrors.IsNotFound(err))
	})

	t.Run("tolerates an already absent velero restore", func(t *testing.T) {
		veleroRestore := BuildRestore(newCesRestore(), testRequest())
		veleroRestore.UID = "22222222-2222-2222-2222-222222222222"
		writes := &writeCounter{}
		k8sClient := newTestClient(t, writes)

		require.NoError(t, DeleteRestore(testCtx, k8sClient, veleroRestore))
	})

	t.Run("propagates other errors", func(t *testing.T) {
		writes := &writeCounter{}
		funcs := writes.interceptor()
		funcs.Delete = func(_ context.Context, _ client.WithWatch, _ client.Object, _ ...client.DeleteOption) error {
			return apierrors.NewForbidden(schema.GroupResource{Group: velerov1.SchemeGroupVersion.Group, Resource: "restores"}, testRequest().Name, assert.AnError)
		}
		testScheme := runtime.NewScheme()
		require.NoError(t, velerov1.AddToScheme(testScheme))
		k8sClient := fake.NewClientBuilder().WithScheme(testScheme).WithInterceptorFuncs(funcs).Build()
		veleroRestore := BuildRestore(newCesRestore(), testRequest())
		veleroRestore.UID = "22222222-2222-2222-2222-222222222222"

		err := DeleteRestore(testCtx, k8sClient, veleroRestore)

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to delete velero restore [test-restore-global-config]")
	})
}
