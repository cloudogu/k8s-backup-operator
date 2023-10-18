package maintenance

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"testing"
)

func TestNewWithLooseCoupling(t *testing.T) {
	// when
	actual := NewWithLooseCoupling(nil, nil)

	// then
	require.NotEmpty(t, actual)
}

func Test_looselyCoupledMaintenanceSwitch_ActivateMaintenanceMode(t *testing.T) {
	t.Run("should not activate maintenance mode if etcd is not found", func(t *testing.T) {
		// given
		notFoundErr := errors.NewNotFound(schema.GroupResource{
			Group:    "apps/v1",
			Resource: "StatefulSet",
		}, "etcd")
		statefulSetClient := newMockStatefulSetInterface(t)
		statefulSetClient.EXPECT().Get(context.Background(), "etcd", metav1.GetOptions{}).Return(nil, notFoundErr)

		sut := &looselyCoupledMaintenanceSwitch{statefulSetClient: statefulSetClient}

		// when
		err := sut.ActivateMaintenanceMode("title", "text")

		// then
		require.NoError(t, err)
	})
	t.Run("should fail if getting etcd returns any error other than 'not found'", func(t *testing.T) {
		// given
		statefulSetClient := newMockStatefulSetInterface(t)
		statefulSetClient.EXPECT().Get(context.Background(), "etcd", metav1.GetOptions{}).Return(nil, assert.AnError)

		sut := &looselyCoupledMaintenanceSwitch{statefulSetClient: statefulSetClient}

		// when
		err := sut.ActivateMaintenanceMode("title", "text")

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to check if etcd is ready")
		assert.ErrorContains(t, err, "failed to get StatefulSet etcd")
	})
	t.Run("should not activate maintenance mode if etcd has no ready replicas", func(t *testing.T) {
		// given
		etcd := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{Name: "etcd", Namespace: "ecosystem"},
			Status:     appsv1.StatefulSetStatus{ReadyReplicas: 0},
		}
		statefulSetClient := newMockStatefulSetInterface(t)
		statefulSetClient.EXPECT().Get(context.Background(), "etcd", metav1.GetOptions{}).Return(etcd, nil)

		sut := &looselyCoupledMaintenanceSwitch{statefulSetClient: statefulSetClient}

		// when
		err := sut.ActivateMaintenanceMode("title", "text")

		// then
		require.NoError(t, err)
	})
	t.Run("should activate maintenance mode if etcd has a ready replica", func(t *testing.T) {
		// given
		maintenance := newMockMaintenanceModeSwitch(t)
		maintenance.EXPECT().ActivateMaintenanceMode("title", "text").Return(nil)

		etcd := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{Name: "etcd", Namespace: "ecosystem"},
			Status:     appsv1.StatefulSetStatus{ReadyReplicas: 1},
		}
		statefulSetClient := newMockStatefulSetInterface(t)
		statefulSetClient.EXPECT().Get(context.Background(), "etcd", metav1.GetOptions{}).Return(etcd, nil)

		sut := &looselyCoupledMaintenanceSwitch{
			maintenanceModeSwitch: maintenance,
			statefulSetClient:     statefulSetClient,
		}

		// when
		err := sut.ActivateMaintenanceMode("title", "text")

		// then
		require.NoError(t, err)
	})
	t.Run("should fail if activating the maintenance mode fails", func(t *testing.T) {
		// given
		maintenance := newMockMaintenanceModeSwitch(t)
		maintenance.EXPECT().ActivateMaintenanceMode("title", "text").Return(assert.AnError)

		etcd := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{Name: "etcd", Namespace: "ecosystem"},
			Status:     appsv1.StatefulSetStatus{ReadyReplicas: 1},
		}
		statefulSetClient := newMockStatefulSetInterface(t)
		statefulSetClient.EXPECT().Get(context.Background(), "etcd", metav1.GetOptions{}).Return(etcd, nil)

		sut := &looselyCoupledMaintenanceSwitch{
			maintenanceModeSwitch: maintenance,
			statefulSetClient:     statefulSetClient,
		}

		// when
		err := sut.ActivateMaintenanceMode("title", "text")

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
	})
}
