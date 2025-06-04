package velero

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
			ObjectMeta: metav1.ObjectMeta{Name: "testBackup", Namespace: testNamespace, Labels: map[string]string{"app": "ces", "k8s.cloudogu.com/part-of": "backup"}},
			Spec: velerov1.BackupSpec{
				TTL:                      metav1.Duration{Duration: 87660 * time.Hour},
				IncludedNamespaces:       []string{testNamespace},
				StorageLocation:          "default",
				DefaultVolumesToFsBackup: &volumeFsBackup,
			},
		}
		mockK8sWatchClient := newMockK8sWatchClient(t)
		mockK8sWatchClient.EXPECT().Create(testCtx, expectedVeleroBackup).Return(assert.AnError)

		sut := &defaultBackupManager{
			recorder:  mockRecorder,
			k8sClient: mockK8sWatchClient,
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
			ObjectMeta: metav1.ObjectMeta{Name: "testBackup", Namespace: testNamespace, Labels: map[string]string{"app": "ces", "k8s.cloudogu.com/part-of": "backup"}},
			Spec: velerov1.BackupSpec{
				TTL:                      metav1.Duration{Duration: 87660 * time.Hour},
				IncludedNamespaces:       []string{testNamespace},
				StorageLocation:          "default",
				DefaultVolumesToFsBackup: &volumeFsBackup,
			},
		}

		mockK8sWatchClient := newMockK8sWatchClient(t)
		mockK8sWatchClient.EXPECT().Create(testCtx, expectedVeleroBackup).Return(nil)
		mockK8sWatchClient.EXPECT().Watch(testCtx, &velerov1.BackupList{},
			&client.ListOptions{FieldSelector: fields.ParseSelectorOrDie(testBackup.GetFieldSelectorWithName()), Namespace: testNamespace}).
			Return(nil, assert.AnError)

		sut := &defaultBackupManager{
			recorder:  mockRecorder,
			k8sClient: mockK8sWatchClient,
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
			ObjectMeta: metav1.ObjectMeta{Name: "testBackup", Namespace: testNamespace, Labels: map[string]string{"app": "ces", "k8s.cloudogu.com/part-of": "backup"}},
			Spec: velerov1.BackupSpec{
				TTL:                      metav1.Duration{Duration: 87660 * time.Hour},
				IncludedNamespaces:       []string{testNamespace},
				StorageLocation:          "default",
				DefaultVolumesToFsBackup: &volumeFsBackup,
			},
		}

		resultChan := make(chan watch.Event)
		mockWatcher := newMockWatchInterface(t)
		mockWatcher.EXPECT().ResultChan().Return(resultChan)
		mockWatcher.EXPECT().Stop().Run(func() {
			close(resultChan)
		})
		mockK8sWatchClient := newMockK8sWatchClient(t)
		mockK8sWatchClient.EXPECT().Create(testCtx, expectedVeleroBackup).Return(nil)
		mockK8sWatchClient.EXPECT().Watch(testCtx, &velerov1.BackupList{},
			&client.ListOptions{FieldSelector: fields.ParseSelectorOrDie(testBackup.GetFieldSelectorWithName()), Namespace: testNamespace}).
			Return(mockWatcher, nil)

		sut := &defaultBackupManager{
			recorder:  mockRecorder,
			k8sClient: mockK8sWatchClient,
		}

		go func() {
			// has to be run in goroutine to work
			resultChan <- watch.Event{
				Type: watch.Deleted,
				Object: &velerov1.Backup{
					ObjectMeta: metav1.ObjectMeta{Name: "testBackup", Namespace: testNamespace, Labels: map[string]string{"app": "ces", "k8s.cloudogu.com/part-of": "backup"}},
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
			ObjectMeta: metav1.ObjectMeta{Name: "testBackup", Namespace: testNamespace, Labels: map[string]string{"app": "ces", "k8s.cloudogu.com/part-of": "backup"}},
			Spec: velerov1.BackupSpec{
				TTL:                      metav1.Duration{Duration: 87660 * time.Hour},
				IncludedNamespaces:       []string{testNamespace},
				StorageLocation:          "default",
				DefaultVolumesToFsBackup: &volumeFsBackup,
			},
		}

		resultChan := make(chan watch.Event)
		mockWatcher := newMockWatchInterface(t)
		mockWatcher.EXPECT().ResultChan().Return(resultChan)
		mockWatcher.EXPECT().Stop().Run(func() {
			close(resultChan)
		})
		mockK8sWatchClient := newMockK8sWatchClient(t)
		mockK8sWatchClient.EXPECT().Create(testCtx, expectedVeleroBackup).Return(nil)
		mockK8sWatchClient.EXPECT().Watch(testCtx, &velerov1.BackupList{},
			&client.ListOptions{FieldSelector: fields.ParseSelectorOrDie(testBackup.GetFieldSelectorWithName()), Namespace: testNamespace}).
			Return(mockWatcher, nil)

		sut := &defaultBackupManager{
			recorder:  mockRecorder,
			k8sClient: mockK8sWatchClient,
		}

		go func() {
			// has to be run in goroutine to work
			resultChan <- watch.Event{
				Type: watch.Modified,
				Object: &velerov1.Backup{
					ObjectMeta: metav1.ObjectMeta{Name: "testBackup", Namespace: testNamespace, Labels: map[string]string{"app": "ces", "k8s.cloudogu.com/part-of": "backup"}},
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
			ObjectMeta: metav1.ObjectMeta{Name: "testBackup", Namespace: testNamespace, Labels: map[string]string{"app": "ces", "k8s.cloudogu.com/part-of": "backup"}},
			Spec: velerov1.BackupSpec{
				TTL:                      metav1.Duration{Duration: 87660 * time.Hour},
				IncludedNamespaces:       []string{testNamespace},
				StorageLocation:          "default",
				DefaultVolumesToFsBackup: &volumeFsBackup,
			},
		}

		resultChan := make(chan watch.Event)
		mockWatcher := newMockWatchInterface(t)
		mockWatcher.EXPECT().ResultChan().Return(resultChan)
		mockWatcher.EXPECT().Stop().Run(func() {
			close(resultChan)
		})
		mockK8sWatchClient := newMockK8sWatchClient(t)
		mockK8sWatchClient.EXPECT().Create(testCtx, expectedVeleroBackup).Return(nil)
		mockK8sWatchClient.EXPECT().Watch(testCtx, &velerov1.BackupList{},
			&client.ListOptions{FieldSelector: fields.ParseSelectorOrDie(testBackup.GetFieldSelectorWithName()), Namespace: testNamespace}).
			Return(mockWatcher, nil)

		sut := &defaultBackupManager{
			recorder:  mockRecorder,
			k8sClient: mockK8sWatchClient,
		}

		go func() {
			// has to be run in goroutine to work
			resultChan <- watch.Event{
				Type: watch.Modified,
				Object: &velerov1.Backup{
					ObjectMeta: metav1.ObjectMeta{Name: "testBackup", Namespace: testNamespace, Labels: map[string]string{"app": "ces", "k8s.cloudogu.com/part-of": "backup"}},
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
			ObjectMeta: metav1.ObjectMeta{Name: "testBackup", Namespace: testNamespace, Labels: map[string]string{"app": "ces", "k8s.cloudogu.com/part-of": "backup"}},
			Spec: velerov1.BackupSpec{
				TTL:                      metav1.Duration{Duration: 87660 * time.Hour},
				IncludedNamespaces:       []string{testNamespace},
				StorageLocation:          "default",
				DefaultVolumesToFsBackup: &volumeFsBackup,
			},
		}

		resultChan := make(chan watch.Event)
		mockWatcher := newMockWatchInterface(t)
		mockWatcher.EXPECT().ResultChan().Return(resultChan)
		mockWatcher.EXPECT().Stop().Run(func() {
			close(resultChan)
		})
		mockK8sWatchClient := newMockK8sWatchClient(t)
		mockK8sWatchClient.EXPECT().Create(testCtx, expectedVeleroBackup).Return(nil)
		mockK8sWatchClient.EXPECT().Watch(testCtx, &velerov1.BackupList{},
			&client.ListOptions{FieldSelector: fields.ParseSelectorOrDie(testBackup.GetFieldSelectorWithName()), Namespace: testNamespace}).
			Return(mockWatcher, nil)

		sut := &defaultBackupManager{
			recorder:  mockRecorder,
			k8sClient: mockK8sWatchClient,
		}

		go func() {
			// has to be run in goroutine to work
			resultChan <- watch.Event{
				Type: watch.Modified,
				Object: &velerov1.Backup{
					ObjectMeta: metav1.ObjectMeta{Name: "testBackup", Namespace: testNamespace, Labels: map[string]string{"app": "ces", "k8s.cloudogu.com/part-of": "backup"}},
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
		watchMock := newMockWatchInterface(t)
		watchMock.EXPECT().ResultChan().Return(watchChannel)
		watchMock.EXPECT().Stop()

		expectedRequest := getVeleroDeleteBackupRequest(backup.Name, testNamespace)
		expectedRequestProcessed := getVeleroDeleteBackupRequest(backup.Name, testNamespace)
		expectedRequestProcessed.Status.Phase = velerov1.DeleteBackupRequestPhaseProcessed

		mockK8sWatchClient := newMockK8sWatchClient(t)
		mockK8sWatchClient.EXPECT().Create(testCtx, expectedRequest).Return(nil)
		mockK8sWatchClient.EXPECT().Watch(testCtx, &velerov1.BackupList{},
			&client.ListOptions{FieldSelector: fields.ParseSelectorOrDie("metadata.name=backup"),
				Namespace: testNamespace, Raw: &metav1.ListOptions{TimeoutSeconds: &deleteWaitTimeout}}).
			Return(watchMock, nil)
		mockK8sWatchClient.EXPECT().Get(testCtx, backup.GetNamespacedName(), expectedRequest).Run(func(ctx context.Context, key types.NamespacedName, obj client.Object, opts ...client.GetOption) {
			*obj.(*velerov1.DeleteBackupRequest) = *expectedRequestProcessed
		}).Return(nil)

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(backup, corev1.EventTypeNormal, backupv1.ProviderDeleteEventReason, "Trigger velero provider to delete backup.")
		recorderMock.EXPECT().Event(backup, corev1.EventTypeNormal, backupv1.ProviderDeleteEventReason, "Provider delete request successful.")

		sut := defaultBackupManager{recorder: recorderMock, k8sClient: mockK8sWatchClient}

		watchTimer := time.NewTimer(time.Second * 2)
		go func() {
			<-watchTimer.C
			event := watch.Event{Type: watch.Modified, Object: expectedRequestProcessed}
			watchChannel <- event
		}()

		// when
		err := sut.DeleteBackup(testCtx, backup)

		// then
		require.NoError(t, err)
	})

	t.Run("should return error on delete backup request creation error", func(t *testing.T) {
		// given
		backup := &backupv1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "backup", Namespace: testNamespace}}

		expectedRequest := getVeleroDeleteBackupRequest(backup.Name, testNamespace)

		mockK8sWatchClient := newMockK8sWatchClient(t)
		mockK8sWatchClient.EXPECT().Create(testCtx, expectedRequest).Return(assert.AnError)

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(backup, corev1.EventTypeNormal, backupv1.ProviderDeleteEventReason, "Trigger velero provider to delete backup.")

		sut := defaultBackupManager{recorder: recorderMock, k8sClient: mockK8sWatchClient}

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

		expectedRequest := getVeleroDeleteBackupRequest(backup.Name, testNamespace)

		mockK8sWatchClient := newMockK8sWatchClient(t)
		mockK8sWatchClient.EXPECT().Create(testCtx, expectedRequest).Return(nil)
		mockK8sWatchClient.EXPECT().Watch(testCtx, &velerov1.BackupList{},
			&client.ListOptions{FieldSelector: fields.ParseSelectorOrDie("metadata.name=backup"),
				Namespace: testNamespace, Raw: &metav1.ListOptions{TimeoutSeconds: &deleteWaitTimeout}}).
			Return(nil, assert.AnError)

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(backup, corev1.EventTypeNormal, backupv1.ProviderDeleteEventReason, "Trigger velero provider to delete backup.")

		sut := defaultBackupManager{recorder: recorderMock, k8sClient: mockK8sWatchClient}

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
		watchMock := newMockWatchInterface(t)
		watchMock.EXPECT().ResultChan().Return(watchChannel)

		expectedRequest := getVeleroDeleteBackupRequest(backup.Name, testNamespace)
		expectedRequestProcessed := getVeleroDeleteBackupRequest(backup.Name, testNamespace)
		expectedRequestProcessed.Status.Phase = velerov1.DeleteBackupRequestPhaseProcessed

		mockK8sWatchClient := newMockK8sWatchClient(t)
		mockK8sWatchClient.EXPECT().Create(testCtx, expectedRequest).Return(nil)
		mockK8sWatchClient.EXPECT().Watch(testCtx, &velerov1.BackupList{},
			&client.ListOptions{FieldSelector: fields.ParseSelectorOrDie("metadata.name=backup"),
				Namespace: testNamespace, Raw: &metav1.ListOptions{TimeoutSeconds: &deleteWaitTimeout}}).
			Return(watchMock, nil)
		mockK8sWatchClient.EXPECT().Get(testCtx, backup.GetNamespacedName(), expectedRequest).Return(assert.AnError)

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(backup, corev1.EventTypeNormal, backupv1.ProviderDeleteEventReason, "Trigger velero provider to delete backup.")

		sut := defaultBackupManager{recorder: recorderMock, k8sClient: mockK8sWatchClient}

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
		watchMock := newMockWatchInterface(t)
		watchMock.EXPECT().ResultChan().Return(watchChannel)

		expectedRequest := getVeleroDeleteBackupRequest(backup.Name, testNamespace)

		mockK8sWatchClient := newMockK8sWatchClient(t)
		mockK8sWatchClient.EXPECT().Create(testCtx, expectedRequest).Return(nil)
		mockK8sWatchClient.EXPECT().Watch(testCtx, &velerov1.BackupList{},
			&client.ListOptions{FieldSelector: fields.ParseSelectorOrDie("metadata.name=backup"),
				Namespace: testNamespace, Raw: &metav1.ListOptions{TimeoutSeconds: &deleteWaitTimeout}}).
			Return(watchMock, nil)
		mockK8sWatchClient.EXPECT().Get(testCtx, backup.GetNamespacedName(), expectedRequest).Return(nil)
		mockK8sWatchClient.EXPECT().Delete(testCtx, expectedRequest).Return(nil)

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(backup, corev1.EventTypeNormal, backupv1.ProviderDeleteEventReason, "Trigger velero provider to delete backup.")
		recorderMock.EXPECT().Event(backup, corev1.EventTypeWarning, backupv1.ProviderDeleteEventReason, "Cleanup velero delete request.")

		sut := defaultBackupManager{recorder: recorderMock, k8sClient: mockK8sWatchClient}

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
		watchMock := newMockWatchInterface(t)
		watchMock.EXPECT().ResultChan().Return(watchChannel)

		expectedRequest := getVeleroDeleteBackupRequest(backup.Name, testNamespace)
		expectedRequestProcessed := getVeleroDeleteBackupRequest(backup.Name, testNamespace)
		expectedRequestProcessed.Status.Phase = velerov1.DeleteBackupRequestPhaseProcessed
		expectedRequestProcessed.Status.Errors = []string{"error1", "error2"}

		mockK8sWatchClient := newMockK8sWatchClient(t)
		mockK8sWatchClient.EXPECT().Create(testCtx, expectedRequest).Return(nil)
		mockK8sWatchClient.EXPECT().Watch(testCtx, &velerov1.BackupList{},
			&client.ListOptions{FieldSelector: fields.ParseSelectorOrDie("metadata.name=backup"),
				Namespace: testNamespace, Raw: &metav1.ListOptions{TimeoutSeconds: &deleteWaitTimeout}}).
			Return(watchMock, nil)
		mockK8sWatchClient.EXPECT().Get(testCtx, backup.GetNamespacedName(), expectedRequest).Run(func(ctx context.Context, key types.NamespacedName, obj client.Object, opts ...client.GetOption) {
			*obj.(*velerov1.DeleteBackupRequest) = *expectedRequestProcessed
		}).Return(nil)
		mockK8sWatchClient.EXPECT().Delete(testCtx, expectedRequestProcessed).Return(nil)

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(backup, corev1.EventTypeNormal, backupv1.ProviderDeleteEventReason, "Trigger velero provider to delete backup.")
		recorderMock.EXPECT().Event(backup, corev1.EventTypeWarning, backupv1.ProviderDeleteEventReason, "Cleanup velero delete request.")
		recorderMock.EXPECT().Event(backup, corev1.EventTypeWarning, backupv1.ErrorOnProviderDeleteEventReason, "velero backup delete request error: error1\nvelero backup delete request error: error2")
		sut := defaultBackupManager{recorder: recorderMock, k8sClient: mockK8sWatchClient}

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

		mockK8sWatchClient := newMockK8sWatchClient(t)
		mockK8sWatchClient.EXPECT().Delete(testCtx, request).Return(assert.AnError)

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(backup, corev1.EventTypeWarning, backupv1.ProviderDeleteEventReason, "Cleanup velero delete request.")
		recorderMock.EXPECT().Event(backup, corev1.EventTypeWarning, backupv1.ProviderDeleteEventReason, "Error cleanup velero delete request.")

		sut := defaultBackupManager{recorder: recorderMock, k8sClient: mockK8sWatchClient}

		// when
		sut.cleanUpDeleteRequest(context.TODO(), backup, request)

	})
}
