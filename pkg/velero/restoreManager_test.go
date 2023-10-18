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
		expectedVeleroRestore := &velerov1.Restore{ObjectMeta: metav1.ObjectMeta{Name: "restore", Namespace: testNamespace}, Spec: velerov1.RestoreSpec{BackupName: "backup", ExistingResourcePolicy: velerov1.PolicyTypeUpdate}}

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
		runVeleroStatusPahseFailureTest(t, velerov1.RestorePhaseFailed)
	})

	t.Run("should return error on velero partially failure", func(t *testing.T) {
		runVeleroStatusPahseFailureTest(t, velerov1.RestorePhasePartiallyFailed)
	})

	t.Run("should return error on velero plugin operation partially failure", func(t *testing.T) {
		runVeleroStatusPahseFailureTest(t, velerov1.RestorePhaseWaitingForPluginOperationsPartiallyFailed)
	})

	t.Run("should return error on velero validation failure", func(t *testing.T) {
		runVeleroStatusPahseFailureTest(t, velerov1.RestorePhaseFailedValidation)
	})
}

func runVeleroStatusPahseFailureTest(t *testing.T, phase velerov1.RestorePhase) {
	// given
	restore := &v1.Restore{ObjectMeta: metav1.ObjectMeta{Name: "restore", Namespace: testNamespace}, Spec: v1.RestoreSpec{BackupName: "backup"}}
	expectedVeleroRestore := &velerov1.Restore{ObjectMeta: metav1.ObjectMeta{Name: "restore", Namespace: testNamespace}, Spec: velerov1.RestoreSpec{BackupName: "backup", ExistingResourcePolicy: velerov1.PolicyTypeUpdate}}

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
