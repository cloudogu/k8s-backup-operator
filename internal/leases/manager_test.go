package leases

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	coordinationv1 "k8s.io/api/coordination/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	testNamespace = "ecosystem"
	testLeaseName = "backup-restore-operation"
	testKind      = "TestOperation"
)

func TestManagerAcquire(t *testing.T) {
	ctx := context.Background()

	t.Run("creates and then recognizes its own lease", func(t *testing.T) {
		holder := testHolder("restore-a", "restore-a-uid")
		k8sClient := newLeaseTestClient(t, holder)
		manager := NewManager(k8sClient, testNamespace, testLeaseName, configMapResolver{client: k8sClient})

		result, err := manager.Acquire(ctx, holder, testKind)
		require.NoError(t, err)
		assert.Equal(t, StateAcquired, result.State)
		assert.Equal(t, holder.Name, result.HolderName)

		result, err = manager.Acquire(ctx, holder, testKind)
		require.NoError(t, err)
		assert.Equal(t, StateHeld, result.State)
		assert.Equal(t, holder.Name, result.HolderName)
	})

	t.Run("reports another active typed holder", func(t *testing.T) {
		holder := testHolder("restore-a", "restore-a-uid")
		contender := testHolder("backup-a", "backup-a-uid")
		lease := NewLease(testNamespace, testLeaseName, holder, testKind)
		k8sClient := newLeaseTestClient(t, holder, contender, lease)
		manager := NewManager(k8sClient, testNamespace, testLeaseName, configMapResolver{client: k8sClient})

		result, err := manager.Acquire(ctx, contender, testKind)
		require.NoError(t, err)
		assert.Equal(t, StateWaiting, result.State)
		assert.Equal(t, holder.Name, result.HolderName)
	})

	t.Run("waits for a typed holder from an unregistered workflow", func(t *testing.T) {
		holder := testHolder("backup-a", "backup-a-uid")
		contender := testHolder("restore-a", "restore-a-uid")
		lease := NewLease(testNamespace, testLeaseName, holder, "Backup")
		k8sClient := newLeaseTestClient(t, holder, contender, lease)
		manager := NewManager(k8sClient, testNamespace, testLeaseName, configMapResolver{client: k8sClient})

		result, err := manager.Acquire(ctx, contender, testKind)
		require.NoError(t, err)
		assert.Equal(t, StateWaiting, result.State)
		assert.Equal(t, holder.Name, result.HolderName)
	})

	for _, test := range []struct {
		name   string
		mutate func(*coordinationv1.Lease)
	}{
		{name: "missing holder UID", mutate: func(lease *coordinationv1.Lease) { lease.Spec.HolderIdentity = nil }},
		{name: "missing holder name", mutate: func(lease *coordinationv1.Lease) { delete(lease.Annotations, HolderNameAnnotation) }},
		{name: "missing holder kind", mutate: func(lease *coordinationv1.Lease) { delete(lease.Annotations, HolderKindAnnotation) }},
	} {
		t.Run("reports an invalid lease with "+test.name, func(t *testing.T) {
			holder := testHolder("restore-a", "restore-a-uid")
			contender := testHolder("backup-a", "backup-a-uid")
			lease := NewLease(testNamespace, testLeaseName, holder, testKind)
			test.mutate(lease)
			k8sClient := newLeaseTestClient(t, holder, contender, lease)
			manager := NewManager(k8sClient, testNamespace, testLeaseName, configMapResolver{client: k8sClient})

			result, err := manager.Acquire(ctx, contender, testKind)

			require.NoError(t, err)
			assert.Equal(t, StateInvalid, result.State)
		})
	}

	t.Run("takes over a lease whose holder disappeared", func(t *testing.T) {
		oldHolder := testHolder("restore-a", "restore-a-uid")
		contender := testHolder("backup-a", "backup-a-uid")
		lease := NewLease(testNamespace, testLeaseName, oldHolder, testKind)
		k8sClient := newLeaseTestClient(t, contender, lease)
		manager := NewManager(k8sClient, testNamespace, testLeaseName, configMapResolver{client: k8sClient})

		result, err := manager.Acquire(ctx, contender, testKind)
		require.NoError(t, err)
		assert.Equal(t, StateAcquired, result.State)
		assert.Equal(t, contender.Name, result.HolderName)

		stored := &coordinationv1.Lease{}
		require.NoError(t, k8sClient.Get(ctx, client.ObjectKeyFromObject(lease), stored))
		assert.True(t, IsHolder(stored, contender, testKind))
	})
}

func TestManagerRelease(t *testing.T) {
	ctx := context.Background()
	holder := testHolder("restore-a", "restore-a-uid")
	contender := testHolder("backup-a", "backup-a-uid")
	lease := NewLease(testNamespace, testLeaseName, holder, testKind)
	k8sClient := newLeaseTestClient(t, holder, contender, lease)
	manager := NewManager(k8sClient, testNamespace, testLeaseName, configMapResolver{client: k8sClient})

	released, err := manager.Release(ctx, contender, testKind)
	require.NoError(t, err)
	assert.False(t, released)

	released, err = manager.Release(ctx, holder, testKind)
	require.NoError(t, err)
	assert.True(t, released)

	stored := &coordinationv1.Lease{}
	err = k8sClient.Get(ctx, client.ObjectKeyFromObject(lease), stored)
	assert.True(t, client.IgnoreNotFound(err) == nil)
	assert.Error(t, err)
}

func TestManagerHolds(t *testing.T) {
	ctx := context.Background()
	holder := testHolder("restore-a", "restore-a-uid")
	contender := testHolder("backup-a", "backup-a-uid")

	t.Run("reports false without ever creating or taking a lease", func(t *testing.T) {
		k8sClient := newLeaseTestClient(t, holder)
		manager := NewManager(k8sClient, testNamespace, testLeaseName, configMapResolver{client: k8sClient})

		holds, err := manager.Holds(ctx, holder, testKind)
		require.NoError(t, err)
		assert.False(t, holds)

		stored := &coordinationv1.Lease{}
		err = k8sClient.Get(ctx, client.ObjectKey{Namespace: testNamespace, Name: testLeaseName}, stored)
		assert.Error(t, err, "Holds must not create the lease")
	})

	t.Run("distinguishes the holder from a contender and does not take over", func(t *testing.T) {
		lease := NewLease(testNamespace, testLeaseName, holder, testKind)
		k8sClient := newLeaseTestClient(t, holder, contender, lease)
		manager := NewManager(k8sClient, testNamespace, testLeaseName, configMapResolver{client: k8sClient})

		holds, err := manager.Holds(ctx, holder, testKind)
		require.NoError(t, err)
		assert.True(t, holds)

		holds, err = manager.Holds(ctx, contender, testKind)
		require.NoError(t, err)
		assert.False(t, holds)

		// A different kind with the same name and UID is not the holder either.
		holds, err = manager.Holds(ctx, holder, "OtherKind")
		require.NoError(t, err)
		assert.False(t, holds)

		stored := &coordinationv1.Lease{}
		require.NoError(t, k8sClient.Get(ctx, client.ObjectKeyFromObject(lease), stored))
		assert.True(t, IsHolder(stored, holder, testKind), "Holds must leave the lease with its owner")
	})

	t.Run("a stale lease of a terminal holder is still not held by a contender", func(t *testing.T) {
		terminalHolder := testHolder("restore-a", "restore-a-uid")
		terminalHolder.Annotations = map[string]string{"terminal": "true"}
		lease := NewLease(testNamespace, testLeaseName, terminalHolder, testKind)
		k8sClient := newLeaseTestClient(t, terminalHolder, contender, lease)
		manager := NewManager(k8sClient, testNamespace, testLeaseName, configMapResolver{client: k8sClient})

		holds, err := manager.Holds(ctx, contender, testKind)
		require.NoError(t, err)
		assert.False(t, holds)
	})
}

type configMapResolver struct {
	client client.Client
}

func (configMapResolver) Kind() string { return testKind }

func (r configMapResolver) Get(ctx context.Context, namespace, name string) (client.Object, error) {
	holder := &corev1.ConfigMap{}
	err := r.client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, holder)
	return holder, err
}

func (configMapResolver) IsTerminal(holder client.Object) bool {
	return holder.GetAnnotations()["terminal"] == "true"
}

func testHolder(name string, uid types.UID) *corev1.ConfigMap {
	return &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: testNamespace, UID: uid}}
}

func newLeaseTestClient(t *testing.T, objects ...client.Object) client.Client {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, coordinationv1.AddToScheme(scheme))
	return fake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).Build()
}
