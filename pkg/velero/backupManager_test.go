package velero

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/watch"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"

	backupv1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
)

var testCtx = context.TODO()

const testNamespace = "test-namespace"

func Test_backupManager_CreateBackup(t *testing.T) {
	t.Run("should fail to create velero backup", func(t *testing.T) {
		// given
		testBackup := &backupv1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "testBackup", Namespace: testNamespace}}

		mockRecorder := newMockEventRecorder(t)
		mockRecorder.EXPECT().Event(testBackup, "Normal", "Creation", "Using velero as backup provider")
		mockRecorder.EXPECT().Event(testBackup, "Warning", "ErrCreation", "failed to apply velero backup 'test-namespace/testBackup' to cluster: assert.AnError general error for testing")

		volumeFsBackup := false
		expectedVeleroBackup := &velerov1.Backup{
			ObjectMeta: metav1.ObjectMeta{Name: "testBackup", Namespace: testNamespace},
			Spec: velerov1.BackupSpec{
				TTL:                      metav1.Duration{Duration: 87660 * time.Hour},
				IncludedNamespaces:       []string{testNamespace},
				StorageLocation:          "default",
				DefaultVolumesToFsBackup: &volumeFsBackup,
			},
		}
		mockVeleroBackupClient := newMockVeleroBackupInterface(t)
		mockVeleroBackupClient.EXPECT().Create(testCtx, expectedVeleroBackup, metav1.CreateOptions{}).Return(nil, assert.AnError)
		mockVeleroInterface := newMockVeleroInterface(t)
		mockVeleroInterface.EXPECT().Backups(testNamespace).Return(mockVeleroBackupClient)
		mockVeleroClient := newMockVeleroClientSet(t)
		mockVeleroClient.EXPECT().VeleroV1().Return(mockVeleroInterface)

		sut := &defaultBackupManager{
			recorder:        mockRecorder,
			veleroClientSet: mockVeleroClient,
		}

		// when
		err := sut.CreateBackup(testCtx, testBackup)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to apply velero backup 'test-namespace/testBackup' to cluster")
	})
	t.Run("should fail to create watch for velero backup", func(t *testing.T) {
		// given
		testBackup := &backupv1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "testBackup", Namespace: testNamespace}}

		mockRecorder := newMockEventRecorder(t)
		mockRecorder.EXPECT().Event(testBackup, "Normal", "Creation", "Using velero as backup provider")
		mockRecorder.EXPECT().Event(testBackup, "Warning", "ErrCreation", "failed to create watch for velero backup 'test-namespace/testBackup': assert.AnError general error for testing")

		volumeFsBackup := false
		expectedVeleroBackup := &velerov1.Backup{
			ObjectMeta: metav1.ObjectMeta{Name: "testBackup", Namespace: testNamespace},
			Spec: velerov1.BackupSpec{
				TTL:                      metav1.Duration{Duration: 87660 * time.Hour},
				IncludedNamespaces:       []string{testNamespace},
				StorageLocation:          "default",
				DefaultVolumesToFsBackup: &volumeFsBackup,
			},
		}
		mockBackupInterface := newMockVeleroBackupInterface(t)
		mockBackupInterface.EXPECT().Create(testCtx, expectedVeleroBackup, metav1.CreateOptions{}).Return(expectedVeleroBackup, nil)
		mockBackupInterface.EXPECT().Watch(testCtx, metav1.ListOptions{FieldSelector: testBackup.GetFieldSelectorWithName()}).Return(nil, assert.AnError)
		mockVeleroInterface := newMockVeleroInterface(t)
		mockVeleroInterface.EXPECT().Backups(testNamespace).Return(mockBackupInterface)
		mockVeleroClient := newMockVeleroClientSet(t)
		mockVeleroClient.EXPECT().VeleroV1().Return(mockVeleroInterface)

		sut := &defaultBackupManager{
			recorder:        mockRecorder,
			veleroClientSet: mockVeleroClient,
		}

		// when
		err := sut.CreateBackup(testCtx, testBackup)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to create watch for velero backup 'test-namespace/testBackup'")
	})
	t.Run("should fail because velero backup got deleted", func(t *testing.T) {
		// given
		testBackup := &backupv1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "testBackup", Namespace: testNamespace}}

		mockRecorder := newMockEventRecorder(t)
		mockRecorder.EXPECT().Event(testBackup, "Normal", "Creation", "Using velero as backup provider")
		mockRecorder.EXPECT().Event(testBackup, "Warning", "ErrCreation", "failed to complete velero backup 'test-namespace/testBackup': the backup got deleted")

		volumeFsBackup := false
		expectedVeleroBackup := &velerov1.Backup{
			ObjectMeta: metav1.ObjectMeta{Name: "testBackup", Namespace: testNamespace},
			Spec: velerov1.BackupSpec{
				TTL:                      metav1.Duration{Duration: 87660 * time.Hour},
				IncludedNamespaces:       []string{testNamespace},
				StorageLocation:          "default",
				DefaultVolumesToFsBackup: &volumeFsBackup,
			},
		}

		resultChan := make(chan watch.Event)
		mockWatcher := newMockEcosystemWatch(t)
		mockWatcher.EXPECT().ResultChan().Return(resultChan)
		mockWatcher.EXPECT().Stop().Run(func() {
			close(resultChan)
		})
		mockBackupInterface := newMockVeleroBackupInterface(t)
		mockBackupInterface.EXPECT().Create(testCtx, expectedVeleroBackup, metav1.CreateOptions{}).Return(expectedVeleroBackup, nil)
		mockBackupInterface.EXPECT().Watch(testCtx, metav1.ListOptions{FieldSelector: testBackup.GetFieldSelectorWithName()}).Return(mockWatcher, nil)
		mockVeleroInterface := newMockVeleroInterface(t)
		mockVeleroInterface.EXPECT().Backups(testNamespace).Return(mockBackupInterface)
		mockVeleroClient := newMockVeleroClientSet(t)
		mockVeleroClient.EXPECT().VeleroV1().Return(mockVeleroInterface)

		sut := &defaultBackupManager{
			recorder:        mockRecorder,
			veleroClientSet: mockVeleroClient,
		}

		go func() {
			// has to be run in goroutine to work
			resultChan <- watch.Event{
				Type: watch.Deleted,
				Object: &velerov1.Backup{
					ObjectMeta: metav1.ObjectMeta{Name: "testBackup", Namespace: testNamespace},
				},
			}
		}()

		// when
		err := sut.CreateBackup(testCtx, testBackup)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to complete velero backup 'test-namespace/testBackup': the backup got deleted")
	})
	t.Run("should fail because velero backup failed validation", func(t *testing.T) {
		// given
		testBackup := &backupv1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "testBackup", Namespace: testNamespace}}

		mockRecorder := newMockEventRecorder(t)
		mockRecorder.EXPECT().Event(testBackup, "Normal", "Creation", "Using velero as backup provider")
		mockRecorder.EXPECT().Event(testBackup, "Warning", "ErrCreation", "failed to complete velero backup 'test-namespace/testBackup': has status phase 'FailedValidation'")

		volumeFsBackup := false
		expectedVeleroBackup := &velerov1.Backup{
			ObjectMeta: metav1.ObjectMeta{Name: "testBackup", Namespace: testNamespace},
			Spec: velerov1.BackupSpec{
				TTL:                      metav1.Duration{Duration: 87660 * time.Hour},
				IncludedNamespaces:       []string{testNamespace},
				StorageLocation:          "default",
				DefaultVolumesToFsBackup: &volumeFsBackup,
			},
		}

		resultChan := make(chan watch.Event)
		mockWatcher := newMockEcosystemWatch(t)
		mockWatcher.EXPECT().ResultChan().Return(resultChan)
		mockWatcher.EXPECT().Stop().Run(func() {
			close(resultChan)
		})
		mockBackupInterface := newMockVeleroBackupInterface(t)
		mockBackupInterface.EXPECT().Create(testCtx, expectedVeleroBackup, metav1.CreateOptions{}).Return(expectedVeleroBackup, nil)
		mockBackupInterface.EXPECT().Watch(testCtx, metav1.ListOptions{FieldSelector: testBackup.GetFieldSelectorWithName()}).Return(mockWatcher, nil)
		mockVeleroInterface := newMockVeleroInterface(t)
		mockVeleroInterface.EXPECT().Backups(testNamespace).Return(mockBackupInterface)
		mockVeleroClient := newMockVeleroClientSet(t)
		mockVeleroClient.EXPECT().VeleroV1().Return(mockVeleroInterface)

		sut := &defaultBackupManager{
			recorder:        mockRecorder,
			veleroClientSet: mockVeleroClient,
		}

		go func() {
			// has to be run in goroutine to work
			resultChan <- watch.Event{
				Type: watch.Modified,
				Object: &velerov1.Backup{
					ObjectMeta: metav1.ObjectMeta{Name: "testBackup", Namespace: testNamespace},
					Status:     velerov1.BackupStatus{Phase: velerov1.BackupPhaseFailedValidation},
				},
			}
		}()

		// when
		err := sut.CreateBackup(testCtx, testBackup)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to complete velero backup 'test-namespace/testBackup': has status phase 'FailedValidation'")
	})
	t.Run("should fail because velero backup has status phase deleting", func(t *testing.T) {
		// given
		testBackup := &backupv1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "testBackup", Namespace: testNamespace}}

		mockRecorder := newMockEventRecorder(t)
		mockRecorder.EXPECT().Event(testBackup, "Normal", "Creation", "Using velero as backup provider")
		mockRecorder.EXPECT().Event(testBackup, "Warning", "ErrCreation", "failed to complete velero backup 'test-namespace/testBackup': invalid status phase 'Deleting'")

		volumeFsBackup := false
		expectedVeleroBackup := &velerov1.Backup{
			ObjectMeta: metav1.ObjectMeta{Name: "testBackup", Namespace: testNamespace},
			Spec: velerov1.BackupSpec{
				TTL:                      metav1.Duration{Duration: 87660 * time.Hour},
				IncludedNamespaces:       []string{testNamespace},
				StorageLocation:          "default",
				DefaultVolumesToFsBackup: &volumeFsBackup,
			},
		}

		resultChan := make(chan watch.Event)
		mockWatcher := newMockEcosystemWatch(t)
		mockWatcher.EXPECT().ResultChan().Return(resultChan)
		mockWatcher.EXPECT().Stop().Run(func() {
			close(resultChan)
		})
		mockBackupInterface := newMockVeleroBackupInterface(t)
		mockBackupInterface.EXPECT().Create(testCtx, expectedVeleroBackup, metav1.CreateOptions{}).Return(expectedVeleroBackup, nil)
		mockBackupInterface.EXPECT().Watch(testCtx, metav1.ListOptions{FieldSelector: testBackup.GetFieldSelectorWithName()}).Return(mockWatcher, nil)
		mockVeleroInterface := newMockVeleroInterface(t)
		mockVeleroInterface.EXPECT().Backups(testNamespace).Return(mockBackupInterface)
		mockVeleroClient := newMockVeleroClientSet(t)
		mockVeleroClient.EXPECT().VeleroV1().Return(mockVeleroInterface)

		sut := &defaultBackupManager{
			recorder:        mockRecorder,
			veleroClientSet: mockVeleroClient,
		}

		go func() {
			// has to be run in goroutine to work
			resultChan <- watch.Event{
				Type: watch.Modified,
				Object: &velerov1.Backup{
					ObjectMeta: metav1.ObjectMeta{Name: "testBackup", Namespace: testNamespace},
					Status:     velerov1.BackupStatus{Phase: velerov1.BackupPhaseDeleting},
				},
			}
		}()

		// when
		err := sut.CreateBackup(testCtx, testBackup)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to complete velero backup 'test-namespace/testBackup': invalid status phase 'Deleting'")
	})
	t.Run("should succeed when velero backup is completed", func(t *testing.T) {
		// given
		testBackup := &backupv1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "testBackup", Namespace: testNamespace}}

		mockRecorder := newMockEventRecorder(t)
		mockRecorder.EXPECT().Event(testBackup, "Normal", "Creation", "Using velero as backup provider")
		mockRecorder.EXPECT().Eventf(testBackup, "Normal", "Creation", "Successfully completed velero backup '%s/%s'", testNamespace, "testBackup")

		volumeFsBackup := false
		expectedVeleroBackup := &velerov1.Backup{
			ObjectMeta: metav1.ObjectMeta{Name: "testBackup", Namespace: testNamespace},
			Spec: velerov1.BackupSpec{
				TTL:                      metav1.Duration{Duration: 87660 * time.Hour},
				IncludedNamespaces:       []string{testNamespace},
				StorageLocation:          "default",
				DefaultVolumesToFsBackup: &volumeFsBackup,
			},
		}

		resultChan := make(chan watch.Event)
		mockWatcher := newMockEcosystemWatch(t)
		mockWatcher.EXPECT().ResultChan().Return(resultChan)
		mockWatcher.EXPECT().Stop().Run(func() {
			close(resultChan)
		})
		mockBackupInterface := newMockVeleroBackupInterface(t)
		mockBackupInterface.EXPECT().Create(testCtx, expectedVeleroBackup, metav1.CreateOptions{}).Return(expectedVeleroBackup, nil)
		mockBackupInterface.EXPECT().Watch(testCtx, metav1.ListOptions{FieldSelector: testBackup.GetFieldSelectorWithName()}).Return(mockWatcher, nil)
		mockVeleroInterface := newMockVeleroInterface(t)
		mockVeleroInterface.EXPECT().Backups(testNamespace).Return(mockBackupInterface)
		mockVeleroClient := newMockVeleroClientSet(t)
		mockVeleroClient.EXPECT().VeleroV1().Return(mockVeleroInterface)

		sut := &defaultBackupManager{
			recorder:        mockRecorder,
			veleroClientSet: mockVeleroClient,
		}

		go func() {
			// has to be run in goroutine to work
			resultChan <- watch.Event{
				Type: watch.Modified,
				Object: &velerov1.Backup{
					ObjectMeta: metav1.ObjectMeta{Name: "testBackup", Namespace: testNamespace},
					Status:     velerov1.BackupStatus{Phase: velerov1.BackupPhaseCompleted},
				},
			}
		}()

		// when
		err := sut.CreateBackup(testCtx, testBackup)

		// then
		require.NoError(t, err)
	})
}

func Test_provider_DeleteBackup(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		backup := &backupv1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "backup", Namespace: testNamespace}}

		watchChannel := make(chan watch.Event)
		watchMock := newMockEcosystemWatch(t)
		watchMock.EXPECT().ResultChan().Return(watchChannel)
		watchMock.EXPECT().Stop()

		veleroBackupDeleteRequestClientMock := newMockVeleroDeleteBackupRequest(t)
		expectedRequest := getVeleroDeleteBackupRequest(backup.Name, testNamespace)
		expectedRequestProcessed := getVeleroDeleteBackupRequest(backup.Name, testNamespace)
		expectedRequestProcessed.Status.Phase = velerov1.DeleteBackupRequestPhaseProcessed
		veleroBackupDeleteRequestClientMock.EXPECT().Create(context.TODO(), expectedRequest, metav1.CreateOptions{}).Return(nil, nil)
		veleroBackupDeleteRequestClientMock.EXPECT().Watch(context.TODO(), metav1.ListOptions{FieldSelector: "metadata.name=backup", TimeoutSeconds: &deleteWaitTimeout}).Return(watchMock, nil)
		veleroBackupDeleteRequestClientMock.EXPECT().Get(context.TODO(), backup.Name, metav1.GetOptions{}).Return(expectedRequestProcessed, nil)

		veleroV1ClientMock := newMockVeleroInterface(t)
		veleroV1ClientMock.EXPECT().DeleteBackupRequests(testNamespace).Return(veleroBackupDeleteRequestClientMock)

		veleroClientSetMock := newMockVeleroClientSet(t)
		veleroClientSetMock.EXPECT().VeleroV1().Return(veleroV1ClientMock)

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(backup, corev1.EventTypeNormal, backupv1.ProviderDeleteEventReason, "Trigger velero provider to delete backup.")
		recorderMock.EXPECT().Event(backup, corev1.EventTypeNormal, backupv1.ProviderDeleteEventReason, "Provider delete request successful.")

		sut := defaultBackupManager{recorder: recorderMock, veleroClientSet: veleroClientSetMock}

		watchTimer := time.NewTimer(time.Second * 2)
		go func() {
			<-watchTimer.C
			event := watch.Event{Type: watch.Modified, Object: expectedRequestProcessed}
			watchChannel <- event
		}()

		// when
		err := sut.DeleteBackup(context.TODO(), backup)

		// then
		require.NoError(t, err)
	})

	t.Run("should return error on delete back request creation error", func(t *testing.T) {
		// given
		backup := &backupv1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "backup", Namespace: testNamespace}}

		veleroBackupDeleteRequestClientMock := newMockVeleroDeleteBackupRequest(t)
		veleroV1ClientMock := newMockVeleroInterface(t)
		veleroV1ClientMock.EXPECT().DeleteBackupRequests(testNamespace).Return(veleroBackupDeleteRequestClientMock)
		veleroClientSetMock := newMockVeleroClientSet(t)
		veleroClientSetMock.EXPECT().VeleroV1().Return(veleroV1ClientMock)

		expectedRequest := getVeleroDeleteBackupRequest(backup.Name, testNamespace)
		veleroBackupDeleteRequestClientMock.EXPECT().Create(context.TODO(), expectedRequest, metav1.CreateOptions{}).Return(nil, assert.AnError)

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(backup, corev1.EventTypeNormal, backupv1.ProviderDeleteEventReason, "Trigger velero provider to delete backup.")

		sut := defaultBackupManager{recorder: recorderMock, veleroClientSet: veleroClientSetMock}

		// when
		err := sut.DeleteBackup(context.TODO(), backup)

		// then
		require.Error(t, err)
		require.ErrorIs(t, err, assert.AnError)
		require.ErrorContains(t, err, "failed to create backup delete request backup")
	})

	t.Run("should return error on watch creation error", func(t *testing.T) {
		// given
		backup := &backupv1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "backup", Namespace: testNamespace}}

		veleroBackupDeleteRequestClientMock := newMockVeleroDeleteBackupRequest(t)
		veleroV1ClientMock := newMockVeleroInterface(t)
		veleroV1ClientMock.EXPECT().DeleteBackupRequests(testNamespace).Return(veleroBackupDeleteRequestClientMock)
		veleroClientSetMock := newMockVeleroClientSet(t)
		veleroClientSetMock.EXPECT().VeleroV1().Return(veleroV1ClientMock)

		expectedRequest := getVeleroDeleteBackupRequest(backup.Name, testNamespace)
		veleroBackupDeleteRequestClientMock.EXPECT().Create(context.TODO(), expectedRequest, metav1.CreateOptions{}).Return(nil, nil)
		veleroBackupDeleteRequestClientMock.EXPECT().Watch(context.TODO(), metav1.ListOptions{FieldSelector: "metadata.name=backup", TimeoutSeconds: &deleteWaitTimeout}).Return(nil, assert.AnError)

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(backup, corev1.EventTypeNormal, backupv1.ProviderDeleteEventReason, "Trigger velero provider to delete backup.")

		sut := defaultBackupManager{recorder: recorderMock, veleroClientSet: veleroClientSetMock}

		// when
		err := sut.DeleteBackup(context.TODO(), backup)

		// then
		require.Error(t, err)
		require.ErrorIs(t, err, assert.AnError)
		require.ErrorContains(t, err, "failed to create watch for delete backup request backup")
	})

	t.Run("should return error on get error after watch was successful", func(t *testing.T) {
		// given
		backup := &backupv1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "backup", Namespace: testNamespace}}

		watchChannel := make(chan watch.Event)
		watchMock := newMockEcosystemWatch(t)
		watchMock.EXPECT().ResultChan().Return(watchChannel)

		veleroBackupDeleteRequestClientMock := newMockVeleroDeleteBackupRequest(t)
		veleroV1ClientMock := newMockVeleroInterface(t)
		veleroV1ClientMock.EXPECT().DeleteBackupRequests(testNamespace).Return(veleroBackupDeleteRequestClientMock)
		veleroClientSetMock := newMockVeleroClientSet(t)
		veleroClientSetMock.EXPECT().VeleroV1().Return(veleroV1ClientMock)

		expectedRequest := getVeleroDeleteBackupRequest(backup.Name, testNamespace)
		expectedRequestProcessed := getVeleroDeleteBackupRequest(backup.Name, testNamespace)
		expectedRequestProcessed.Status.Phase = velerov1.DeleteBackupRequestPhaseProcessed
		veleroBackupDeleteRequestClientMock.EXPECT().Create(context.TODO(), expectedRequest, metav1.CreateOptions{}).Return(nil, nil)
		veleroBackupDeleteRequestClientMock.EXPECT().Watch(context.TODO(), metav1.ListOptions{FieldSelector: "metadata.name=backup", TimeoutSeconds: &deleteWaitTimeout}).Return(watchMock, nil)
		veleroBackupDeleteRequestClientMock.EXPECT().Get(context.TODO(), backup.Name, metav1.GetOptions{}).Return(nil, assert.AnError)

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(backup, corev1.EventTypeNormal, backupv1.ProviderDeleteEventReason, "Trigger velero provider to delete backup.")

		sut := defaultBackupManager{recorder: recorderMock, veleroClientSet: veleroClientSetMock}

		watchTimer := time.NewTimer(time.Second * 1)
		go func() {
			<-watchTimer.C
			close(watchChannel)
		}()

		// when
		err := sut.DeleteBackup(context.TODO(), backup)

		// then
		require.Error(t, err)
		require.ErrorIs(t, err, assert.AnError)
		require.ErrorContains(t, err, "failed to get delete backup request backup")
	})

	t.Run("should return error and clean up if request status is not processed after timeout", func(t *testing.T) {
		// given
		backup := &backupv1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "backup", Namespace: testNamespace}}

		watchChannel := make(chan watch.Event)
		watchMock := newMockEcosystemWatch(t)
		watchMock.EXPECT().ResultChan().Return(watchChannel)

		veleroBackupDeleteRequestClientMock := newMockVeleroDeleteBackupRequest(t)
		veleroV1ClientMock := newMockVeleroInterface(t)
		veleroV1ClientMock.EXPECT().DeleteBackupRequests(testNamespace).Return(veleroBackupDeleteRequestClientMock)
		veleroClientSetMock := newMockVeleroClientSet(t)
		veleroClientSetMock.EXPECT().VeleroV1().Return(veleroV1ClientMock)

		expectedRequest := getVeleroDeleteBackupRequest(backup.Name, testNamespace)
		expectedRequestProcessed := getVeleroDeleteBackupRequest(backup.Name, testNamespace)
		expectedRequestProcessed.Status.Phase = velerov1.DeleteBackupRequestPhaseProcessed
		veleroBackupDeleteRequestClientMock.EXPECT().Create(context.TODO(), expectedRequest, metav1.CreateOptions{}).Return(nil, nil)
		veleroBackupDeleteRequestClientMock.EXPECT().Watch(context.TODO(), metav1.ListOptions{FieldSelector: "metadata.name=backup", TimeoutSeconds: &deleteWaitTimeout}).Return(watchMock, nil)
		veleroBackupDeleteRequestClientMock.EXPECT().Get(context.TODO(), backup.Name, metav1.GetOptions{}).Return(expectedRequest, nil)
		veleroBackupDeleteRequestClientMock.EXPECT().Delete(context.TODO(), backup.Name, metav1.DeleteOptions{}).Return(nil)

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(backup, corev1.EventTypeNormal, backupv1.ProviderDeleteEventReason, "Trigger velero provider to delete backup.")
		recorderMock.EXPECT().Event(backup, corev1.EventTypeWarning, backupv1.ProviderDeleteEventReason, "Cleanup velero delete request.")

		sut := defaultBackupManager{recorder: recorderMock, veleroClientSet: veleroClientSetMock}

		watchTimer := time.NewTimer(time.Second * 1)
		go func() {
			<-watchTimer.C
			close(watchChannel)
		}()

		// when
		err := sut.DeleteBackup(context.TODO(), backup)

		// then
		require.Error(t, err)
		require.ErrorContains(t, err, "failed to delete backup backup: timout waiting for backup delete request backup")
	})

	t.Run("should return all errors in multierror and clean up if request status has errors", func(t *testing.T) {
		// given
		backup := &backupv1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "backup", Namespace: testNamespace}}

		watchChannel := make(chan watch.Event)
		watchMock := newMockEcosystemWatch(t)
		watchMock.EXPECT().ResultChan().Return(watchChannel)

		veleroBackupDeleteRequestClientMock := newMockVeleroDeleteBackupRequest(t)
		veleroV1ClientMock := newMockVeleroInterface(t)
		veleroV1ClientMock.EXPECT().DeleteBackupRequests(testNamespace).Return(veleroBackupDeleteRequestClientMock)
		veleroClientSetMock := newMockVeleroClientSet(t)
		veleroClientSetMock.EXPECT().VeleroV1().Return(veleroV1ClientMock)

		expectedRequest := getVeleroDeleteBackupRequest(backup.Name, testNamespace)
		expectedRequestProcessed := getVeleroDeleteBackupRequest(backup.Name, testNamespace)
		expectedRequestProcessed.Status.Phase = velerov1.DeleteBackupRequestPhaseProcessed
		expectedRequestProcessed.Status.Errors = []string{"error1", "error2"}
		veleroBackupDeleteRequestClientMock.EXPECT().Create(context.TODO(), expectedRequest, metav1.CreateOptions{}).Return(nil, nil)
		veleroBackupDeleteRequestClientMock.EXPECT().Watch(context.TODO(), metav1.ListOptions{FieldSelector: "metadata.name=backup", TimeoutSeconds: &deleteWaitTimeout}).Return(watchMock, nil)
		veleroBackupDeleteRequestClientMock.EXPECT().Get(context.TODO(), backup.Name, metav1.GetOptions{}).Return(expectedRequestProcessed, nil)
		veleroBackupDeleteRequestClientMock.EXPECT().Delete(context.TODO(), backup.Name, metav1.DeleteOptions{}).Return(nil)

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(backup, corev1.EventTypeNormal, backupv1.ProviderDeleteEventReason, "Trigger velero provider to delete backup.")
		recorderMock.EXPECT().Event(backup, corev1.EventTypeWarning, backupv1.ProviderDeleteEventReason, "Cleanup velero delete request.")
		recorderMock.EXPECT().Event(backup, corev1.EventTypeWarning, backupv1.ErrorOnProviderDeleteEventReason, "velero backup delete request error: error1\nvelero backup delete request error: error2")
		sut := defaultBackupManager{recorder: recorderMock, veleroClientSet: veleroClientSetMock}

		watchTimer := time.NewTimer(time.Second * 1)
		go func() {
			<-watchTimer.C
			close(watchChannel)
		}()

		// when
		err := sut.DeleteBackup(context.TODO(), backup)

		// then
		require.Error(t, err)
		require.ErrorContains(t, err, "failed to delete backup")
	})
}

func getVeleroDeleteBackupRequest(name, namespace string) *velerov1.DeleteBackupRequest {
	return &velerov1.DeleteBackupRequest{
		TypeMeta: metav1.TypeMeta{
			APIVersion: velerov1.SchemeGroupVersion.String(),
			Kind:       veleroBackupDeleteRequestKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Spec: velerov1.DeleteBackupRequestSpec{
			BackupName: name,
		},
	}
}

func Test_provider_cleanUpDeleteRequest(t *testing.T) {
	t.Run("should return no error on error delete", func(t *testing.T) {
		// given
		backup := &backupv1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "backup", Namespace: testNamespace}}
		request := getVeleroDeleteBackupRequest(backup.Name, backup.Namespace)

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(backup, corev1.EventTypeWarning, backupv1.ProviderDeleteEventReason, "Cleanup velero delete request.")
		recorderMock.EXPECT().Event(backup, corev1.EventTypeWarning, backupv1.ProviderDeleteEventReason, "Error cleanup velero delete request.")
		veleroClientSetMock := newMockVeleroClientSet(t)
		veleroV1ClientMock := newMockVeleroInterface(t)
		veleroClientSetMock.EXPECT().VeleroV1().Return(veleroV1ClientMock)
		veleroBackupDeleteRequestClientMock := newMockVeleroDeleteBackupRequest(t)
		veleroV1ClientMock.EXPECT().DeleteBackupRequests(testNamespace).Return(veleroBackupDeleteRequestClientMock)
		veleroBackupDeleteRequestClientMock.EXPECT().Delete(context.TODO(), request.Name, metav1.DeleteOptions{}).Return(assert.AnError)

		sut := defaultBackupManager{recorder: recorderMock, veleroClientSet: veleroClientSetMock}

		// when
		sut.cleanUpDeleteRequest(context.TODO(), backup, request)

	})
}
