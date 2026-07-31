package velero

import (
	"context"
	"testing"

	v1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestNewDefaultRestoreManager(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// when
		newDefaultRestoreManager := newDefaultRestoreManager(newMockK8sWatchClient(t), newMockEventRecorder(t))

		// then
		require.NotEmpty(t, newDefaultRestoreManager)
	})
}

func Test_defaultRestoreManager_DeleteRestore(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		restore := &v1.Restore{ObjectMeta: metav1.ObjectMeta{Name: "restore", Namespace: testNamespace}, Spec: v1.RestoreSpec{BackupName: "backup"}}
		veleroRestore := &velerov1.Restore{ObjectMeta: metav1.ObjectMeta{Name: "restore", Namespace: testNamespace}}

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(restore, corev1.EventTypeNormal, v1.DeleteEventReason, "Using velero as restore provider")

		mockK8sWatchClient := newMockK8sWatchClient(t)
		mockK8sWatchClient.EXPECT().Get(testCtx, client.ObjectKeyFromObject(restore), &velerov1.Restore{}).RunAndReturn(
			func(_ context.Context, _ client.ObjectKey, actual client.Object, _ ...client.GetOption) error {
				*actual.(*velerov1.Restore) = *veleroRestore
				return nil
			},
		)
		mockK8sWatchClient.EXPECT().Delete(testCtx, veleroRestore).Return(nil)

		sut := &defaultRestoreManager{k8sClient: mockK8sWatchClient, recorder: recorderMock}

		// when
		err := sut.DeleteRestore(testCtx, restore)

		// then
		require.NoError(t, err)
	})

	t.Run("should ignore if velero restore resource is not found", func(t *testing.T) {
		// given
		restore := &v1.Restore{ObjectMeta: metav1.ObjectMeta{Name: "restore", Namespace: testNamespace}, Spec: v1.RestoreSpec{BackupName: "backup"}}

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(restore, corev1.EventTypeNormal, v1.DeleteEventReason, "Using velero as restore provider")

		mockK8sWatchClient := newMockK8sWatchClient(t)
		mockK8sWatchClient.EXPECT().Get(testCtx, client.ObjectKeyFromObject(restore), &velerov1.Restore{}).Return(errors.NewNotFound(schema.GroupResource{
			Group:    "velero.io",
			Resource: "restores",
		}, "restore"))

		sut := &defaultRestoreManager{k8sClient: mockK8sWatchClient, recorder: recorderMock}

		// when
		err := sut.DeleteRestore(testCtx, restore)

		// then
		require.NoError(t, err)
	})

	t.Run("should return error on get error", func(t *testing.T) {
		// given
		restore := &v1.Restore{ObjectMeta: metav1.ObjectMeta{Name: "restore", Namespace: testNamespace}, Spec: v1.RestoreSpec{BackupName: "backup"}}

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(restore, corev1.EventTypeNormal, v1.DeleteEventReason, "Using velero as restore provider")

		mockK8sWatchClient := newMockK8sWatchClient(t)
		mockK8sWatchClient.EXPECT().Get(testCtx, client.ObjectKeyFromObject(restore), &velerov1.Restore{}).Return(assert.AnError)

		sut := &defaultRestoreManager{k8sClient: mockK8sWatchClient, recorder: recorderMock}

		// when
		err := sut.DeleteRestore(testCtx, restore)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get velero restore [restore]")
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("should ignore if velero restore is deleted concurrently", func(t *testing.T) {
		// given
		restore := &v1.Restore{ObjectMeta: metav1.ObjectMeta{Name: "restore", Namespace: testNamespace}, Spec: v1.RestoreSpec{BackupName: "backup"}}
		veleroRestore := &velerov1.Restore{ObjectMeta: metav1.ObjectMeta{Name: "restore", Namespace: testNamespace}}

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(restore, corev1.EventTypeNormal, v1.DeleteEventReason, "Using velero as restore provider")

		mockK8sWatchClient := newMockK8sWatchClient(t)
		mockK8sWatchClient.EXPECT().Get(testCtx, client.ObjectKeyFromObject(restore), &velerov1.Restore{}).RunAndReturn(
			func(_ context.Context, _ client.ObjectKey, actual client.Object, _ ...client.GetOption) error {
				*actual.(*velerov1.Restore) = *veleroRestore
				return nil
			},
		)
		mockK8sWatchClient.EXPECT().Delete(testCtx, veleroRestore).Return(errors.NewNotFound(schema.GroupResource{
			Group:    "velero.io",
			Resource: "restores",
		}, "restore"))

		sut := &defaultRestoreManager{k8sClient: mockK8sWatchClient, recorder: recorderMock}

		// when
		err := sut.DeleteRestore(testCtx, restore)

		// then
		require.NoError(t, err)
	})

	t.Run("should return error on delete error", func(t *testing.T) {
		// given
		restore := &v1.Restore{ObjectMeta: metav1.ObjectMeta{Name: "restore", Namespace: testNamespace}, Spec: v1.RestoreSpec{BackupName: "backup"}}
		veleroRestore := &velerov1.Restore{ObjectMeta: metav1.ObjectMeta{Name: "restore", Namespace: testNamespace}}

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(restore, corev1.EventTypeNormal, v1.DeleteEventReason, "Using velero as restore provider")

		mockK8sWatchClient := newMockK8sWatchClient(t)
		mockK8sWatchClient.EXPECT().Get(testCtx, client.ObjectKeyFromObject(restore), &velerov1.Restore{}).RunAndReturn(
			func(_ context.Context, _ client.ObjectKey, actual client.Object, _ ...client.GetOption) error {
				*actual.(*velerov1.Restore) = *veleroRestore
				return nil
			},
		)
		mockK8sWatchClient.EXPECT().Delete(testCtx, veleroRestore).Return(assert.AnError)

		sut := &defaultRestoreManager{k8sClient: mockK8sWatchClient, recorder: recorderMock}
		// when
		err := sut.DeleteRestore(testCtx, restore)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to delete velero restore [restore]")
		assert.ErrorIs(t, err, assert.AnError)
	})
}
