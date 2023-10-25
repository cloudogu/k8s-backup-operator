package maintenance

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"testing"
	"time"
)

var testCtx = context.TODO()

func TestNewWithLooseCoupling(t *testing.T) {
	// when
	actual := NewWithLooseCoupling(nil, nil, nil)

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
		err := sut.ActivateMaintenanceMode(testCtx, "title", "text")

		// then
		require.NoError(t, err)
	})

	t.Run("should not activate maintenance mode if etcd-headless service is not found", func(t *testing.T) {
		// given
		notFoundErr := errors.NewNotFound(schema.GroupResource{
			Group:    "core/v1",
			Resource: "Service",
		}, "etcd")
		statefulSetClient := newMockStatefulSetInterface(t)
		statefulSetClient.EXPECT().Get(context.Background(), "etcd", metav1.GetOptions{}).Return(nil, nil)
		serviceClient := newMockServiceInterface(t)
		serviceClient.EXPECT().Get(context.Background(), "etcd-headless", metav1.GetOptions{}).Return(nil, notFoundErr)

		sut := &looselyCoupledMaintenanceSwitch{statefulSetClient: statefulSetClient, serviceInterface: serviceClient}

		// when
		err := sut.ActivateMaintenanceMode(testCtx, "title", "text")

		// then
		require.NoError(t, err)
	})

	t.Run("should not activate maintenance mode if etcd service is not found", func(t *testing.T) {
		// given
		notFoundErr := errors.NewNotFound(schema.GroupResource{
			Group:    "core/v1",
			Resource: "Service",
		}, "etcd")
		statefulSetClient := newMockStatefulSetInterface(t)
		statefulSetClient.EXPECT().Get(context.Background(), "etcd", metav1.GetOptions{}).Return(nil, nil)
		serviceClient := newMockServiceInterface(t)
		serviceClient.EXPECT().Get(context.Background(), "etcd-headless", metav1.GetOptions{}).Return(nil, nil)
		serviceClient.EXPECT().Get(context.Background(), "etcd", metav1.GetOptions{}).Return(nil, notFoundErr)

		sut := &looselyCoupledMaintenanceSwitch{statefulSetClient: statefulSetClient, serviceInterface: serviceClient}

		// when
		err := sut.ActivateMaintenanceMode(testCtx, "title", "text")

		// then
		require.NoError(t, err)
	})

	t.Run("should fail if getting etcd returns any error other than 'not found'", func(t *testing.T) {
		// given
		statefulSetClient := newMockStatefulSetInterface(t)
		statefulSetClient.EXPECT().Get(context.Background(), "etcd", metav1.GetOptions{}).Return(nil, assert.AnError)

		sut := &looselyCoupledMaintenanceSwitch{statefulSetClient: statefulSetClient}

		// when
		err := sut.ActivateMaintenanceMode(testCtx, "title", "text")

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to check if etcd is ready")
		assert.ErrorContains(t, err, "failed to get statefulset [etcd]")
	})
	t.Run("should not activate maintenance mode if etcd has no ready replicas", func(t *testing.T) {
		// given
		etcd := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{Name: "etcd", Namespace: "ecosystem"},
			Status:     appsv1.StatefulSetStatus{ReadyReplicas: 0},
		}
		statefulSetClient := newMockStatefulSetInterface(t)
		statefulSetClient.EXPECT().Get(context.Background(), "etcd", metav1.GetOptions{}).Return(etcd, nil)
		serviceClient := newMockServiceInterface(t)
		serviceClient.EXPECT().Get(context.Background(), "etcd-headless", metav1.GetOptions{}).Return(nil, nil)
		serviceClient.EXPECT().Get(context.Background(), "etcd", metav1.GetOptions{}).Return(nil, nil)

		sut := &looselyCoupledMaintenanceSwitch{statefulSetClient: statefulSetClient, serviceInterface: serviceClient}

		// when
		err := sut.ActivateMaintenanceMode(testCtx, "title", "text")

		// then
		require.NoError(t, err)
	})
	t.Run("should activate maintenance mode if etcd has a ready replica", func(t *testing.T) {
		// given
		maintenance := newMockMaintenanceModeSwitch(t)
		maintenance.EXPECT().ActivateMaintenanceMode(testCtx, "title", "text").Return(nil)

		etcd := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{Name: "etcd", Namespace: "ecosystem"},
			Status:     appsv1.StatefulSetStatus{ReadyReplicas: 1},
		}
		statefulSetClient := newMockStatefulSetInterface(t)
		statefulSetClient.EXPECT().Get(context.Background(), "etcd", metav1.GetOptions{}).Return(etcd, nil)
		serviceClient := newMockServiceInterface(t)
		serviceClient.EXPECT().Get(context.Background(), "etcd-headless", metav1.GetOptions{}).Return(nil, nil)
		serviceClient.EXPECT().Get(context.Background(), "etcd", metav1.GetOptions{}).Return(nil, nil)

		sut := &looselyCoupledMaintenanceSwitch{
			maintenanceModeSwitch: maintenance,
			statefulSetClient:     statefulSetClient,
			serviceInterface:      serviceClient,
		}

		// when
		err := sut.ActivateMaintenanceMode(testCtx, "title", "text")

		// then
		require.NoError(t, err)
	})
	t.Run("should fail if activating the maintenance mode fails", func(t *testing.T) {
		// given
		maintenance := newMockMaintenanceModeSwitch(t)
		maintenance.EXPECT().ActivateMaintenanceMode(testCtx, "title", "text").Return(assert.AnError)

		etcd := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{Name: "etcd", Namespace: "ecosystem"},
			Status:     appsv1.StatefulSetStatus{ReadyReplicas: 1},
		}
		statefulSetClient := newMockStatefulSetInterface(t)
		statefulSetClient.EXPECT().Get(context.Background(), "etcd", metav1.GetOptions{}).Return(etcd, nil)
		serviceClient := newMockServiceInterface(t)
		serviceClient.EXPECT().Get(context.Background(), "etcd-headless", metav1.GetOptions{}).Return(nil, nil)
		serviceClient.EXPECT().Get(context.Background(), "etcd", metav1.GetOptions{}).Return(nil, nil)

		sut := &looselyCoupledMaintenanceSwitch{
			maintenanceModeSwitch: maintenance,
			statefulSetClient:     statefulSetClient,
			serviceInterface:      serviceClient,
		}

		// when
		err := sut.ActivateMaintenanceMode(testCtx, "title", "text")

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
	})
}

func Test_looselyCoupledMaintenanceSwitch_DeactivateMaintenanceMode(t *testing.T) {
	t.Run("should fail if watch fails", func(t *testing.T) {
		// given
		statefulSetClient := newMockStatefulSetInterface(t)
		statefulSetClient.EXPECT().Watch(mock.Anything, metav1.ListOptions{FieldSelector: "metadata.name=etcd"}).Return(nil, assert.AnError)

		sut := &looselyCoupledMaintenanceSwitch{
			statefulSetClient: statefulSetClient,
		}

		// when
		err := sut.DeactivateMaintenanceMode(testCtx)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to create watch for StatefulSet etcd")
		assert.ErrorContains(t, err, "failed to wait for ready etcd")
	})
	t.Run("should fail if watch timeout is reached", func(t *testing.T) {
		// given
		oldTimeout := waitForEtcdTimeout
		defer func() { waitForEtcdTimeout = oldTimeout }()
		waitForEtcdTimeout = 1 * time.Millisecond

		watchChan := make(chan watch.Event)
		watchMock := newMockWatcher(t)
		watchMock.EXPECT().Stop().Run(func() { close(watchChan) })
		watchMock.EXPECT().ResultChan().Return(watchChan)

		statefulSetClient := newMockStatefulSetInterface(t)
		statefulSetClient.EXPECT().Watch(mock.Anything, metav1.ListOptions{FieldSelector: "metadata.name=etcd"}).Return(watchMock, nil)

		sut := &looselyCoupledMaintenanceSwitch{
			statefulSetClient: statefulSetClient,
		}

		// when
		err := sut.DeactivateMaintenanceMode(testCtx)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "waiting for etcd to become ready timed out")
		assert.ErrorContains(t, err, "failed to wait for ready etcd")
	})
	t.Run("should fail if watch event has unexpected object type", func(t *testing.T) {
		// given
		oldTimeout := waitForEtcdTimeout
		defer func() { waitForEtcdTimeout = oldTimeout }()
		waitForEtcdTimeout = 1 * time.Millisecond

		watchChan := make(chan watch.Event)
		go func() {
			// goroutine is somehow necessary for event to be recognized
			watchChan <- watch.Event{Object: &corev1.ConfigMap{}}
		}()

		watchMock := newMockWatcher(t)
		watchMock.EXPECT().Stop().Run(func() { close(watchChan) })
		watchMock.EXPECT().ResultChan().Return(watchChan)

		statefulSetClient := newMockStatefulSetInterface(t)
		statefulSetClient.EXPECT().Watch(mock.Anything, metav1.ListOptions{FieldSelector: "metadata.name=etcd"}).Return(watchMock, nil)

		sut := &looselyCoupledMaintenanceSwitch{
			statefulSetClient: statefulSetClient,
		}

		// when
		err := sut.DeactivateMaintenanceMode(testCtx)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "waiting for etcd to become ready timed out")
		assert.ErrorContains(t, err, "failed to wait for ready etcd")
	})
	t.Run("should deactivate maintenance if StatefulSet has ready replicas", func(t *testing.T) {
		// given
		watchChan := make(chan watch.Event)
		go func() {
			// goroutine is somehow necessary for event to be recognized
			watchChan <- watch.Event{Object: &appsv1.StatefulSet{
				Status: appsv1.StatefulSetStatus{ReadyReplicas: 1},
			}}
		}()

		watchMock := newMockWatcher(t)
		watchMock.EXPECT().Stop().Run(func() { close(watchChan) })
		watchMock.EXPECT().ResultChan().Return(watchChan)

		statefulSetClient := newMockStatefulSetInterface(t)
		statefulSetClient.EXPECT().Watch(mock.Anything, metav1.ListOptions{FieldSelector: "metadata.name=etcd"}).Return(watchMock, nil)

		maintenance := newMockMaintenanceModeSwitch(t)
		maintenance.EXPECT().DeactivateMaintenanceMode(testCtx).Return(nil)

		sut := &looselyCoupledMaintenanceSwitch{
			maintenanceModeSwitch: maintenance,
			statefulSetClient:     statefulSetClient,
		}

		// when
		err := sut.DeactivateMaintenanceMode(testCtx)

		// then
		require.NoError(t, err)
	})
	t.Run("should fail if deactivating maintenance fails", func(t *testing.T) {
		// given
		watchChan := make(chan watch.Event)
		go func() {
			// goroutine is somehow necessary for event to be recognized
			watchChan <- watch.Event{Object: &appsv1.StatefulSet{
				Status: appsv1.StatefulSetStatus{ReadyReplicas: 1},
			}}
		}()

		watchMock := newMockWatcher(t)
		watchMock.EXPECT().Stop().Run(func() { close(watchChan) })
		watchMock.EXPECT().ResultChan().Return(watchChan)

		statefulSetClient := newMockStatefulSetInterface(t)
		statefulSetClient.EXPECT().Watch(mock.Anything, metav1.ListOptions{FieldSelector: "metadata.name=etcd"}).Return(watchMock, nil)

		maintenance := newMockMaintenanceModeSwitch(t)
		maintenance.EXPECT().DeactivateMaintenanceMode(testCtx).Return(assert.AnError)

		sut := &looselyCoupledMaintenanceSwitch{
			maintenanceModeSwitch: maintenance,
			statefulSetClient:     statefulSetClient,
		}

		// when
		err := sut.DeactivateMaintenanceMode(testCtx)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
	})
}
