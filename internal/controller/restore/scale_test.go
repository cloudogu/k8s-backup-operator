package restore

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func int32Pointer(i int32) *int32 {
	return &i
}

func emptyList(_ context.Context, _ client.ObjectList, _ ...client.ListOption) error {
	return nil
}

func TestNewManager(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		clientMock := newMockK8sClient(t)

		// when
		manager := NewScaleManager(clientMock, testNamespace)

		// then
		require.NotNil(t, manager)
	})
}

func TestDefaultManager_ScaleDown(t *testing.T) {
	t.Run("should scale down deployments", func(t *testing.T) {
		// given
		deploy := appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-deploy",
				Namespace: testNamespace,
				Labels: map[string]string{
					labelScaledownScope: "my-scope",
				},
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: int32Pointer(3),
			},
		}

		clientMock := newMockK8sClient(t)
		clientMock.EXPECT().List(testCtx, &appsv1.DeploymentList{}, mock.Anything, mock.Anything).
			RunAndReturn(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
				list.(*appsv1.DeploymentList).Items = []appsv1.Deployment{deploy}
				return nil
			})
		clientMock.EXPECT().Update(testCtx, mock.MatchedBy(func(obj client.Object) bool {
			d, ok := obj.(*appsv1.Deployment)
			return ok && d.Name == "test-deploy" && *d.Spec.Replicas == 0 && d.Labels[labelScaledownReplicas] == "3"
		})).Return(nil)
		clientMock.EXPECT().List(testCtx, &appsv1.StatefulSetList{}, mock.Anything, mock.Anything).
			RunAndReturn(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
				return nil
			})
		clientMock.EXPECT().List(testCtx, &appsv1.ReplicaSetList{}, mock.Anything, mock.Anything).
			RunAndReturn(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
				return nil
			})
		clientMock.EXPECT().List(testCtx, &corev1.ReplicationControllerList{}, mock.Anything, mock.Anything).
			RunAndReturn(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
				return nil
			})

		sut := NewScaleManager(clientMock, testNamespace)

		// when
		err := sut.ScaleDown(testCtx)

		// then
		require.NoError(t, err)
	})

	t.Run("should scale down statefulsets", func(t *testing.T) {
		// given
		sts := appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-sts",
				Namespace: testNamespace,
				Labels: map[string]string{
					labelScaledownScope: "my-scope",
				},
			},
			Spec: appsv1.StatefulSetSpec{
				Replicas: int32Pointer(2),
			},
		}

		clientMock := newMockK8sClient(t)
		clientMock.EXPECT().List(testCtx, &appsv1.DeploymentList{}, mock.Anything, mock.Anything).
			RunAndReturn(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
				return nil
			})
		clientMock.EXPECT().List(testCtx, &appsv1.StatefulSetList{}, mock.Anything, mock.Anything).
			RunAndReturn(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
				list.(*appsv1.StatefulSetList).Items = []appsv1.StatefulSet{sts}
				return nil
			})
		clientMock.EXPECT().Update(testCtx, mock.MatchedBy(func(obj client.Object) bool {
			s, ok := obj.(*appsv1.StatefulSet)
			return ok && s.Name == "test-sts" && *s.Spec.Replicas == 0 && s.Labels[labelScaledownReplicas] == "2"
		})).Return(nil)
		clientMock.EXPECT().List(testCtx, &appsv1.ReplicaSetList{}, mock.Anything, mock.Anything).
			RunAndReturn(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
				return nil
			})
		clientMock.EXPECT().List(testCtx, &corev1.ReplicationControllerList{}, mock.Anything, mock.Anything).
			RunAndReturn(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
				return nil
			})

		sut := NewScaleManager(clientMock, testNamespace)

		// when
		err := sut.ScaleDown(testCtx)

		// then
		require.NoError(t, err)
	})

	t.Run("should skip replicasets with owner references", func(t *testing.T) {
		// given
		rs := appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-rs",
				Namespace: testNamespace,
				Labels: map[string]string{
					labelScaledownScope: "my-scope",
				},
				OwnerReferences: []metav1.OwnerReference{
					{Name: "parent-deploy", Kind: "Deployment"},
				},
			},
			Spec: appsv1.ReplicaSetSpec{
				Replicas: int32Pointer(3),
			},
		}

		clientMock := newMockK8sClient(t)
		clientMock.EXPECT().List(testCtx, &appsv1.DeploymentList{}, mock.Anything, mock.Anything).
			RunAndReturn(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
				return nil
			})
		clientMock.EXPECT().List(testCtx, &appsv1.StatefulSetList{}, mock.Anything, mock.Anything).
			RunAndReturn(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
				return nil
			})
		clientMock.EXPECT().List(testCtx, &appsv1.ReplicaSetList{}, mock.Anything, mock.Anything).
			RunAndReturn(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
				list.(*appsv1.ReplicaSetList).Items = []appsv1.ReplicaSet{rs}
				return nil
			})
		clientMock.EXPECT().List(testCtx, &corev1.ReplicationControllerList{}, mock.Anything, mock.Anything).
			RunAndReturn(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
				return nil
			})

		sut := NewScaleManager(clientMock, testNamespace)

		// when
		err := sut.ScaleDown(testCtx)

		// then
		require.NoError(t, err)
		// No Update should have been called
	})

	t.Run("should skip already scaled down resources", func(t *testing.T) {
		// given
		deploy := appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-deploy",
				Namespace: testNamespace,
				Labels: map[string]string{
					labelScaledownScope:    "my-scope",
					labelScaledownReplicas: "3",
				},
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: int32Pointer(0),
			},
		}

		clientMock := newMockK8sClient(t)
		clientMock.EXPECT().List(testCtx, &appsv1.DeploymentList{}, mock.Anything, mock.Anything).
			RunAndReturn(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
				list.(*appsv1.DeploymentList).Items = []appsv1.Deployment{deploy}
				return nil
			})
		clientMock.EXPECT().List(testCtx, &appsv1.StatefulSetList{}, mock.Anything, mock.Anything).
			RunAndReturn(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
				return nil
			})
		clientMock.EXPECT().List(testCtx, &appsv1.ReplicaSetList{}, mock.Anything, mock.Anything).
			RunAndReturn(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
				return nil
			})
		clientMock.EXPECT().List(testCtx, &corev1.ReplicationControllerList{}, mock.Anything, mock.Anything).
			RunAndReturn(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
				return nil
			})

		sut := NewScaleManager(clientMock, testNamespace)

		// when
		err := sut.ScaleDown(testCtx)

		// then
		require.NoError(t, err)
		// No Update should have been called
	})

	t.Run("should default replicas to 0 when nil", func(t *testing.T) {
		// given
		deploy := appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-deploy",
				Namespace: testNamespace,
				Labels: map[string]string{
					labelScaledownScope: "my-scope",
				},
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: nil,
			},
		}

		clientMock := newMockK8sClient(t)
		clientMock.EXPECT().List(testCtx, &appsv1.DeploymentList{}, mock.Anything, mock.Anything).
			RunAndReturn(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
				list.(*appsv1.DeploymentList).Items = []appsv1.Deployment{deploy}
				return nil
			})
		clientMock.EXPECT().Update(testCtx, mock.MatchedBy(func(obj client.Object) bool {
			d, ok := obj.(*appsv1.Deployment)
			return ok && d.Labels[labelScaledownReplicas] == "0"
		})).Return(nil)
		clientMock.EXPECT().List(testCtx, &appsv1.StatefulSetList{}, mock.Anything, mock.Anything).
			RunAndReturn(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
				return nil
			})
		clientMock.EXPECT().List(testCtx, &appsv1.ReplicaSetList{}, mock.Anything, mock.Anything).
			RunAndReturn(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
				return nil
			})
		clientMock.EXPECT().List(testCtx, &corev1.ReplicationControllerList{}, mock.Anything, mock.Anything).
			RunAndReturn(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
				return nil
			})

		sut := NewScaleManager(clientMock, testNamespace)

		// when
		err := sut.ScaleDown(testCtx)

		// then
		require.NoError(t, err)
	})

	t.Run("should return error on list failure", func(t *testing.T) {
		// given
		clientMock := newMockK8sClient(t)
		clientMock.EXPECT().List(testCtx, &appsv1.DeploymentList{}, mock.Anything, mock.Anything).Return(assert.AnError)

		sut := NewScaleManager(clientMock, testNamespace)

		// when
		err := sut.ScaleDown(testCtx)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to list deployments for scaledown")
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("should return error on update failure", func(t *testing.T) {
		// given
		deploy := appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-deploy",
				Namespace: testNamespace,
				Labels: map[string]string{
					labelScaledownScope: "my-scope",
				},
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: int32Pointer(3),
			},
		}

		clientMock := newMockK8sClient(t)
		clientMock.EXPECT().List(testCtx, &appsv1.DeploymentList{}, mock.Anything, mock.Anything).
			RunAndReturn(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
				list.(*appsv1.DeploymentList).Items = []appsv1.Deployment{deploy}
				return nil
			})
		clientMock.EXPECT().Update(testCtx, mock.Anything).Return(assert.AnError)

		sut := NewScaleManager(clientMock, testNamespace)

		// when
		err := sut.ScaleDown(testCtx)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to scale down deployment test-deploy")
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("should scale down replicationcontrollers", func(t *testing.T) {
		// given
		rc := corev1.ReplicationController{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-rc",
				Namespace: testNamespace,
				Labels: map[string]string{
					labelScaledownScope: "my-scope",
				},
			},
			Spec: corev1.ReplicationControllerSpec{
				Replicas: int32Pointer(5),
			},
		}

		clientMock := newMockK8sClient(t)
		clientMock.EXPECT().List(testCtx, &appsv1.DeploymentList{}, mock.Anything, mock.Anything).
			RunAndReturn(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
				return nil
			})
		clientMock.EXPECT().List(testCtx, &appsv1.StatefulSetList{}, mock.Anything, mock.Anything).
			RunAndReturn(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
				return nil
			})
		clientMock.EXPECT().List(testCtx, &appsv1.ReplicaSetList{}, mock.Anything, mock.Anything).
			RunAndReturn(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
				return nil
			})
		clientMock.EXPECT().List(testCtx, &corev1.ReplicationControllerList{}, mock.Anything, mock.Anything).
			RunAndReturn(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
				list.(*corev1.ReplicationControllerList).Items = []corev1.ReplicationController{rc}
				return nil
			})
		clientMock.EXPECT().Update(testCtx, mock.MatchedBy(func(obj client.Object) bool {
			r, ok := obj.(*corev1.ReplicationController)
			return ok && r.Name == "test-rc" && *r.Spec.Replicas == 0 && r.Labels[labelScaledownReplicas] == "5"
		})).Return(nil)

		sut := NewScaleManager(clientMock, testNamespace)

		// when
		err := sut.ScaleDown(testCtx)

		// then
		require.NoError(t, err)
	})

	t.Run("should succeed with no resources", func(t *testing.T) {
		// given
		clientMock := newMockK8sClient(t)
		clientMock.EXPECT().List(testCtx, &appsv1.DeploymentList{}, mock.Anything, mock.Anything).
			RunAndReturn(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
				return nil
			})
		clientMock.EXPECT().List(testCtx, &appsv1.StatefulSetList{}, mock.Anything, mock.Anything).
			RunAndReturn(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
				return nil
			})
		clientMock.EXPECT().List(testCtx, &appsv1.ReplicaSetList{}, mock.Anything, mock.Anything).
			RunAndReturn(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
				return nil
			})
		clientMock.EXPECT().List(testCtx, &corev1.ReplicationControllerList{}, mock.Anything, mock.Anything).
			RunAndReturn(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
				return nil
			})

		sut := NewScaleManager(clientMock, testNamespace)

		// when
		err := sut.ScaleDown(testCtx)

		// then
		require.NoError(t, err)
	})

	t.Run("should scale down standalone replicaset", func(t *testing.T) {
		// given
		rs := appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "test-rs",
				Labels: map[string]string{labelScaledownScope: "my-scope"},
			},
			Spec: appsv1.ReplicaSetSpec{Replicas: int32Pointer(4)},
		}

		clientMock := newMockK8sClient(t)
		clientMock.EXPECT().List(testCtx, &appsv1.DeploymentList{}, mock.Anything, mock.Anything).RunAndReturn(emptyList)
		clientMock.EXPECT().List(testCtx, &appsv1.StatefulSetList{}, mock.Anything, mock.Anything).RunAndReturn(emptyList)
		clientMock.EXPECT().List(testCtx, &appsv1.ReplicaSetList{}, mock.Anything, mock.Anything).
			RunAndReturn(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
				list.(*appsv1.ReplicaSetList).Items = []appsv1.ReplicaSet{rs}
				return nil
			})
		clientMock.EXPECT().Update(testCtx, mock.MatchedBy(func(obj client.Object) bool {
			r := obj.(*appsv1.ReplicaSet)
			return r.Name == "test-rs" && *r.Spec.Replicas == 0 && r.Labels[labelScaledownReplicas] == "4"
		})).Return(nil)
		clientMock.EXPECT().List(testCtx, &corev1.ReplicationControllerList{}, mock.Anything, mock.Anything).RunAndReturn(emptyList)

		// when
		err := NewScaleManager(clientMock, testNamespace).ScaleDown(testCtx)

		// then
		require.NoError(t, err)
	})

	t.Run("should return error on statefulset list failure", func(t *testing.T) {
		// given
		clientMock := newMockK8sClient(t)
		clientMock.EXPECT().List(testCtx, &appsv1.DeploymentList{}, mock.Anything, mock.Anything).RunAndReturn(emptyList)
		clientMock.EXPECT().List(testCtx, &appsv1.StatefulSetList{}, mock.Anything, mock.Anything).Return(assert.AnError)

		// when
		err := NewScaleManager(clientMock, testNamespace).ScaleDown(testCtx)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to scale down StatefulSets")
	})

	t.Run("should return error on replicaset list failure", func(t *testing.T) {
		// given
		clientMock := newMockK8sClient(t)
		clientMock.EXPECT().List(testCtx, &appsv1.DeploymentList{}, mock.Anything, mock.Anything).RunAndReturn(emptyList)
		clientMock.EXPECT().List(testCtx, &appsv1.StatefulSetList{}, mock.Anything, mock.Anything).RunAndReturn(emptyList)
		clientMock.EXPECT().List(testCtx, &appsv1.ReplicaSetList{}, mock.Anything, mock.Anything).Return(assert.AnError)

		// when
		err := NewScaleManager(clientMock, testNamespace).ScaleDown(testCtx)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to scale down ReplicaSets")
	})

	t.Run("should return error on replicationcontroller list failure", func(t *testing.T) {
		// given
		clientMock := newMockK8sClient(t)
		clientMock.EXPECT().List(testCtx, &appsv1.DeploymentList{}, mock.Anything, mock.Anything).RunAndReturn(emptyList)
		clientMock.EXPECT().List(testCtx, &appsv1.StatefulSetList{}, mock.Anything, mock.Anything).RunAndReturn(emptyList)
		clientMock.EXPECT().List(testCtx, &appsv1.ReplicaSetList{}, mock.Anything, mock.Anything).RunAndReturn(emptyList)
		clientMock.EXPECT().List(testCtx, &corev1.ReplicationControllerList{}, mock.Anything, mock.Anything).Return(assert.AnError)

		// when
		err := NewScaleManager(clientMock, testNamespace).ScaleDown(testCtx)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to scale down ReplicationControllers")
	})

	t.Run("should return error on statefulset update failure", func(t *testing.T) {
		// given
		sts := appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "test-sts",
				Labels: map[string]string{labelScaledownScope: "my-scope"},
			},
			Spec: appsv1.StatefulSetSpec{Replicas: int32Pointer(2)},
		}

		clientMock := newMockK8sClient(t)
		clientMock.EXPECT().List(testCtx, &appsv1.DeploymentList{}, mock.Anything, mock.Anything).RunAndReturn(emptyList)
		clientMock.EXPECT().List(testCtx, &appsv1.StatefulSetList{}, mock.Anything, mock.Anything).
			RunAndReturn(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
				list.(*appsv1.StatefulSetList).Items = []appsv1.StatefulSet{sts}
				return nil
			})
		clientMock.EXPECT().Update(testCtx, mock.Anything).Return(assert.AnError)

		// when
		err := NewScaleManager(clientMock, testNamespace).ScaleDown(testCtx)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to scale down statefulset test-sts")
	})

	t.Run("should return error on replicaset update failure", func(t *testing.T) {
		// given
		rs := appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "test-rs",
				Labels: map[string]string{labelScaledownScope: "my-scope"},
			},
			Spec: appsv1.ReplicaSetSpec{Replicas: int32Pointer(2)},
		}

		clientMock := newMockK8sClient(t)
		clientMock.EXPECT().List(testCtx, &appsv1.DeploymentList{}, mock.Anything, mock.Anything).RunAndReturn(emptyList)
		clientMock.EXPECT().List(testCtx, &appsv1.StatefulSetList{}, mock.Anything, mock.Anything).RunAndReturn(emptyList)
		clientMock.EXPECT().List(testCtx, &appsv1.ReplicaSetList{}, mock.Anything, mock.Anything).
			RunAndReturn(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
				list.(*appsv1.ReplicaSetList).Items = []appsv1.ReplicaSet{rs}
				return nil
			})
		clientMock.EXPECT().Update(testCtx, mock.Anything).Return(assert.AnError)

		// when
		err := NewScaleManager(clientMock, testNamespace).ScaleDown(testCtx)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to scale down replicaset test-rs")
	})

	t.Run("should return error on replicationcontroller update failure", func(t *testing.T) {
		// given
		rc := corev1.ReplicationController{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "test-rc",
				Labels: map[string]string{labelScaledownScope: "my-scope"},
			},
			Spec: corev1.ReplicationControllerSpec{Replicas: int32Pointer(2)},
		}

		clientMock := newMockK8sClient(t)
		clientMock.EXPECT().List(testCtx, &appsv1.DeploymentList{}, mock.Anything, mock.Anything).RunAndReturn(emptyList)
		clientMock.EXPECT().List(testCtx, &appsv1.StatefulSetList{}, mock.Anything, mock.Anything).RunAndReturn(emptyList)
		clientMock.EXPECT().List(testCtx, &appsv1.ReplicaSetList{}, mock.Anything, mock.Anything).RunAndReturn(emptyList)
		clientMock.EXPECT().List(testCtx, &corev1.ReplicationControllerList{}, mock.Anything, mock.Anything).
			RunAndReturn(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
				list.(*corev1.ReplicationControllerList).Items = []corev1.ReplicationController{rc}
				return nil
			})
		clientMock.EXPECT().Update(testCtx, mock.Anything).Return(assert.AnError)

		// when
		err := NewScaleManager(clientMock, testNamespace).ScaleDown(testCtx)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to scale down replicationcontroller test-rc")
	})
}

func TestDefaultManager_ScaleUp(t *testing.T) {
	t.Run("should scale up deployments", func(t *testing.T) {
		// given
		deploy := appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-deploy",
				Namespace: testNamespace,
				Labels: map[string]string{
					labelScaledownScope:    "my-scope",
					labelScaledownReplicas: "3",
				},
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: int32Pointer(0),
			},
		}

		clientMock := newMockK8sClient(t)
		clientMock.EXPECT().List(testCtx, &appsv1.DeploymentList{}, mock.Anything, mock.Anything).
			RunAndReturn(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
				list.(*appsv1.DeploymentList).Items = []appsv1.Deployment{deploy}
				return nil
			})
		clientMock.EXPECT().Update(testCtx, mock.MatchedBy(func(obj client.Object) bool {
			d, ok := obj.(*appsv1.Deployment)
			if !ok {
				return false
			}
			_, hasReplicasLabel := d.Labels[labelScaledownReplicas]
			return d.Name == "test-deploy" && *d.Spec.Replicas == 3 && hasReplicasLabel
		})).Return(nil)
		clientMock.EXPECT().List(testCtx, &appsv1.StatefulSetList{}, mock.Anything, mock.Anything).
			RunAndReturn(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
				return nil
			})
		clientMock.EXPECT().List(testCtx, &appsv1.ReplicaSetList{}, mock.Anything, mock.Anything).
			RunAndReturn(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
				return nil
			})
		clientMock.EXPECT().List(testCtx, &corev1.ReplicationControllerList{}, mock.Anything, mock.Anything).
			RunAndReturn(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
				return nil
			})

		sut := NewScaleManager(clientMock, testNamespace)

		// when
		err := sut.ScaleUp(testCtx)

		// then
		require.NoError(t, err)
	})

	t.Run("should skip resources without replicas label", func(t *testing.T) {
		// given
		deploy := appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-deploy",
				Namespace: testNamespace,
				Labels: map[string]string{
					labelScaledownScope: "my-scope",
				},
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: int32Pointer(1),
			},
		}

		clientMock := newMockK8sClient(t)
		clientMock.EXPECT().List(testCtx, &appsv1.DeploymentList{}, mock.Anything, mock.Anything).
			RunAndReturn(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
				list.(*appsv1.DeploymentList).Items = []appsv1.Deployment{deploy}
				return nil
			})
		clientMock.EXPECT().List(testCtx, &appsv1.StatefulSetList{}, mock.Anything, mock.Anything).
			RunAndReturn(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
				return nil
			})
		clientMock.EXPECT().List(testCtx, &appsv1.ReplicaSetList{}, mock.Anything, mock.Anything).
			RunAndReturn(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
				return nil
			})
		clientMock.EXPECT().List(testCtx, &corev1.ReplicationControllerList{}, mock.Anything, mock.Anything).
			RunAndReturn(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
				return nil
			})

		sut := NewScaleManager(clientMock, testNamespace)

		// when
		err := sut.ScaleUp(testCtx)

		// then
		require.NoError(t, err)
		// No Update should have been called
	})

	t.Run("should not update workloads that already have their target replicas", func(t *testing.T) {
		deployment := appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{
				labelScaledownScope: "my-scope", labelScaledownReplicas: "3",
			}},
			Spec: appsv1.DeploymentSpec{Replicas: int32Pointer(3)},
		}
		statefulSet := appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{
				labelScaledownScope: "my-scope", labelScaledownReplicas: "2",
			}},
			Spec: appsv1.StatefulSetSpec{Replicas: int32Pointer(2)},
		}
		replicaSet := appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{
				labelScaledownScope: "my-scope", labelScaledownReplicas: "4",
			}},
			Spec: appsv1.ReplicaSetSpec{Replicas: int32Pointer(4)},
		}
		replicationController := corev1.ReplicationController{
			ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{
				labelScaledownScope: "my-scope", labelScaledownReplicas: "5",
			}},
			Spec: corev1.ReplicationControllerSpec{Replicas: int32Pointer(5)},
		}

		clientMock := newMockK8sClient(t)
		clientMock.EXPECT().List(testCtx, &appsv1.DeploymentList{}, mock.Anything, mock.Anything).
			RunAndReturn(func(_ context.Context, list client.ObjectList, _ ...client.ListOption) error {
				list.(*appsv1.DeploymentList).Items = []appsv1.Deployment{deployment}
				return nil
			})
		clientMock.EXPECT().List(testCtx, &appsv1.StatefulSetList{}, mock.Anything, mock.Anything).
			RunAndReturn(func(_ context.Context, list client.ObjectList, _ ...client.ListOption) error {
				list.(*appsv1.StatefulSetList).Items = []appsv1.StatefulSet{statefulSet}
				return nil
			})
		clientMock.EXPECT().List(testCtx, &appsv1.ReplicaSetList{}, mock.Anything, mock.Anything).
			RunAndReturn(func(_ context.Context, list client.ObjectList, _ ...client.ListOption) error {
				list.(*appsv1.ReplicaSetList).Items = []appsv1.ReplicaSet{replicaSet}
				return nil
			})
		clientMock.EXPECT().List(testCtx, &corev1.ReplicationControllerList{}, mock.Anything, mock.Anything).
			RunAndReturn(func(_ context.Context, list client.ObjectList, _ ...client.ListOption) error {
				list.(*corev1.ReplicationControllerList).Items = []corev1.ReplicationController{replicationController}
				return nil
			})

		err := NewScaleManager(clientMock, testNamespace).ScaleUp(testCtx)

		require.NoError(t, err)
	})

	t.Run("should return error on invalid replicas label", func(t *testing.T) {
		// given
		deploy := appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-deploy",
				Namespace: testNamespace,
				Labels: map[string]string{
					labelScaledownScope:    "my-scope",
					labelScaledownReplicas: "invalid",
				},
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: int32Pointer(0),
			},
		}

		clientMock := newMockK8sClient(t)
		clientMock.EXPECT().List(testCtx, &appsv1.DeploymentList{}, mock.Anything, mock.Anything).
			RunAndReturn(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
				list.(*appsv1.DeploymentList).Items = []appsv1.Deployment{deploy}
				return nil
			})

		sut := NewScaleManager(clientMock, testNamespace)

		// when
		err := sut.ScaleUp(testCtx)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to parse stored replica count for deployment test-deploy")
	})

	t.Run("should return error on list failure", func(t *testing.T) {
		// given
		clientMock := newMockK8sClient(t)
		clientMock.EXPECT().List(testCtx, &appsv1.DeploymentList{}, mock.Anything, mock.Anything).Return(assert.AnError)

		sut := NewScaleManager(clientMock, testNamespace)

		// when
		err := sut.ScaleUp(testCtx)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to list deployments for scaleup")
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("should return error on update failure", func(t *testing.T) {
		// given
		deploy := appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-deploy",
				Namespace: testNamespace,
				Labels: map[string]string{
					labelScaledownScope:    "my-scope",
					labelScaledownReplicas: "3",
				},
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: int32Pointer(0),
			},
		}

		clientMock := newMockK8sClient(t)
		clientMock.EXPECT().List(testCtx, &appsv1.DeploymentList{}, mock.Anything, mock.Anything).
			RunAndReturn(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
				list.(*appsv1.DeploymentList).Items = []appsv1.Deployment{deploy}
				return nil
			})
		clientMock.EXPECT().Update(testCtx, mock.Anything).Return(assert.AnError)

		sut := NewScaleManager(clientMock, testNamespace)

		// when
		err := sut.ScaleUp(testCtx)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to scale up deployment test-deploy")
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("should scale up statefulsets", func(t *testing.T) {
		// given
		sts := appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "test-sts",
				Labels: map[string]string{labelScaledownScope: "my-scope", labelScaledownReplicas: "2"},
			},
			Spec: appsv1.StatefulSetSpec{Replicas: int32Pointer(0)},
		}

		clientMock := newMockK8sClient(t)
		clientMock.EXPECT().List(testCtx, &appsv1.DeploymentList{}, mock.Anything, mock.Anything).RunAndReturn(emptyList)
		clientMock.EXPECT().List(testCtx, &appsv1.StatefulSetList{}, mock.Anything, mock.Anything).
			RunAndReturn(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
				list.(*appsv1.StatefulSetList).Items = []appsv1.StatefulSet{sts}
				return nil
			})
		clientMock.EXPECT().Update(testCtx, mock.MatchedBy(func(obj client.Object) bool {
			s, ok := obj.(*appsv1.StatefulSet)
			if !ok {
				return false
			}
			_, hasLabel := s.Labels[labelScaledownReplicas]
			return s.Name == "test-sts" && *s.Spec.Replicas == 2 && hasLabel
		})).Return(nil)
		clientMock.EXPECT().List(testCtx, &appsv1.ReplicaSetList{}, mock.Anything, mock.Anything).RunAndReturn(emptyList)
		clientMock.EXPECT().List(testCtx, &corev1.ReplicationControllerList{}, mock.Anything, mock.Anything).RunAndReturn(emptyList)

		// when
		err := NewScaleManager(clientMock, testNamespace).ScaleUp(testCtx)

		// then
		require.NoError(t, err)
	})

	t.Run("should scale up replicasets", func(t *testing.T) {
		// given
		rs := appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "test-rs",
				Labels: map[string]string{labelScaledownScope: "my-scope", labelScaledownReplicas: "4"},
			},
			Spec: appsv1.ReplicaSetSpec{Replicas: int32Pointer(0)},
		}

		clientMock := newMockK8sClient(t)
		clientMock.EXPECT().List(testCtx, &appsv1.DeploymentList{}, mock.Anything, mock.Anything).RunAndReturn(emptyList)
		clientMock.EXPECT().List(testCtx, &appsv1.StatefulSetList{}, mock.Anything, mock.Anything).RunAndReturn(emptyList)
		clientMock.EXPECT().List(testCtx, &appsv1.ReplicaSetList{}, mock.Anything, mock.Anything).
			RunAndReturn(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
				list.(*appsv1.ReplicaSetList).Items = []appsv1.ReplicaSet{rs}
				return nil
			})
		clientMock.EXPECT().Update(testCtx, mock.MatchedBy(func(obj client.Object) bool {
			r, ok := obj.(*appsv1.ReplicaSet)
			if !ok {
				return false
			}
			_, hasLabel := r.Labels[labelScaledownReplicas]
			return r.Name == "test-rs" && *r.Spec.Replicas == 4 && hasLabel
		})).Return(nil)
		clientMock.EXPECT().List(testCtx, &corev1.ReplicationControllerList{}, mock.Anything, mock.Anything).RunAndReturn(emptyList)

		// when
		err := NewScaleManager(clientMock, testNamespace).ScaleUp(testCtx)

		// then
		require.NoError(t, err)
	})

	t.Run("should return error on statefulset list failure during scaleup", func(t *testing.T) {
		// given
		clientMock := newMockK8sClient(t)
		clientMock.EXPECT().List(testCtx, &appsv1.DeploymentList{}, mock.Anything, mock.Anything).RunAndReturn(emptyList)
		clientMock.EXPECT().List(testCtx, &appsv1.StatefulSetList{}, mock.Anything, mock.Anything).Return(assert.AnError)

		// when
		err := NewScaleManager(clientMock, testNamespace).ScaleUp(testCtx)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to scale up StatefulSets")
	})

	t.Run("should return error on replicaset list failure during scaleup", func(t *testing.T) {
		// given
		clientMock := newMockK8sClient(t)
		clientMock.EXPECT().List(testCtx, &appsv1.DeploymentList{}, mock.Anything, mock.Anything).RunAndReturn(emptyList)
		clientMock.EXPECT().List(testCtx, &appsv1.StatefulSetList{}, mock.Anything, mock.Anything).RunAndReturn(emptyList)
		clientMock.EXPECT().List(testCtx, &appsv1.ReplicaSetList{}, mock.Anything, mock.Anything).Return(assert.AnError)

		// when
		err := NewScaleManager(clientMock, testNamespace).ScaleUp(testCtx)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to scale up ReplicaSets")
	})

	t.Run("should return error on replicationcontroller list failure during scaleup", func(t *testing.T) {
		// given
		clientMock := newMockK8sClient(t)
		clientMock.EXPECT().List(testCtx, &appsv1.DeploymentList{}, mock.Anything, mock.Anything).RunAndReturn(emptyList)
		clientMock.EXPECT().List(testCtx, &appsv1.StatefulSetList{}, mock.Anything, mock.Anything).RunAndReturn(emptyList)
		clientMock.EXPECT().List(testCtx, &appsv1.ReplicaSetList{}, mock.Anything, mock.Anything).RunAndReturn(emptyList)
		clientMock.EXPECT().List(testCtx, &corev1.ReplicationControllerList{}, mock.Anything, mock.Anything).Return(assert.AnError)

		// when
		err := NewScaleManager(clientMock, testNamespace).ScaleUp(testCtx)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to scale up ReplicationControllers")
	})

	t.Run("should return error on statefulset parse failure during scaleup", func(t *testing.T) {
		// given
		sts := appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "test-sts",
				Labels: map[string]string{labelScaledownScope: "my-scope", labelScaledownReplicas: "invalid"},
			},
		}

		clientMock := newMockK8sClient(t)
		clientMock.EXPECT().List(testCtx, &appsv1.DeploymentList{}, mock.Anything, mock.Anything).RunAndReturn(emptyList)
		clientMock.EXPECT().List(testCtx, &appsv1.StatefulSetList{}, mock.Anything, mock.Anything).
			RunAndReturn(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
				list.(*appsv1.StatefulSetList).Items = []appsv1.StatefulSet{sts}
				return nil
			})

		// when
		err := NewScaleManager(clientMock, testNamespace).ScaleUp(testCtx)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to parse stored replica count for statefulset test-sts")
	})

	t.Run("should return error on replicaset parse failure during scaleup", func(t *testing.T) {
		// given
		rs := appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "test-rs",
				Labels: map[string]string{labelScaledownScope: "my-scope", labelScaledownReplicas: "invalid"},
			},
		}

		clientMock := newMockK8sClient(t)
		clientMock.EXPECT().List(testCtx, &appsv1.DeploymentList{}, mock.Anything, mock.Anything).RunAndReturn(emptyList)
		clientMock.EXPECT().List(testCtx, &appsv1.StatefulSetList{}, mock.Anything, mock.Anything).RunAndReturn(emptyList)
		clientMock.EXPECT().List(testCtx, &appsv1.ReplicaSetList{}, mock.Anything, mock.Anything).
			RunAndReturn(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
				list.(*appsv1.ReplicaSetList).Items = []appsv1.ReplicaSet{rs}
				return nil
			})

		// when
		err := NewScaleManager(clientMock, testNamespace).ScaleUp(testCtx)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to parse stored replica count for replicaset test-rs")
	})

	t.Run("should return error on replicationcontroller parse failure during scaleup", func(t *testing.T) {
		// given
		rc := corev1.ReplicationController{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "test-rc",
				Labels: map[string]string{labelScaledownScope: "my-scope", labelScaledownReplicas: "invalid"},
			},
		}

		clientMock := newMockK8sClient(t)
		clientMock.EXPECT().List(testCtx, &appsv1.DeploymentList{}, mock.Anything, mock.Anything).RunAndReturn(emptyList)
		clientMock.EXPECT().List(testCtx, &appsv1.StatefulSetList{}, mock.Anything, mock.Anything).RunAndReturn(emptyList)
		clientMock.EXPECT().List(testCtx, &appsv1.ReplicaSetList{}, mock.Anything, mock.Anything).RunAndReturn(emptyList)
		clientMock.EXPECT().List(testCtx, &corev1.ReplicationControllerList{}, mock.Anything, mock.Anything).
			RunAndReturn(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
				list.(*corev1.ReplicationControllerList).Items = []corev1.ReplicationController{rc}
				return nil
			})

		// when
		err := NewScaleManager(clientMock, testNamespace).ScaleUp(testCtx)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to parse stored replica count for replicationcontroller test-rc")
	})

	t.Run("should return error on statefulset update failure during scaleup", func(t *testing.T) {
		// given
		sts := appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "test-sts",
				Labels: map[string]string{labelScaledownScope: "my-scope", labelScaledownReplicas: "2"},
			},
		}

		clientMock := newMockK8sClient(t)
		clientMock.EXPECT().List(testCtx, &appsv1.DeploymentList{}, mock.Anything, mock.Anything).RunAndReturn(emptyList)
		clientMock.EXPECT().List(testCtx, &appsv1.StatefulSetList{}, mock.Anything, mock.Anything).
			RunAndReturn(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
				list.(*appsv1.StatefulSetList).Items = []appsv1.StatefulSet{sts}
				return nil
			})
		clientMock.EXPECT().Update(testCtx, mock.Anything).Return(assert.AnError)

		// when
		err := NewScaleManager(clientMock, testNamespace).ScaleUp(testCtx)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to scale up statefulset test-sts")
	})

	t.Run("should return error on replicaset update failure during scaleup", func(t *testing.T) {
		// given
		rs := appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "test-rs",
				Labels: map[string]string{labelScaledownScope: "my-scope", labelScaledownReplicas: "4"},
			},
		}

		clientMock := newMockK8sClient(t)
		clientMock.EXPECT().List(testCtx, &appsv1.DeploymentList{}, mock.Anything, mock.Anything).RunAndReturn(emptyList)
		clientMock.EXPECT().List(testCtx, &appsv1.StatefulSetList{}, mock.Anything, mock.Anything).RunAndReturn(emptyList)
		clientMock.EXPECT().List(testCtx, &appsv1.ReplicaSetList{}, mock.Anything, mock.Anything).
			RunAndReturn(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
				list.(*appsv1.ReplicaSetList).Items = []appsv1.ReplicaSet{rs}
				return nil
			})
		clientMock.EXPECT().Update(testCtx, mock.Anything).Return(assert.AnError)

		// when
		err := NewScaleManager(clientMock, testNamespace).ScaleUp(testCtx)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to scale up replicaset test-rs")
	})

	t.Run("should return error on replicationcontroller update failure during scaleup", func(t *testing.T) {
		// given
		rc := corev1.ReplicationController{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "test-rc",
				Labels: map[string]string{labelScaledownScope: "my-scope", labelScaledownReplicas: "5"},
			},
		}

		clientMock := newMockK8sClient(t)
		clientMock.EXPECT().List(testCtx, &appsv1.DeploymentList{}, mock.Anything, mock.Anything).RunAndReturn(emptyList)
		clientMock.EXPECT().List(testCtx, &appsv1.StatefulSetList{}, mock.Anything, mock.Anything).RunAndReturn(emptyList)
		clientMock.EXPECT().List(testCtx, &appsv1.ReplicaSetList{}, mock.Anything, mock.Anything).RunAndReturn(emptyList)
		clientMock.EXPECT().List(testCtx, &corev1.ReplicationControllerList{}, mock.Anything, mock.Anything).
			RunAndReturn(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
				list.(*corev1.ReplicationControllerList).Items = []corev1.ReplicationController{rc}
				return nil
			})
		clientMock.EXPECT().Update(testCtx, mock.Anything).Return(assert.AnError)

		// when
		err := NewScaleManager(clientMock, testNamespace).ScaleUp(testCtx)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to scale up replicationcontroller test-rc")
	})

	t.Run("should scale up replicationcontrollers", func(t *testing.T) {
		// given
		rc := corev1.ReplicationController{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-rc",
				Namespace: testNamespace,
				Labels: map[string]string{
					labelScaledownScope:    "my-scope",
					labelScaledownReplicas: "5",
				},
			},
			Spec: corev1.ReplicationControllerSpec{
				Replicas: int32Pointer(0),
			},
		}

		clientMock := newMockK8sClient(t)
		clientMock.EXPECT().List(testCtx, &appsv1.DeploymentList{}, mock.Anything, mock.Anything).
			RunAndReturn(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
				return nil
			})
		clientMock.EXPECT().List(testCtx, &appsv1.StatefulSetList{}, mock.Anything, mock.Anything).
			RunAndReturn(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
				return nil
			})
		clientMock.EXPECT().List(testCtx, &appsv1.ReplicaSetList{}, mock.Anything, mock.Anything).
			RunAndReturn(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
				return nil
			})
		clientMock.EXPECT().List(testCtx, &corev1.ReplicationControllerList{}, mock.Anything, mock.Anything).
			RunAndReturn(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
				list.(*corev1.ReplicationControllerList).Items = []corev1.ReplicationController{rc}
				return nil
			})
		clientMock.EXPECT().Update(testCtx, mock.MatchedBy(func(obj client.Object) bool {
			r, ok := obj.(*corev1.ReplicationController)
			if !ok {
				return false
			}
			_, hasReplicasLabel := r.Labels[labelScaledownReplicas]
			return r.Name == "test-rc" && *r.Spec.Replicas == 5 && hasReplicasLabel
		})).Return(nil)

		sut := NewScaleManager(clientMock, testNamespace)

		// when
		err := sut.ScaleUp(testCtx)

		// then
		require.NoError(t, err)
	})
}

func TestFinalizeScaleUpRemovesReplicaLabelsFromEverySupportedWorkload(t *testing.T) {
	testClient := newTestClient(t, interceptor.Funcs{}, readyWorkloads()...)
	manager := NewScaleManager(testClient, testNamespace)

	err := manager.FinalizeScaleUp(testCtx)

	require.NoError(t, err)
	workloads := []client.Object{
		&appsv1.Deployment{},
		&appsv1.StatefulSet{},
		&appsv1.ReplicaSet{},
		&corev1.ReplicationController{},
	}
	names := []string{"deployment", "statefulset", "replicaset", "replicationcontroller"}
	for i, workload := range workloads {
		require.NoError(t, testClient.Get(testCtx, client.ObjectKey{Namespace: testNamespace, Name: names[i]}, workload))
		assert.Equal(t, "restore", workload.GetLabels()[labelScaledownScope])
		_, hasStoredReplicas := workload.GetLabels()[labelScaledownReplicas]
		assert.False(t, hasStoredReplicas)
	}

	ready, readyErr := manager.AreWorkloadsReady(testCtx)
	require.NoError(t, readyErr)
	assert.True(t, ready, "scope workloads must remain observable after their replica labels were removed")
}

func TestFinalizeScaleUpIsIdempotent(t *testing.T) {
	testClient := newTestClient(t, interceptor.Funcs{}, readyWorkloads()...)
	manager := NewScaleManager(testClient, testNamespace)

	require.NoError(t, manager.FinalizeScaleUp(testCtx))
	require.NoError(t, manager.FinalizeScaleUp(testCtx))
}

func TestFinalizeScaleUpCanFinishAPartiallyFailedCleanup(t *testing.T) {
	failStatefulSetOnce := true
	failingUpdate := interceptor.Funcs{
		Update: func(ctx context.Context, wrapped client.WithWatch, object client.Object, opts ...client.UpdateOption) error {
			if _, isStatefulSet := object.(*appsv1.StatefulSet); isStatefulSet && failStatefulSetOnce {
				failStatefulSetOnce = false
				return assert.AnError
			}

			return wrapped.Update(ctx, object, opts...)
		},
	}
	testClient := newTestClient(t, failingUpdate, readyWorkloads()...)
	manager := NewScaleManager(testClient, testNamespace)

	firstErr := manager.FinalizeScaleUp(testCtx)

	require.ErrorIs(t, firstErr, assert.AnError)
	assert.ErrorContains(t, firstErr, "failed to finalize scale-up for statefulset statefulset")

	require.NoError(t, manager.FinalizeScaleUp(testCtx))
	for _, workload := range []client.Object{
		&appsv1.Deployment{}, &appsv1.StatefulSet{}, &appsv1.ReplicaSet{}, &corev1.ReplicationController{},
	} {
		listKey := client.ObjectKey{Namespace: testNamespace}
		switch workload.(type) {
		case *appsv1.Deployment:
			listKey.Name = "deployment"
		case *appsv1.StatefulSet:
			listKey.Name = "statefulset"
		case *appsv1.ReplicaSet:
			listKey.Name = "replicaset"
		case *corev1.ReplicationController:
			listKey.Name = "replicationcontroller"
		}
		require.NoError(t, testClient.Get(testCtx, listKey, workload))
		_, hasStoredReplicas := workload.GetLabels()[labelScaledownReplicas]
		assert.False(t, hasStoredReplicas)
	}
}

func TestFinalizeScaleUpReturnsListErrors(t *testing.T) {
	failingList := interceptor.Funcs{
		List: func(_ context.Context, _ client.WithWatch, list client.ObjectList, _ ...client.ListOption) error {
			if _, isDeploymentList := list.(*appsv1.DeploymentList); isDeploymentList {
				return assert.AnError
			}
			return nil
		},
	}
	manager := NewScaleManager(newTestClient(t, failingList), testNamespace)

	err := manager.FinalizeScaleUp(testCtx)

	require.ErrorIs(t, err, assert.AnError)
	assert.ErrorContains(t, err, "failed to list deployments for scale-up finalization")
}

func recoveryLabels(target string) map[string]string {
	return map[string]string{
		labelScaledownScope:    "restore",
		labelScaledownReplicas: target,
	}
}

func readyWorkloads() []client.Object {
	return []client.Object{
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "deployment", Namespace: testNamespace, Generation: 2, Labels: recoveryLabels("3")},
			Spec:       appsv1.DeploymentSpec{Replicas: int32Pointer(3)},
			Status: appsv1.DeploymentStatus{
				ObservedGeneration: 2, Replicas: 3, UpdatedReplicas: 3, ReadyReplicas: 3, AvailableReplicas: 3,
			},
		},
		&appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{Name: "statefulset", Namespace: testNamespace, Generation: 2, Labels: recoveryLabels("2")},
			Spec:       appsv1.StatefulSetSpec{Replicas: int32Pointer(2)},
			Status: appsv1.StatefulSetStatus{
				ObservedGeneration: 2, Replicas: 2, ReadyReplicas: 2, AvailableReplicas: 2,
			},
		},
		&appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{Name: "replicaset", Namespace: testNamespace, Generation: 2, Labels: recoveryLabels("4")},
			Spec:       appsv1.ReplicaSetSpec{Replicas: int32Pointer(4)},
			Status: appsv1.ReplicaSetStatus{
				ObservedGeneration: 2, Replicas: 4, ReadyReplicas: 4, AvailableReplicas: 4,
			},
		},
		&corev1.ReplicationController{
			ObjectMeta: metav1.ObjectMeta{Name: "replicationcontroller", Namespace: testNamespace, Generation: 2, Labels: recoveryLabels("5")},
			Spec:       corev1.ReplicationControllerSpec{Replicas: int32Pointer(5)},
			Status: corev1.ReplicationControllerStatus{
				ObservedGeneration: 2, Replicas: 5, ReadyReplicas: 5, AvailableReplicas: 5,
			},
		},
	}
}

func TestAreWorkloadsReadyReturnsTrueWhenEveryMarkedWorkloadReachedItsTarget(t *testing.T) {
	manager := NewScaleManager(newTestClient(t, interceptor.Funcs{}, readyWorkloads()...), testNamespace)

	ready, err := manager.AreWorkloadsReady(testCtx)

	require.NoError(t, err)
	assert.True(t, ready)
}

func TestAreWorkloadsReadyReturnsFalseForEverySupportedUnreadyWorkloadKind(t *testing.T) {
	for _, workload := range readyWorkloads() {
		t.Run(fmt.Sprintf("%T", workload), func(t *testing.T) {
			switch typed := workload.(type) {
			case *appsv1.Deployment:
				typed.Status.ReadyReplicas--
			case *appsv1.StatefulSet:
				typed.Status.ReadyReplicas--
			case *appsv1.ReplicaSet:
				typed.Status.ReadyReplicas--
			case *corev1.ReplicationController:
				typed.Status.ReadyReplicas--
			}

			manager := NewScaleManager(newTestClient(t, interceptor.Funcs{}, workload), testNamespace)

			ready, err := manager.AreWorkloadsReady(testCtx)

			require.NoError(t, err)
			assert.False(t, ready)
		})
	}
}

func TestAreWorkloadsReadyTreatsAnEmptyRecoverySetAsReady(t *testing.T) {
	manager := NewScaleManager(newTestClient(t, interceptor.Funcs{}), testNamespace)

	ready, err := manager.AreWorkloadsReady(testCtx)

	require.NoError(t, err)
	assert.True(t, ready)
}

func TestAreWorkloadsReadyAcceptsAConvergedZeroReplicaTarget(t *testing.T) {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "deployment", Namespace: testNamespace, Generation: 2, Labels: recoveryLabels("0")},
		Spec:       appsv1.DeploymentSpec{Replicas: int32Pointer(0)},
		Status:     appsv1.DeploymentStatus{ObservedGeneration: 2},
	}
	manager := NewScaleManager(newTestClient(t, interceptor.Funcs{}, deployment), testNamespace)

	ready, err := manager.AreWorkloadsReady(testCtx)

	require.NoError(t, err)
	assert.True(t, ready)
}

func TestAreWorkloadsReadyRejectsAnInvalidStoredReplicaCount(t *testing.T) {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "deployment", Namespace: testNamespace, Labels: recoveryLabels("invalid")},
	}
	manager := NewScaleManager(newTestClient(t, interceptor.Funcs{}, deployment), testNamespace)

	ready, err := manager.AreWorkloadsReady(testCtx)

	require.Error(t, err)
	assert.False(t, ready)
	assert.ErrorContains(t, err, "failed to parse stored replica count for deployment deployment")
}

func TestAreWorkloadsReadyReturnsListErrors(t *testing.T) {
	failingList := interceptor.Funcs{
		List: func(ctx context.Context, wrapped client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
			if _, isDeploymentList := list.(*appsv1.DeploymentList); isDeploymentList {
				return assert.AnError
			}

			return wrapped.List(ctx, list, opts...)
		},
	}
	manager := NewScaleManager(newTestClient(t, failingList), testNamespace)

	ready, err := manager.AreWorkloadsReady(testCtx)

	require.Error(t, err)
	assert.False(t, ready)
	assert.ErrorIs(t, err, assert.AnError)
	assert.ErrorContains(t, err, "failed to list deployments for readiness check")
}
