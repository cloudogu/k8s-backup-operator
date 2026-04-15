package scale

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var testCtx = context.TODO()

const testNamespace = "test-namespace"

func int32Pointer(i int32) *int32 {
	return &i
}

func TestNewManager(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		clientMock := newMockK8sClient(t)

		// when
		manager := NewManager(clientMock, testNamespace)

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

		sut := NewManager(clientMock, testNamespace)

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

		sut := NewManager(clientMock, testNamespace)

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

		sut := NewManager(clientMock, testNamespace)

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

		sut := NewManager(clientMock, testNamespace)

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

		sut := NewManager(clientMock, testNamespace)

		// when
		err := sut.ScaleDown(testCtx)

		// then
		require.NoError(t, err)
	})

	t.Run("should return error on list failure", func(t *testing.T) {
		// given
		clientMock := newMockK8sClient(t)
		clientMock.EXPECT().List(testCtx, &appsv1.DeploymentList{}, mock.Anything, mock.Anything).Return(assert.AnError)

		sut := NewManager(clientMock, testNamespace)

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

		sut := NewManager(clientMock, testNamespace)

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

		sut := NewManager(clientMock, testNamespace)

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

		sut := NewManager(clientMock, testNamespace)

		// when
		err := sut.ScaleDown(testCtx)

		// then
		require.NoError(t, err)
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
			return d.Name == "test-deploy" && *d.Spec.Replicas == 3 && !hasReplicasLabel
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

		sut := NewManager(clientMock, testNamespace)

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

		sut := NewManager(clientMock, testNamespace)

		// when
		err := sut.ScaleUp(testCtx)

		// then
		require.NoError(t, err)
		// No Update should have been called
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

		sut := NewManager(clientMock, testNamespace)

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

		sut := NewManager(clientMock, testNamespace)

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

		sut := NewManager(clientMock, testNamespace)

		// when
		err := sut.ScaleUp(testCtx)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to scale up deployment test-deploy")
		assert.ErrorIs(t, err, assert.AnError)
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
			return r.Name == "test-rc" && *r.Spec.Replicas == 5 && !hasReplicasLabel
		})).Return(nil)

		sut := NewManager(clientMock, testNamespace)

		// when
		err := sut.ScaleUp(testCtx)

		// then
		require.NoError(t, err)
	})
}
