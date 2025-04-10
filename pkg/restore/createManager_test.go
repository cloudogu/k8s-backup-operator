package restore

import (
	v1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
	"github.com/cloudogu/k8s-backup-operator/pkg/provider"
	"github.com/cloudogu/k8s-registry-lib/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func Test_newCreateManager(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		configMapMock := newMockConfigMapInterface(t)
		corev1Client := newMockCoreV1Interface(t)
		corev1Client.EXPECT().ConfigMaps(mock.Anything).Return(configMapMock)
		clientSetMock := newMockEcosystemInterface(t)
		clientSetMock.EXPECT().CoreV1().Return(corev1Client)
		ownerReferenceRestoreMock := newMockOwnerReferenceRestore(t)

		// when
		manager := newCreateManager(clientSetMock, testNamespace, nil, nil, ownerReferenceRestoreMock)

		// then
		require.NotNil(t, manager)
	})
}

func Test_defaultCreateManager_create(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		restore := &v1.Restore{ObjectMeta: metav1.ObjectMeta{Name: "restore", Namespace: testNamespace}, Spec: v1.RestoreSpec{BackupName: "backup", Provider: "velero"}}

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(restore, corev1.EventTypeNormal, v1.CreateEventReason, "Start restore process")

		restoreClientMock := newMockEcosystemRestoreInterface(t)
		restoreClientMock.EXPECT().UpdateStatusInProgress(testCtx, restore).Return(restore, nil)
		restoreClientMock.EXPECT().AddFinalizer(testCtx, restore, "cloudogu-restore-finalizer").Return(restore, nil)
		restoreClientMock.EXPECT().AddLabels(testCtx, restore).Return(restore, nil)

		providerMock := newMockRestoreProvider(t)
		providerMock.EXPECT().CheckReady(testCtx).Return(nil)
		providerMock.EXPECT().CreateRestore(testCtx, restore).Return(nil)
		providerMock.EXPECT().SyncBackups(testCtx).Return(nil)
		oldNewVeleroProvider := provider.NewVeleroProvider
		provider.NewVeleroProvider = func(clientSet provider.EcosystemClientSet, recorder provider.EventRecorder, namespace string) (provider.Provider, error) {
			return providerMock, nil
		}
		defer func() { provider.NewVeleroProvider = oldNewVeleroProvider }()

		maintenanceModeMock := newMockMaintenanceModeSwitch(t)
		maintenanceModeMock.EXPECT().Activate(testCtx, repository.MaintenanceModeDescription{Title: "Service temporary unavailable", Text: "Restore in progress"}).Return(nil)
		maintenanceModeMock.EXPECT().Deactivate(testCtx).Return(nil)

		restoreClientMock.EXPECT().UpdateStatusCompleted(testCtx, restore).Return(restore, nil)

		cleanupMock := newMockCleanupManager(t)
		cleanupMock.EXPECT().Cleanup(testCtx).Return(nil)

		v1Alpha1Client := newMockEcosystemV1Alpha1Interface(t)
		v1Alpha1Client.EXPECT().Restores(testNamespace).Return(restoreClientMock)
		clientSetMock := newMockEcosystemInterface(t)
		clientSetMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Client)

		ownerReferenceRestoreMock := newMockOwnerReferenceRestore(t)
		ownerReferenceRestoreMock.EXPECT().RestoreOwnerReferences(testCtx).Return(nil)

		sut := &defaultCreateManager{recorder: recorderMock, ecosystemClientSet: clientSetMock, maintenanceModeSwitch: maintenanceModeMock, cleanup: cleanupMock, namespace: testNamespace, ownerRefRestorer: ownerReferenceRestoreMock}

		// when
		err := sut.create(testCtx, restore)

		// then
		require.NoError(t, err)
	})

	t.Run("should fail to sync backups after restore", func(t *testing.T) {
		// given
		restore := &v1.Restore{ObjectMeta: metav1.ObjectMeta{Name: "restore", Namespace: testNamespace}, Spec: v1.RestoreSpec{BackupName: "backup", Provider: "velero"}}

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(restore, corev1.EventTypeNormal, v1.CreateEventReason, "Start restore process")

		restoreClientMock := newMockEcosystemRestoreInterface(t)
		restoreClientMock.EXPECT().UpdateStatusInProgress(testCtx, restore).Return(restore, nil)
		restoreClientMock.EXPECT().AddFinalizer(testCtx, restore, "cloudogu-restore-finalizer").Return(restore, nil)
		restoreClientMock.EXPECT().AddLabels(testCtx, restore).Return(restore, nil)

		providerMock := newMockRestoreProvider(t)
		providerMock.EXPECT().CheckReady(testCtx).Return(nil)
		providerMock.EXPECT().CreateRestore(testCtx, restore).Return(nil)
		providerMock.EXPECT().SyncBackups(testCtx).Return(assert.AnError)
		oldNewVeleroProvider := provider.NewVeleroProvider
		provider.NewVeleroProvider = func(clientSet provider.EcosystemClientSet, recorder provider.EventRecorder, namespace string) (provider.Provider, error) {
			return providerMock, nil
		}
		defer func() { provider.NewVeleroProvider = oldNewVeleroProvider }()

		maintenanceModeMock := newMockMaintenanceModeSwitch(t)
		maintenanceModeMock.EXPECT().Activate(testCtx, repository.MaintenanceModeDescription{Title: "Service temporary unavailable", Text: "Restore in progress"}).Return(nil)
		maintenanceModeMock.EXPECT().Deactivate(testCtx).Return(nil)

		cleanupMock := newMockCleanupManager(t)
		cleanupMock.EXPECT().Cleanup(testCtx).Return(nil)

		v1Alpha1Client := newMockEcosystemV1Alpha1Interface(t)
		v1Alpha1Client.EXPECT().Restores(testNamespace).Return(restoreClientMock)
		clientSetMock := newMockEcosystemInterface(t)
		clientSetMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Client)

		ownerReferenceRestoreMock := newMockOwnerReferenceRestore(t)

		sut := &defaultCreateManager{recorder: recorderMock, ecosystemClientSet: clientSetMock, maintenanceModeSwitch: maintenanceModeMock, cleanup: cleanupMock, namespace: testNamespace, ownerRefRestorer: ownerReferenceRestoreMock}

		// when
		err := sut.create(testCtx, restore)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to sync backups with provider")
	})

	t.Run("should return error on failing update status in progress", func(t *testing.T) {
		// given
		restore := &v1.Restore{ObjectMeta: metav1.ObjectMeta{Name: "restore", Namespace: testNamespace}, Spec: v1.RestoreSpec{BackupName: "backup"}}

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(restore, corev1.EventTypeNormal, v1.CreateEventReason, "Start restore process")

		restoreClientMock := newMockEcosystemRestoreInterface(t)
		restoreClientMock.EXPECT().UpdateStatusInProgress(testCtx, restore).Return(nil, assert.AnError)

		v1Alpha1Client := newMockEcosystemV1Alpha1Interface(t)
		v1Alpha1Client.EXPECT().Restores(testNamespace).Return(restoreClientMock)
		clientSetMock := newMockEcosystemInterface(t)
		clientSetMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Client)

		ownerReferenceRestoreMock := newMockOwnerReferenceRestore(t)

		sut := &defaultCreateManager{recorder: recorderMock, ecosystemClientSet: clientSetMock, namespace: testNamespace, ownerRefRestorer: ownerReferenceRestoreMock}

		// when
		err := sut.create(testCtx, restore)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to set status [in progress] in restore resource [restore]")
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("should return error on failing add finalizer", func(t *testing.T) {
		// given
		restore := &v1.Restore{ObjectMeta: metav1.ObjectMeta{Name: "restore", Namespace: testNamespace}, Spec: v1.RestoreSpec{BackupName: "backup"}}

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(restore, corev1.EventTypeNormal, v1.CreateEventReason, "Start restore process")

		restoreClientMock := newMockEcosystemRestoreInterface(t)
		restoreClientMock.EXPECT().UpdateStatusInProgress(testCtx, restore).Return(restore, nil)
		restoreClientMock.EXPECT().AddFinalizer(testCtx, restore, "cloudogu-restore-finalizer").Return(nil, assert.AnError)

		v1Alpha1Client := newMockEcosystemV1Alpha1Interface(t)
		v1Alpha1Client.EXPECT().Restores(testNamespace).Return(restoreClientMock)
		clientSetMock := newMockEcosystemInterface(t)
		clientSetMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Client)

		ownerReferenceRestoreMock := newMockOwnerReferenceRestore(t)

		sut := &defaultCreateManager{recorder: recorderMock, ecosystemClientSet: clientSetMock, namespace: testNamespace, ownerRefRestorer: ownerReferenceRestoreMock}

		// when
		err := sut.create(testCtx, restore)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to add finalizer [cloudogu-restore-finalizer] in restore resource [restore]")
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("should return error on failing add finalizer", func(t *testing.T) {
		// given
		restore := &v1.Restore{ObjectMeta: metav1.ObjectMeta{Name: "restore", Namespace: testNamespace}, Spec: v1.RestoreSpec{BackupName: "backup"}}

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(restore, corev1.EventTypeNormal, v1.CreateEventReason, "Start restore process")

		restoreClientMock := newMockEcosystemRestoreInterface(t)
		restoreClientMock.EXPECT().UpdateStatusInProgress(testCtx, restore).Return(restore, nil)
		restoreClientMock.EXPECT().AddFinalizer(testCtx, restore, "cloudogu-restore-finalizer").Return(restore, nil)
		restoreClientMock.EXPECT().AddLabels(testCtx, restore).Return(nil, assert.AnError)

		v1Alpha1Client := newMockEcosystemV1Alpha1Interface(t)
		v1Alpha1Client.EXPECT().Restores(testNamespace).Return(restoreClientMock)
		clientSetMock := newMockEcosystemInterface(t)
		clientSetMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Client)

		ownerReferenceRestoreMock := newMockOwnerReferenceRestore(t)

		sut := &defaultCreateManager{recorder: recorderMock, ecosystemClientSet: clientSetMock, namespace: testNamespace, ownerRefRestorer: ownerReferenceRestoreMock}

		// when
		err := sut.create(testCtx, restore)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to add labels to restore resource [restore]")
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("should return error on failing getting restore provider", func(t *testing.T) {
		// given
		restore := &v1.Restore{ObjectMeta: metav1.ObjectMeta{Name: "restore", Namespace: testNamespace}, Spec: v1.RestoreSpec{BackupName: "backup", Provider: "velero"}}

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(restore, corev1.EventTypeNormal, v1.CreateEventReason, "Start restore process")

		restoreClientMock := newMockEcosystemRestoreInterface(t)
		restoreClientMock.EXPECT().UpdateStatusInProgress(testCtx, restore).Return(restore, nil)
		restoreClientMock.EXPECT().AddFinalizer(testCtx, restore, "cloudogu-restore-finalizer").Return(restore, nil)
		restoreClientMock.EXPECT().AddLabels(testCtx, restore).Return(restore, nil)

		oldNewVeleroProvider := provider.NewVeleroProvider
		provider.NewVeleroProvider = func(clientSet provider.EcosystemClientSet, recorder provider.EventRecorder, namespace string) (provider.Provider, error) {
			return nil, assert.AnError
		}
		defer func() { provider.NewVeleroProvider = oldNewVeleroProvider }()

		v1Alpha1Client := newMockEcosystemV1Alpha1Interface(t)
		v1Alpha1Client.EXPECT().Restores(testNamespace).Return(restoreClientMock)
		clientSetMock := newMockEcosystemInterface(t)
		clientSetMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Client)

		ownerReferenceRestoreMock := newMockOwnerReferenceRestore(t)

		sut := &defaultCreateManager{recorder: recorderMock, ecosystemClientSet: clientSetMock, namespace: testNamespace, ownerRefRestorer: ownerReferenceRestoreMock}

		// when
		err := sut.create(testCtx, restore)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get restore provider [velero]")
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("should continue with restore when failing ti activate maintenance mode", func(t *testing.T) {
		// given
		restore := &v1.Restore{ObjectMeta: metav1.ObjectMeta{Name: "restore", Namespace: testNamespace}, Spec: v1.RestoreSpec{BackupName: "backup", Provider: "velero"}}

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(restore, corev1.EventTypeNormal, v1.CreateEventReason, "Start restore process")

		restoreClientMock := newMockEcosystemRestoreInterface(t)
		restoreClientMock.EXPECT().UpdateStatusInProgress(testCtx, restore).Return(restore, nil)
		restoreClientMock.EXPECT().AddFinalizer(testCtx, restore, "cloudogu-restore-finalizer").Return(restore, nil)
		restoreClientMock.EXPECT().AddLabels(testCtx, restore).Return(restore, nil)

		providerMock := newMockRestoreProvider(t)
		providerMock.EXPECT().CheckReady(testCtx).Return(nil)
		providerMock.EXPECT().CreateRestore(testCtx, restore).Return(nil)
		providerMock.EXPECT().SyncBackups(testCtx).Return(nil)
		oldNewVeleroProvider := provider.NewVeleroProvider
		provider.NewVeleroProvider = func(clientSet provider.EcosystemClientSet, recorder provider.EventRecorder, namespace string) (provider.Provider, error) {
			return providerMock, nil
		}
		defer func() { provider.NewVeleroProvider = oldNewVeleroProvider }()

		maintenanceModeMock := newMockMaintenanceModeSwitch(t)
		maintenanceModeMock.EXPECT().Activate(testCtx, repository.MaintenanceModeDescription{Title: "Service temporary unavailable", Text: "Restore in progress"}).Return(assert.AnError)
		maintenanceModeMock.EXPECT().Deactivate(testCtx).Return(nil)

		restoreClientMock.EXPECT().UpdateStatusCompleted(testCtx, restore).Return(restore, nil)

		cleanupMock := newMockCleanupManager(t)
		cleanupMock.EXPECT().Cleanup(testCtx).Return(nil)

		v1Alpha1Client := newMockEcosystemV1Alpha1Interface(t)
		v1Alpha1Client.EXPECT().Restores(testNamespace).Return(restoreClientMock)
		clientSetMock := newMockEcosystemInterface(t)
		clientSetMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Client)

		ownerReferenceRestoreMock := newMockOwnerReferenceRestore(t)
		ownerReferenceRestoreMock.EXPECT().RestoreOwnerReferences(testCtx).Return(nil)

		sut := &defaultCreateManager{recorder: recorderMock, ecosystemClientSet: clientSetMock, maintenanceModeSwitch: maintenanceModeMock, cleanup: cleanupMock, namespace: testNamespace, ownerRefRestorer: ownerReferenceRestoreMock}

		// when
		err := sut.create(testCtx, restore)

		// then
		require.NoError(t, err)
	})

	t.Run("should return error on cleanup error", func(t *testing.T) {
		// given
		restore := &v1.Restore{ObjectMeta: metav1.ObjectMeta{Name: "restore", Namespace: testNamespace}, Spec: v1.RestoreSpec{BackupName: "backup", Provider: "velero"}}

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(restore, corev1.EventTypeNormal, v1.CreateEventReason, "Start restore process")

		restoreClientMock := newMockEcosystemRestoreInterface(t)
		restoreClientMock.EXPECT().UpdateStatusInProgress(testCtx, restore).Return(restore, nil)
		restoreClientMock.EXPECT().AddFinalizer(testCtx, restore, "cloudogu-restore-finalizer").Return(restore, nil)
		restoreClientMock.EXPECT().AddLabels(testCtx, restore).Return(restore, nil)

		providerMock := newMockRestoreProvider(t)
		providerMock.EXPECT().CheckReady(testCtx).Return(nil)
		oldNewVeleroProvider := provider.NewVeleroProvider
		provider.NewVeleroProvider = func(clientSet provider.EcosystemClientSet, recorder provider.EventRecorder, namespace string) (provider.Provider, error) {
			return providerMock, nil
		}
		defer func() { provider.NewVeleroProvider = oldNewVeleroProvider }()

		maintenanceModeMock := newMockMaintenanceModeSwitch(t)
		maintenanceModeMock.EXPECT().Activate(testCtx, repository.MaintenanceModeDescription{Title: "Service temporary unavailable", Text: "Restore in progress"}).Return(nil)
		maintenanceModeMock.EXPECT().Deactivate(testCtx).Return(nil)

		cleanupMock := newMockCleanupManager(t)
		cleanupMock.EXPECT().Cleanup(testCtx).Return(assert.AnError)
		v1Alpha1Client := newMockEcosystemV1Alpha1Interface(t)
		v1Alpha1Client.EXPECT().Restores(testNamespace).Return(restoreClientMock)
		clientSetMock := newMockEcosystemInterface(t)
		clientSetMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Client)

		ownerReferenceRestoreMock := newMockOwnerReferenceRestore(t)

		sut := &defaultCreateManager{recorder: recorderMock, ecosystemClientSet: clientSetMock, maintenanceModeSwitch: maintenanceModeMock, cleanup: cleanupMock, namespace: testNamespace, ownerRefRestorer: ownerReferenceRestoreMock}

		// when
		err := sut.create(testCtx, restore)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to cleanup before restore")
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("should return error on provider error", func(t *testing.T) {
		// given
		restore := &v1.Restore{ObjectMeta: metav1.ObjectMeta{Name: "restore", Namespace: testNamespace}, Spec: v1.RestoreSpec{BackupName: "backup", Provider: "velero"}}

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(restore, corev1.EventTypeNormal, v1.CreateEventReason, "Start restore process")

		restoreClientMock := newMockEcosystemRestoreInterface(t)
		restoreClientMock.EXPECT().UpdateStatusInProgress(testCtx, restore).Return(restore, nil)
		restoreClientMock.EXPECT().AddFinalizer(testCtx, restore, "cloudogu-restore-finalizer").Return(restore, nil)
		restoreClientMock.EXPECT().AddLabels(testCtx, restore).Return(restore, nil)
		restoreClientMock.EXPECT().UpdateStatusFailed(testCtx, restore).Return(restore, nil)

		providerMock := newMockRestoreProvider(t)
		providerMock.EXPECT().CheckReady(testCtx).Return(nil)
		providerMock.EXPECT().CreateRestore(testCtx, restore).Return(assert.AnError)
		oldNewVeleroProvider := provider.NewVeleroProvider
		provider.NewVeleroProvider = func(clientSet provider.EcosystemClientSet, recorder provider.EventRecorder, namespace string) (provider.Provider, error) {
			return providerMock, nil
		}
		defer func() { provider.NewVeleroProvider = oldNewVeleroProvider }()

		maintenanceModeMock := newMockMaintenanceModeSwitch(t)
		maintenanceModeMock.EXPECT().Activate(testCtx, repository.MaintenanceModeDescription{Title: "Service temporary unavailable", Text: "Restore in progress"}).Return(nil)
		maintenanceModeMock.EXPECT().Deactivate(testCtx).Return(nil)

		cleanupMock := newMockCleanupManager(t)
		cleanupMock.EXPECT().Cleanup(testCtx).Return(nil)
		v1Alpha1Client := newMockEcosystemV1Alpha1Interface(t)
		v1Alpha1Client.EXPECT().Restores(testNamespace).Return(restoreClientMock)
		clientSetMock := newMockEcosystemInterface(t)
		clientSetMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Client)

		ownerReferenceRestoreMock := newMockOwnerReferenceRestore(t)

		sut := &defaultCreateManager{recorder: recorderMock, ecosystemClientSet: clientSetMock, maintenanceModeSwitch: maintenanceModeMock, cleanup: cleanupMock, namespace: testNamespace, ownerRefRestorer: ownerReferenceRestoreMock}

		// when
		err := sut.create(testCtx, restore)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to trigger provider")
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("should wrap status error failing calling provider and update status", func(t *testing.T) {
		// given
		restore := &v1.Restore{ObjectMeta: metav1.ObjectMeta{Name: "restore", Namespace: testNamespace}, Spec: v1.RestoreSpec{BackupName: "backup", Provider: "velero"}}

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(restore, corev1.EventTypeNormal, v1.CreateEventReason, "Start restore process")

		restoreClientMock := newMockEcosystemRestoreInterface(t)
		restoreClientMock.EXPECT().UpdateStatusInProgress(testCtx, restore).Return(restore, nil)
		restoreClientMock.EXPECT().AddFinalizer(testCtx, restore, "cloudogu-restore-finalizer").Return(restore, nil)
		restoreClientMock.EXPECT().AddLabels(testCtx, restore).Return(restore, nil)
		restoreClientMock.EXPECT().UpdateStatusFailed(testCtx, restore).Return(nil, assert.AnError)

		providerMock := newMockRestoreProvider(t)
		providerMock.EXPECT().CheckReady(testCtx).Return(nil)
		providerMock.EXPECT().CreateRestore(testCtx, restore).Return(assert.AnError)
		oldNewVeleroProvider := provider.NewVeleroProvider
		provider.NewVeleroProvider = func(clientSet provider.EcosystemClientSet, recorder provider.EventRecorder, namespace string) (provider.Provider, error) {
			return providerMock, nil
		}
		defer func() { provider.NewVeleroProvider = oldNewVeleroProvider }()

		maintenanceModeMock := newMockMaintenanceModeSwitch(t)
		maintenanceModeMock.EXPECT().Activate(testCtx, repository.MaintenanceModeDescription{Title: "Service temporary unavailable", Text: "Restore in progress"}).Return(nil)
		maintenanceModeMock.EXPECT().Deactivate(testCtx).Return(nil)

		cleanupMock := newMockCleanupManager(t)
		cleanupMock.EXPECT().Cleanup(testCtx).Return(nil)
		v1Alpha1Client := newMockEcosystemV1Alpha1Interface(t)
		v1Alpha1Client.EXPECT().Restores(testNamespace).Return(restoreClientMock)
		clientSetMock := newMockEcosystemInterface(t)
		clientSetMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Client)

		ownerReferenceRestoreMock := newMockOwnerReferenceRestore(t)

		sut := &defaultCreateManager{recorder: recorderMock, ecosystemClientSet: clientSetMock, maintenanceModeSwitch: maintenanceModeMock, cleanup: cleanupMock, namespace: testNamespace, ownerRefRestorer: ownerReferenceRestoreMock}

		// when
		err := sut.create(testCtx, restore)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to trigger provider: assert.AnError general error for testing\nfailed to update restore status to 'failed':")
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("should return error on failing setting completed status", func(t *testing.T) {
		// given
		restore := &v1.Restore{ObjectMeta: metav1.ObjectMeta{Name: "restore", Namespace: testNamespace}, Spec: v1.RestoreSpec{BackupName: "backup", Provider: "velero"}}

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(restore, corev1.EventTypeNormal, v1.CreateEventReason, "Start restore process")

		restoreClientMock := newMockEcosystemRestoreInterface(t)
		restoreClientMock.EXPECT().UpdateStatusInProgress(testCtx, restore).Return(restore, nil)
		restoreClientMock.EXPECT().AddFinalizer(testCtx, restore, "cloudogu-restore-finalizer").Return(restore, nil)
		restoreClientMock.EXPECT().AddLabels(testCtx, restore).Return(restore, nil)
		restoreClientMock.EXPECT().UpdateStatusCompleted(testCtx, restore).Return(nil, assert.AnError)

		providerMock := newMockRestoreProvider(t)
		providerMock.EXPECT().CheckReady(testCtx).Return(nil)
		providerMock.EXPECT().CreateRestore(testCtx, restore).Return(nil)
		providerMock.EXPECT().SyncBackups(testCtx).Return(nil)
		oldNewVeleroProvider := provider.NewVeleroProvider
		provider.NewVeleroProvider = func(clientSet provider.EcosystemClientSet, recorder provider.EventRecorder, namespace string) (provider.Provider, error) {
			return providerMock, nil
		}
		defer func() { provider.NewVeleroProvider = oldNewVeleroProvider }()

		maintenanceModeMock := newMockMaintenanceModeSwitch(t)
		maintenanceModeMock.EXPECT().Activate(testCtx, repository.MaintenanceModeDescription{Title: "Service temporary unavailable", Text: "Restore in progress"}).Return(nil)
		maintenanceModeMock.EXPECT().Deactivate(testCtx).Return(nil)

		cleanupMock := newMockCleanupManager(t)
		cleanupMock.EXPECT().Cleanup(testCtx).Return(nil)

		v1Alpha1Client := newMockEcosystemV1Alpha1Interface(t)
		v1Alpha1Client.EXPECT().Restores(testNamespace).Return(restoreClientMock)
		clientSetMock := newMockEcosystemInterface(t)
		clientSetMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Client)

		ownerReferenceRestoreMock := newMockOwnerReferenceRestore(t)
		ownerReferenceRestoreMock.EXPECT().RestoreOwnerReferences(testCtx).Return(nil)

		sut := &defaultCreateManager{recorder: recorderMock, ecosystemClientSet: clientSetMock, maintenanceModeSwitch: maintenanceModeMock, cleanup: cleanupMock, namespace: testNamespace, ownerRefRestorer: ownerReferenceRestoreMock}

		// when
		err := sut.create(testCtx, restore)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to set status [completed] in restore resource [restore]")
		assert.ErrorIs(t, err, assert.AnError)
	})
}
