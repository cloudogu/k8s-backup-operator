package scale

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestRepeatedScaleDownPreservesTheOriginalReplicaCountAndPerformsOneUpdate(t *testing.T) {
	ctx := t.Context()
	deployment := newIdempotencyDeployment(3)
	fakeClient, updateCount := newScaleIdempotencyClient(t, deployment)
	manager := NewManager(fakeClient, testNamespace)

	require.NoError(t, manager.ScaleDown(ctx))
	require.NoError(t, manager.ScaleDown(ctx))

	persisted := &appsv1.Deployment{}
	require.NoError(t, fakeClient.Get(ctx, client.ObjectKeyFromObject(deployment), persisted))
	assert.Equal(t, int32(0), *persisted.Spec.Replicas)
	assert.Equal(t, "3", persisted.Labels[labelScaledownReplicas])
	assert.Equal(t, int32(1), updateCount.Load())
}

func TestRepeatedScaleUpRestoresTheOriginalReplicaCountAndPerformsOneUpdate(t *testing.T) {
	ctx := t.Context()
	deployment := newIdempotencyDeployment(0)
	deployment.Labels[labelScaledownReplicas] = "3"
	fakeClient, updateCount := newScaleIdempotencyClient(t, deployment)
	manager := NewManager(fakeClient, testNamespace)

	require.NoError(t, manager.ScaleUp(ctx))
	require.NoError(t, manager.ScaleUp(ctx))

	persisted := &appsv1.Deployment{}
	require.NoError(t, fakeClient.Get(ctx, client.ObjectKeyFromObject(deployment), persisted))
	assert.Equal(t, int32(3), *persisted.Spec.Replicas)
	assert.NotContains(t, persisted.Labels, labelScaledownReplicas)
	assert.Equal(t, int32(1), updateCount.Load())
}

func newIdempotencyDeployment(replicas int32) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "idempotency-deployment",
			Namespace: testNamespace,
			Labels: map[string]string{
				labelScaledownScope: "restore",
			},
		},
		Spec: appsv1.DeploymentSpec{Replicas: &replicas},
	}
}

func newScaleIdempotencyClient(
	t *testing.T,
	objects ...client.Object,
) (client.WithWatch, *atomic.Int32) {
	t.Helper()

	scheme := runtime.NewScheme()
	require.NoError(t, appsv1.AddToScheme(scheme))
	require.NoError(t, corev1.AddToScheme(scheme))

	updateCount := &atomic.Int32{}
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(objects...).
		WithInterceptorFuncs(interceptor.Funcs{
			Update: func(ctx context.Context, wrapped client.WithWatch, object client.Object, opts ...client.UpdateOption) error {
				updateCount.Add(1)
				return wrapped.Update(ctx, object, opts...)
			},
		}).
		Build()
	return fakeClient, updateCount
}
