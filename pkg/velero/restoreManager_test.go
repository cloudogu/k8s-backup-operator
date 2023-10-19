package velero

import (
	"fmt"
	v1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"testing"
	"time"
)

func TestNewDefaultRestoreManager(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// when
		newDefaultRestoreManager := NewDefaultRestoreManager(nil, nil)

		// then
		require.NotNil(t, newDefaultRestoreManager)
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

		veleroRestoreClientMock := newMockVeleroRestoreInterface(t)
		veleroInterfaceMock := newMockVeleroInterface(t)
		veleroInterfaceMock.EXPECT().Restores(testNamespace).Return(veleroRestoreClientMock)
		veleroClientMock := newMockVeleroClientSet(t)
		veleroClientMock.EXPECT().VeleroV1().Return(veleroInterfaceMock)
		veleroRestoreClientMock.EXPECT().Create(testCtx, expectedVeleroRestore, metav1.CreateOptions{}).Return(expectedVeleroRestore, nil)

		watchMock := newMockEcosystemWatch(t)
		channel := make(chan watch.Event)
		watchMock.EXPECT().ResultChan().Return(channel)
		watchMock.EXPECT().Stop().Run(func() {
			close(channel)
		})
		veleroRestoreClientMock.EXPECT().Watch(testCtx, metav1.ListOptions{FieldSelector: "metadata.name=restore"}).Return(watchMock, nil)

		timer := time.NewTimer(time.Second)
		go func() {
			<-timer.C
			expectedVeleroRestore.Status.Phase = velerov1.RestorePhaseCompleted
			channel <- watch.Event{Type: watch.Modified, Object: expectedVeleroRestore}
		}()

		sut := &defaultRestoreManager{veleroClientSet: veleroClientMock, recorder: recorderMock}

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

		veleroRestoreClientMock := newMockVeleroRestoreInterface(t)
		veleroInterfaceMock := newMockVeleroInterface(t)
		veleroInterfaceMock.EXPECT().Restores(testNamespace).Return(veleroRestoreClientMock)
		veleroClientMock := newMockVeleroClientSet(t)
		veleroClientMock.EXPECT().VeleroV1().Return(veleroInterfaceMock)
		veleroRestoreClientMock.EXPECT().Create(testCtx, expectedVeleroRestore, metav1.CreateOptions{}).Return(nil, assert.AnError)

		sut := &defaultRestoreManager{veleroClientSet: veleroClientMock, recorder: recorderMock}

		// when
		err := sut.CreateRestore(testCtx, restore)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to create velero restore [restore]")
	})

	t.Run("should return error on create velero restore wach error", func(t *testing.T) {
		// given
		restore := &v1.Restore{ObjectMeta: metav1.ObjectMeta{Name: "restore", Namespace: testNamespace}, Spec: v1.RestoreSpec{BackupName: "backup"}}
		expectedVeleroRestore := getExpectedVeleroRestore(restore)

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(restore, corev1.EventTypeNormal, "Creation", "Using velero as restore provider")
		recorderMock.EXPECT().Event(restore, corev1.EventTypeWarning, "ErrCreation", "failed to create velero restore watch: assert.AnError general error for testing")

		veleroRestoreClientMock := newMockVeleroRestoreInterface(t)
		veleroInterfaceMock := newMockVeleroInterface(t)
		veleroInterfaceMock.EXPECT().Restores(testNamespace).Return(veleroRestoreClientMock)
		veleroClientMock := newMockVeleroClientSet(t)
		veleroClientMock.EXPECT().VeleroV1().Return(veleroInterfaceMock)
		veleroRestoreClientMock.EXPECT().Create(testCtx, expectedVeleroRestore, metav1.CreateOptions{}).Return(expectedVeleroRestore, nil)
		veleroRestoreClientMock.EXPECT().Watch(testCtx, metav1.ListOptions{FieldSelector: "metadata.name=restore"}).Return(nil, assert.AnError)

		sut := &defaultRestoreManager{veleroClientSet: veleroClientMock, recorder: recorderMock}

		// when
		err := sut.CreateRestore(testCtx, restore)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to create velero restore watch")
	})

	t.Run("should return error on wrong event object", func(t *testing.T) {
		// given
		restore := &v1.Restore{ObjectMeta: metav1.ObjectMeta{Name: "restore", Namespace: testNamespace}, Spec: v1.RestoreSpec{BackupName: "backup"}}
		expectedVeleroRestore := getExpectedVeleroRestore(restore)

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(restore, corev1.EventTypeNormal, "Creation", "Using velero as restore provider")
		recorderMock.EXPECT().Event(restore, corev1.EventTypeWarning, "ErrCreation", "got event with wrong object type when watching velero restore")

		veleroRestoreClientMock := newMockVeleroRestoreInterface(t)
		veleroInterfaceMock := newMockVeleroInterface(t)
		veleroInterfaceMock.EXPECT().Restores(testNamespace).Return(veleroRestoreClientMock)
		veleroClientMock := newMockVeleroClientSet(t)
		veleroClientMock.EXPECT().VeleroV1().Return(veleroInterfaceMock)
		veleroRestoreClientMock.EXPECT().Create(testCtx, expectedVeleroRestore, metav1.CreateOptions{}).Return(expectedVeleroRestore, nil)

		watchMock := newMockEcosystemWatch(t)
		channel := make(chan watch.Event)
		watchMock.EXPECT().ResultChan().Return(channel)
		watchMock.EXPECT().Stop().Run(func() {
			close(channel)
		})
		veleroRestoreClientMock.EXPECT().Watch(testCtx, metav1.ListOptions{FieldSelector: "metadata.name=restore"}).Return(watchMock, nil)

		timer := time.NewTimer(time.Second)
		go func() {
			<-timer.C
			channel <- watch.Event{Type: watch.Modified, Object: &v1.Restore{}}
		}()

		sut := &defaultRestoreManager{veleroClientSet: veleroClientMock, recorder: recorderMock}

		// when
		err := sut.CreateRestore(testCtx, restore)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "got event with wrong object type when watching velero restore")
	})

	t.Run("should return error on delete event", func(t *testing.T) {
		// given
		restore := &v1.Restore{ObjectMeta: metav1.ObjectMeta{Name: "restore", Namespace: testNamespace}, Spec: v1.RestoreSpec{BackupName: "backup"}}
		expectedVeleroRestore := getExpectedVeleroRestore(restore)

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(restore, corev1.EventTypeNormal, "Creation", "Using velero as restore provider")
		recorderMock.EXPECT().Event(restore, corev1.EventTypeWarning, "ErrCreation", "failed to complete velero restore [restore]: the restore got deleted")

		veleroRestoreClientMock := newMockVeleroRestoreInterface(t)
		veleroInterfaceMock := newMockVeleroInterface(t)
		veleroInterfaceMock.EXPECT().Restores(testNamespace).Return(veleroRestoreClientMock)
		veleroClientMock := newMockVeleroClientSet(t)
		veleroClientMock.EXPECT().VeleroV1().Return(veleroInterfaceMock)
		veleroRestoreClientMock.EXPECT().Create(testCtx, expectedVeleroRestore, metav1.CreateOptions{}).Return(expectedVeleroRestore, nil)

		watchMock := newMockEcosystemWatch(t)
		channel := make(chan watch.Event)
		watchMock.EXPECT().ResultChan().Return(channel)
		watchMock.EXPECT().Stop().Run(func() {
			close(channel)
		})
		veleroRestoreClientMock.EXPECT().Watch(testCtx, metav1.ListOptions{FieldSelector: "metadata.name=restore"}).Return(watchMock, nil)

		timer := time.NewTimer(time.Second)
		go func() {
			<-timer.C
			channel <- watch.Event{Type: watch.Deleted, Object: expectedVeleroRestore}
		}()

		sut := &defaultRestoreManager{veleroClientSet: veleroClientMock, recorder: recorderMock}

		// when
		err := sut.CreateRestore(testCtx, restore)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to complete velero restore [restore]: the restore got deleted")
	})
}

func getExpectedVeleroRestore(restore *v1.Restore) *velerov1.Restore {
	return &velerov1.Restore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      restore.Name,
			Namespace: restore.Namespace,
		},
		Spec: velerov1.RestoreSpec{
			BackupName:             restore.Spec.BackupName,
			ExistingResourcePolicy: velerov1.PolicyTypeUpdate,
			RestoreStatus:          &velerov1.RestoreStatusSpec{IncludedResources: []string{"*"}},
			LabelSelector: &metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{
					{Key: "app.kubernetes.io/part-of", Operator: metav1.LabelSelectorOpNotIn, Values: []string{"k8s-backup-operator"}}}}}}
}

func runVeleroStatusPhaseFailureTest(t *testing.T, phase velerov1.RestorePhase) {
	// given
	restore := &v1.Restore{ObjectMeta: metav1.ObjectMeta{Name: "restore", Namespace: testNamespace}, Spec: v1.RestoreSpec{BackupName: "backup"}}
	expectedVeleroRestore := getExpectedVeleroRestore(restore)

	recorderMock := newMockEventRecorder(t)
	recorderMock.EXPECT().Event(restore, corev1.EventTypeNormal, "Creation", "Using velero as restore provider")
	recorderMock.EXPECT().Event(restore, corev1.EventTypeWarning, "ErrCreation", fmt.Sprintf("failed to complete velero restore [restore]: has status phase [%s]", phase))

	veleroRestoreClientMock := newMockVeleroRestoreInterface(t)
	veleroInterfaceMock := newMockVeleroInterface(t)
	veleroInterfaceMock.EXPECT().Restores(testNamespace).Return(veleroRestoreClientMock)
	veleroClientMock := newMockVeleroClientSet(t)
	veleroClientMock.EXPECT().VeleroV1().Return(veleroInterfaceMock)
	veleroRestoreClientMock.EXPECT().Create(testCtx, expectedVeleroRestore, metav1.CreateOptions{}).Return(expectedVeleroRestore, nil)

	watchMock := newMockEcosystemWatch(t)
	channel := make(chan watch.Event)
	watchMock.EXPECT().ResultChan().Return(channel)
	watchMock.EXPECT().Stop().Run(func() {
		close(channel)
	})
	veleroRestoreClientMock.EXPECT().Watch(testCtx, metav1.ListOptions{FieldSelector: "metadata.name=restore"}).Return(watchMock, nil)

	timer := time.NewTimer(time.Second)
	go func() {
		<-timer.C
		expectedVeleroRestore.Status.Phase = phase
		channel <- watch.Event{Type: watch.Modified, Object: expectedVeleroRestore}
	}()

	sut := &defaultRestoreManager{veleroClientSet: veleroClientMock, recorder: recorderMock}

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

		veleroRestoreClientMock := newMockVeleroRestoreInterface(t)
		veleroInterfaceMock := newMockVeleroInterface(t)
		veleroInterfaceMock.EXPECT().Restores(testNamespace).Return(veleroRestoreClientMock)
		veleroClientMock := newMockVeleroClientSet(t)
		veleroClientMock.EXPECT().VeleroV1().Return(veleroInterfaceMock)
		veleroRestoreClientMock.EXPECT().Delete(testCtx, restore.Name, metav1.DeleteOptions{}).Return(nil)

		sut := &defaultRestoreManager{veleroClientSet: veleroClientMock, recorder: recorderMock}

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

		veleroRestoreClientMock := newMockVeleroRestoreInterface(t)
		veleroInterfaceMock := newMockVeleroInterface(t)
		veleroInterfaceMock.EXPECT().Restores(testNamespace).Return(veleroRestoreClientMock)
		veleroClientMock := newMockVeleroClientSet(t)
		veleroClientMock.EXPECT().VeleroV1().Return(veleroInterfaceMock)
		veleroRestoreClientMock.EXPECT().Delete(testCtx, restore.Name, metav1.DeleteOptions{}).Return(assert.AnError)

		sut := &defaultRestoreManager{veleroClientSet: veleroClientMock, recorder: recorderMock}

		// when
		err := sut.DeleteRestore(testCtx, restore)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to delete velero restore [restore]")
		assert.Error(t, err, assert.AnError)
	})
}
