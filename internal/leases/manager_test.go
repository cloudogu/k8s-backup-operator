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
		assert.Equal(t, StateChanged, result.State)

		result, err = manager.Acquire(ctx, holder, testKind)
		require.NoError(t, err)
		assert.Equal(t, StateAcquired, result.State)
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

	t.Run("takes over a lease whose holder disappeared", func(t *testing.T) {
		oldHolder := testHolder("restore-a", "restore-a-uid")
		contender := testHolder("backup-a", "backup-a-uid")
		lease := NewLease(testNamespace, testLeaseName, oldHolder, testKind)
		k8sClient := newLeaseTestClient(t, contender, lease)
		manager := NewManager(k8sClient, testNamespace, testLeaseName, configMapResolver{client: k8sClient})

		result, err := manager.Acquire(ctx, contender, testKind)
		require.NoError(t, err)
		assert.Equal(t, StateChanged, result.State)

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

type configMapResolver struct {
	client client.Client
}

func (configMapResolver) Kind() string { return testKind }

func (r configMapResolver) Get(ctx context.Context, namespace, name string) (client.Object, error) {
	holder := &corev1.ConfigMap{}
	err := r.client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, holder)
	return holder, err
}

func (r configMapResolver) FindByUID(ctx context.Context, namespace string, uid types.UID) (client.Object, error) {
	holders := &corev1.ConfigMapList{}
	if err := r.client.List(ctx, holders, client.InNamespace(namespace)); err != nil {
		return nil, err
	}
	for i := range holders.Items {
		if holders.Items[i].UID == uid {
			return &holders.Items[i], nil
		}
	}
	return nil, nil
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
