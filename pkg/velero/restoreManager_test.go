package velero

import (
	"fmt"
	"testing"
	"time"

	v1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
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

func Test_defaultRestoreManager_CreateRestore(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		restore := &v1.Restore{ObjectMeta: metav1.ObjectMeta{Name: "restore", Namespace: testNamespace}, Spec: v1.RestoreSpec{BackupName: "backup"}}
		expectedVeleroRestore := getExpectedVeleroRestore(restore)

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(restore, corev1.EventTypeNormal, "Creation", "Using velero as restore provider")
		recorderMock.EXPECT().Eventf(restore, corev1.EventTypeNormal, "Creation", "Successfully completed velero restore [%s]", "restore")

		mockK8sWatchClient := newMockK8sWatchClient(t)
		mockK8sWatchClient.EXPECT().Create(testCtx, expectedVeleroRestore).Return(nil)
		watchMock := newMockWatchInterface(t)
		channel := make(chan watch.Event)
		watchMock.EXPECT().ResultChan().Return(channel)
		watchMock.EXPECT().Stop().Run(func() {
			close(channel)
		})
		mockK8sWatchClient.EXPECT().Watch(testCtx, &velerov1.RestoreList{}, &client.ListOptions{FieldSelector: fields.ParseSelectorOrDie("metadata.name=restore"), Namespace: testNamespace}).Return(watchMock, nil)

		timer := time.NewTimer(time.Second)
		go func() {
			<-timer.C
			expectedVeleroRestore.Status.Phase = velerov1.RestorePhaseCompleted
			channel <- watch.Event{Type: watch.Modified, Object: expectedVeleroRestore}
		}()

		sut := &defaultRestoreManager{k8sClient: mockK8sWatchClient, recorder: recorderMock}

		// when
		err := sut.CreateRestore(testCtx, restore)

		// then
		require.NoError(t, err)
	})

	t.Run("should return error on velero failure", func(t *testing.T) {
		runVeleroStatusPhaseFailureTest(t, velerov1.RestorePhaseFailed)
	})

	t.Run("should return error on velero partially failure", func(t *testing.T) {
		runVeleroStatusPhaseFailureTest(t, velerov1.RestorePhasePartiallyFailed)
	})

	t.Run("should return error on velero plugin operation partially failure", func(t *testing.T) {
		runVeleroStatusPhaseFailureTest(t, velerov1.RestorePhaseWaitingForPluginOperationsPartiallyFailed)
	})

	t.Run("should return error on velero validation failure", func(t *testing.T) {
		runVeleroStatusPhaseFailureTest(t, velerov1.RestorePhaseFailedValidation)
	})

	t.Run("should return error on create velero restore error", func(t *testing.T) {
		// given
		restore := &v1.Restore{ObjectMeta: metav1.ObjectMeta{Name: "restore", Namespace: testNamespace}, Spec: v1.RestoreSpec{BackupName: "backup"}}
		expectedVeleroRestore := getExpectedVeleroRestore(restore)

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(restore, corev1.EventTypeNormal, "Creation", "Using velero as restore provider")
		recorderMock.EXPECT().Event(restore, corev1.EventTypeWarning, "ErrCreation", "failed to create velero restore [restore]: assert.AnError general error for testing")

		mockK8sWatchClient := newMockK8sWatchClient(t)
		mockK8sWatchClient.EXPECT().Create(testCtx, expectedVeleroRestore).Return(assert.AnError)

		sut := &defaultRestoreManager{k8sClient: mockK8sWatchClient, recorder: recorderMock}

		// when
		err := sut.CreateRestore(testCtx, restore)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to create velero restore [restore]")
	})

	t.Run("should return error on create velero restore watch error", func(t *testing.T) {
		// given
		restore := &v1.Restore{ObjectMeta: metav1.ObjectMeta{Name: "restore", Namespace: testNamespace}, Spec: v1.RestoreSpec{BackupName: "backup"}}
		expectedVeleroRestore := getExpectedVeleroRestore(restore)

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(restore, corev1.EventTypeNormal, "Creation", "Using velero as restore provider")
		recorderMock.EXPECT().Event(restore, corev1.EventTypeWarning, "ErrCreation", "failed to create velero restore watch: assert.AnError general error for testing")

		mockK8sWatchClient := newMockK8sWatchClient(t)
		mockK8sWatchClient.EXPECT().Create(testCtx, expectedVeleroRestore).Return(nil)
		mockK8sWatchClient.EXPECT().Watch(testCtx, &velerov1.RestoreList{}, &client.ListOptions{FieldSelector: fields.ParseSelectorOrDie("metadata.name=restore"), Namespace: testNamespace}).Return(nil, assert.AnError)

		sut := &defaultRestoreManager{k8sClient: mockK8sWatchClient, recorder: recorderMock}

		// when
		err := sut.CreateRestore(testCtx, restore)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to create velero restore watch")
	})

	t.Run("should return error on delete event", func(t *testing.T) {
		// given
		restore := &v1.Restore{ObjectMeta: metav1.ObjectMeta{Name: "restore", Namespace: testNamespace}, Spec: v1.RestoreSpec{BackupName: "backup"}}
		expectedVeleroRestore := getExpectedVeleroRestore(restore)

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(restore, corev1.EventTypeNormal, "Creation", "Using velero as restore provider")
		recorderMock.EXPECT().Event(restore, corev1.EventTypeWarning, "ErrCreation", "failed to complete velero restore [restore]: the restore got deleted")

		mockK8sWatchClient := newMockK8sWatchClient(t)
		mockK8sWatchClient.EXPECT().Create(testCtx, expectedVeleroRestore).Return(nil)
		watchMock := newMockWatchInterface(t)
		channel := make(chan watch.Event)
		watchMock.EXPECT().ResultChan().Return(channel)
		watchMock.EXPECT().Stop().Run(func() {
			close(channel)
		})
		mockK8sWatchClient.EXPECT().Watch(testCtx, &velerov1.RestoreList{}, &client.ListOptions{FieldSelector: fields.ParseSelectorOrDie("metadata.name=restore"), Namespace: testNamespace}).Return(watchMock, nil)

		go func() {
			time.Sleep(time.Second)
			channel <- watch.Event{Type: watch.Modified, Object: &appsv1.Deployment{}}
			time.Sleep(time.Second)
			channel <- watch.Event{Type: watch.Deleted, Object: expectedVeleroRestore}
		}()

		sut := &defaultRestoreManager{k8sClient: mockK8sWatchClient, recorder: recorderMock}

		// when
		err := sut.CreateRestore(testCtx, restore)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to complete velero restore [restore]: the restore got deleted")
	})
}

func getExpectedVeleroRestore(restore *v1.Restore) *velerov1.Restore {
	apiGroup := ""
	return &velerov1.Restore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      restore.Name,
			Namespace: restore.Namespace,
		},
		Spec: velerov1.RestoreSpec{
			BackupName:             restore.Spec.BackupName,
			ExistingResourcePolicy: velerov1.PolicyTypeUpdate,
			ResourceModifier:       &corev1.TypedLocalObjectReference{APIGroup: &apiGroup, Kind: "ConfigMap", Name: "k8s-backup-operator-restore-dogu-modifier"},
		},
	}
}

func runVeleroStatusPhaseFailureTest(t *testing.T, phase velerov1.RestorePhase) {
	// given
	restore := &v1.Restore{ObjectMeta: metav1.ObjectMeta{Name: "restore", Namespace: testNamespace}, Spec: v1.RestoreSpec{BackupName: "backup"}}
	expectedVeleroRestore := getExpectedVeleroRestore(restore)

	recorderMock := newMockEventRecorder(t)
	recorderMock.EXPECT().Event(restore, corev1.EventTypeNormal, "Creation", "Using velero as restore provider")
	recorderMock.EXPECT().Event(restore, corev1.EventTypeWarning, "ErrCreation", fmt.Sprintf("failed to complete velero restore [restore]: has status phase [%s]", phase))

	mockK8sWatchClient := newMockK8sWatchClient(t)
	mockK8sWatchClient.EXPECT().Create(testCtx, expectedVeleroRestore).Return(nil)
	watchMock := newMockWatchInterface(t)
	channel := make(chan watch.Event)
	watchMock.EXPECT().ResultChan().Return(channel)
	watchMock.EXPECT().Stop().Run(func() {
		close(channel)
	})
	mockK8sWatchClient.EXPECT().Watch(testCtx, &velerov1.RestoreList{}, &client.ListOptions{FieldSelector: fields.ParseSelectorOrDie("metadata.name=restore"), Namespace: testNamespace}).Return(watchMock, nil)

	timer := time.NewTimer(time.Second)
	go func() {
		<-timer.C
		expectedVeleroRestore.Status.Phase = phase
		channel <- watch.Event{Type: watch.Modified, Object: expectedVeleroRestore}
	}()

	sut := &defaultRestoreManager{k8sClient: mockK8sWatchClient, recorder: recorderMock}

	// when
	err := sut.CreateRestore(testCtx, restore)

	// then
	require.Error(t, err)
	assert.ErrorContains(t, err, fmt.Sprintf("failed to complete velero restore [restore]: has status phase [%s]", phase))
}

func Test_defaultRestoreManager_DeleteRestore(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		restore := &v1.Restore{ObjectMeta: metav1.ObjectMeta{Name: "restore", Namespace: testNamespace}, Spec: v1.RestoreSpec{BackupName: "backup"}}

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(restore, corev1.EventTypeNormal, v1.DeleteEventReason, "Using velero as restore provider")

		mockK8sWatchClient := newMockK8sWatchClient(t)
		mockK8sWatchClient.EXPECT().Delete(testCtx, restore).Return(nil)

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
		mockK8sWatchClient.EXPECT().Delete(testCtx, restore).Return(errors.NewNotFound(schema.GroupResource{
			Group:    "velero.io",
			Resource: "restore",
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

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(restore, corev1.EventTypeNormal, v1.DeleteEventReason, "Using velero as restore provider")

		mockK8sWatchClient := newMockK8sWatchClient(t)
		mockK8sWatchClient.EXPECT().Delete(testCtx, restore).Return(assert.AnError)

		sut := &defaultRestoreManager{k8sClient: mockK8sWatchClient, recorder: recorderMock}
		// when
		err := sut.DeleteRestore(testCtx, restore)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to delete velero restore [restore]")
		assert.Error(t, err, assert.AnError)
	})
}
