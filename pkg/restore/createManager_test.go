package restore

import (
	v1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
	"github.com/cloudogu/k8s-backup-operator/pkg/provider"
	"github.com/cloudogu/k8s-backup-operator/pkg/requeue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"testing"
)

func Test_newCreateManager(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		globalConfigMock := newMockConfigurationContext(t)
		registryMock := newMockCesRegistry(t)
		registryMock.EXPECT().GlobalConfig().Return(globalConfigMock)

		// when
		manager := newCreateManager(nil, nil, nil, registryMock)

		// then
		require.NotNil(t, manager)
	})
}

func Test_defaultCreateManager_create(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		restore := &v1.Restore{ObjectMeta: metav1.ObjectMeta{Name: "restore", Namespace: testNamespace}, Spec: v1.RestoreSpec{BackupName: "backup"}}
		backup := &v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "backup", Namespace: testNamespace}, Spec: v1.BackupSpec{Provider: "velero"}}

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(restore, corev1.EventTypeNormal, v1.CreateEventReason, "Start restore process")

		restoreClientMock := newMockEcosystemRestoreInterface(t)
		backupClientMock := newMockEcosystemBackupInterface(t)
		restoreClientMock.EXPECT().UpdateStatusInProgress(testCtx, restore).Return(restore, nil)
		restoreClientMock.EXPECT().AddFinalizer(testCtx, restore, "cloudogu-restore-finalizer").Return(restore, nil)
		backupClientMock.EXPECT().Get(testCtx, "backup", metav1.GetOptions{}).Return(backup, nil)

		providerMock := newMockRestoreProvider(t)
		providerMock.EXPECT().CheckReady(testCtx).Return(nil)
		providerMock.EXPECT().CreateRestore(testCtx, restore).Return(nil)
		oldNewVeleroProvider := provider.NewVeleroProvider
		provider.NewVeleroProvider = func(recorder provider.EventRecorder, namespace string) (provider.Provider, error) {
			return providerMock, nil
		}
		defer func() { provider.NewVeleroProvider = oldNewVeleroProvider }()

		maintenanceModeMock := newMockMaintenanceModeSwitch(t)
		maintenanceModeMock.EXPECT().ActivateMaintenanceMode("Service temporary unavailable", "Restore in progress").Return(nil)
		maintenanceModeMock.EXPECT().DeactivateMaintenanceMode().Return(nil)

		restoreClientMock.EXPECT().UpdateStatusCompleted(testCtx, restore).Return(restore, nil)

		sut := &defaultCreateManager{recorder: recorderMock, restoreClient: restoreClientMock, backupClient: backupClientMock, maintenanceModeSwitch: maintenanceModeMock}

		// when
		err := sut.create(testCtx, restore)

		// then
		require.NoError(t, err)
	})

	t.Run("should return error on failing update status in progress", func(t *testing.T) {
		// given
		restore := &v1.Restore{ObjectMeta: metav1.ObjectMeta{Name: "restore", Namespace: testNamespace}, Spec: v1.RestoreSpec{BackupName: "backup"}}

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(restore, corev1.EventTypeNormal, v1.CreateEventReason, "Start restore process")

		restoreClientMock := newMockEcosystemRestoreInterface(t)
		restoreClientMock.EXPECT().UpdateStatusInProgress(testCtx, restore).Return(nil, assert.AnError)

		sut := &defaultCreateManager{recorder: recorderMock, restoreClient: restoreClientMock}

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

		sut := &defaultCreateManager{recorder: recorderMock, restoreClient: restoreClientMock}

		// when
		err := sut.create(testCtx, restore)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to add finalizer [cloudogu-restore-finalizer] in restore resource [restore]")
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("should return error if provided backup is not found", func(t *testing.T) {
		// given
		restore := &v1.Restore{ObjectMeta: metav1.ObjectMeta{Name: "restore", Namespace: testNamespace}, Spec: v1.RestoreSpec{BackupName: "backup"}}

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(restore, corev1.EventTypeNormal, v1.CreateEventReason, "Start restore process")

		restoreClientMock := newMockEcosystemRestoreInterface(t)
		restoreClientMock.EXPECT().UpdateStatusInProgress(testCtx, restore).Return(restore, nil)
		restoreClientMock.EXPECT().AddFinalizer(testCtx, restore, "cloudogu-restore-finalizer").Return(restore, nil)
		backupClientMock := newMockEcosystemBackupInterface(t)
		backupClientMock.EXPECT().Get(testCtx, "backup", metav1.GetOptions{}).Return(nil, apierrors.NewNotFound(schema.GroupResource(metav1.GroupResource{}), "backup"))

		sut := &defaultCreateManager{recorder: recorderMock, restoreClient: restoreClientMock, backupClient: backupClientMock}

		// when
		err := sut.create(testCtx, restore)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "found no backup with name [backup] for restore resource [restore]:  \"backup\" not found")
	})

	t.Run("should return requeue error on failing getting backup", func(t *testing.T) {
		// given
		restore := &v1.Restore{ObjectMeta: metav1.ObjectMeta{Name: "restore", Namespace: testNamespace}, Spec: v1.RestoreSpec{BackupName: "backup"}}

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(restore, corev1.EventTypeNormal, v1.CreateEventReason, "Start restore process")

		restoreClientMock := newMockEcosystemRestoreInterface(t)
		restoreClientMock.EXPECT().UpdateStatusInProgress(testCtx, restore).Return(restore, nil)
		restoreClientMock.EXPECT().AddFinalizer(testCtx, restore, "cloudogu-restore-finalizer").Return(restore, nil)
		backupClientMock := newMockEcosystemBackupInterface(t)
		backupClientMock.EXPECT().Get(testCtx, "backup", metav1.GetOptions{}).Return(nil, assert.AnError)

		sut := &defaultCreateManager{recorder: recorderMock, restoreClient: restoreClientMock, backupClient: backupClientMock}

		// when
		err := sut.create(testCtx, restore)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get backup [backup] for restore resource [restore]")
		assert.IsType(t, &requeue.GenericRequeueableError{}, err)
	})

	t.Run("should return error on failing getting restore provider", func(t *testing.T) {
		// given
		restore := &v1.Restore{ObjectMeta: metav1.ObjectMeta{Name: "restore", Namespace: testNamespace}, Spec: v1.RestoreSpec{BackupName: "backup"}}
		backup := &v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "backup", Namespace: testNamespace}, Spec: v1.BackupSpec{Provider: "velero"}}

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(restore, corev1.EventTypeNormal, v1.CreateEventReason, "Start restore process")

		restoreClientMock := newMockEcosystemRestoreInterface(t)
		restoreClientMock.EXPECT().UpdateStatusInProgress(testCtx, restore).Return(restore, nil)
		restoreClientMock.EXPECT().AddFinalizer(testCtx, restore, "cloudogu-restore-finalizer").Return(restore, nil)
		backupClientMock := newMockEcosystemBackupInterface(t)
		backupClientMock.EXPECT().Get(testCtx, "backup", metav1.GetOptions{}).Return(backup, nil)

		oldNewVeleroProvider := provider.NewVeleroProvider
		provider.NewVeleroProvider = func(recorder provider.EventRecorder, namespace string) (provider.Provider, error) {
			return nil, assert.AnError
		}
		defer func() { provider.NewVeleroProvider = oldNewVeleroProvider }()

		sut := &defaultCreateManager{recorder: recorderMock, restoreClient: restoreClientMock, backupClient: backupClientMock}

		// when
		err := sut.create(testCtx, restore)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get restore provider [velero]")
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("should return error failing activate maintenance mode", func(t *testing.T) {
		// given
		restore := &v1.Restore{ObjectMeta: metav1.ObjectMeta{Name: "restore", Namespace: testNamespace}, Spec: v1.RestoreSpec{BackupName: "backup"}}
		backup := &v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "backup", Namespace: testNamespace}, Spec: v1.BackupSpec{Provider: "velero"}}

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(restore, corev1.EventTypeNormal, v1.CreateEventReason, "Start restore process")

		restoreClientMock := newMockEcosystemRestoreInterface(t)
		restoreClientMock.EXPECT().UpdateStatusInProgress(testCtx, restore).Return(restore, nil)
		restoreClientMock.EXPECT().AddFinalizer(testCtx, restore, "cloudogu-restore-finalizer").Return(restore, nil)
		backupClientMock := newMockEcosystemBackupInterface(t)
		backupClientMock.EXPECT().Get(testCtx, "backup", metav1.GetOptions{}).Return(backup, nil)

		providerMock := newMockRestoreProvider(t)
		providerMock.EXPECT().CheckReady(testCtx).Return(nil)
		oldNewVeleroProvider := provider.NewVeleroProvider
		provider.NewVeleroProvider = func(recorder provider.EventRecorder, namespace string) (provider.Provider, error) {
			return providerMock, nil
		}
		defer func() { provider.NewVeleroProvider = oldNewVeleroProvider }()

		maintenanceModeMock := newMockMaintenanceModeSwitch(t)
		maintenanceModeMock.EXPECT().ActivateMaintenanceMode("Service temporary unavailable", "Restore in progress").Return(assert.AnError)

		sut := &defaultCreateManager{recorder: recorderMock, restoreClient: restoreClientMock, backupClient: backupClientMock, maintenanceModeSwitch: maintenanceModeMock}

		// when
		err := sut.create(testCtx, restore)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to activate maintenance mode")
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("should return error on provider error", func(t *testing.T) {
		// given
		restore := &v1.Restore{ObjectMeta: metav1.ObjectMeta{Name: "restore", Namespace: testNamespace}, Spec: v1.RestoreSpec{BackupName: "backup"}}
		backup := &v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "backup", Namespace: testNamespace}, Spec: v1.BackupSpec{Provider: "velero"}}

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(restore, corev1.EventTypeNormal, v1.CreateEventReason, "Start restore process")

		restoreClientMock := newMockEcosystemRestoreInterface(t)
		restoreClientMock.EXPECT().UpdateStatusInProgress(testCtx, restore).Return(restore, nil)
		restoreClientMock.EXPECT().AddFinalizer(testCtx, restore, "cloudogu-restore-finalizer").Return(restore, nil)
		restoreClientMock.EXPECT().UpdateStatusFailed(testCtx, restore).Return(restore, nil)
		backupClientMock := newMockEcosystemBackupInterface(t)
		backupClientMock.EXPECT().Get(testCtx, "backup", metav1.GetOptions{}).Return(backup, nil)

		providerMock := newMockRestoreProvider(t)
		providerMock.EXPECT().CheckReady(testCtx).Return(nil)
		providerMock.EXPECT().CreateRestore(testCtx, restore).Return(assert.AnError)
		oldNewVeleroProvider := provider.NewVeleroProvider
		provider.NewVeleroProvider = func(recorder provider.EventRecorder, namespace string) (provider.Provider, error) {
			return providerMock, nil
		}
		defer func() { provider.NewVeleroProvider = oldNewVeleroProvider }()

		maintenanceModeMock := newMockMaintenanceModeSwitch(t)
		maintenanceModeMock.EXPECT().ActivateMaintenanceMode("Service temporary unavailable", "Restore in progress").Return(nil)
		maintenanceModeMock.EXPECT().DeactivateMaintenanceMode().Return(nil)

		sut := &defaultCreateManager{recorder: recorderMock, restoreClient: restoreClientMock, backupClient: backupClientMock, maintenanceModeSwitch: maintenanceModeMock}

		// when
		err := sut.create(testCtx, restore)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to trigger provider")
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("should wrap status error failing calling provider and update status", func(t *testing.T) {
		// given
		restore := &v1.Restore{ObjectMeta: metav1.ObjectMeta{Name: "restore", Namespace: testNamespace}, Spec: v1.RestoreSpec{BackupName: "backup"}}
		backup := &v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "backup", Namespace: testNamespace}, Spec: v1.BackupSpec{Provider: "velero"}}

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(restore, corev1.EventTypeNormal, v1.CreateEventReason, "Start restore process")

		restoreClientMock := newMockEcosystemRestoreInterface(t)
		restoreClientMock.EXPECT().UpdateStatusInProgress(testCtx, restore).Return(restore, nil)
		restoreClientMock.EXPECT().AddFinalizer(testCtx, restore, "cloudogu-restore-finalizer").Return(restore, nil)
		restoreClientMock.EXPECT().UpdateStatusFailed(testCtx, restore).Return(nil, assert.AnError)
		backupClientMock := newMockEcosystemBackupInterface(t)
		backupClientMock.EXPECT().Get(testCtx, "backup", metav1.GetOptions{}).Return(backup, nil)

		providerMock := newMockRestoreProvider(t)
		providerMock.EXPECT().CheckReady(testCtx).Return(nil)
		providerMock.EXPECT().CreateRestore(testCtx, restore).Return(assert.AnError)
		oldNewVeleroProvider := provider.NewVeleroProvider
		provider.NewVeleroProvider = func(recorder provider.EventRecorder, namespace string) (provider.Provider, error) {
			return providerMock, nil
		}
		defer func() { provider.NewVeleroProvider = oldNewVeleroProvider }()

		maintenanceModeMock := newMockMaintenanceModeSwitch(t)
		maintenanceModeMock.EXPECT().ActivateMaintenanceMode("Service temporary unavailable", "Restore in progress").Return(nil)
		maintenanceModeMock.EXPECT().DeactivateMaintenanceMode().Return(nil)

		sut := &defaultCreateManager{recorder: recorderMock, restoreClient: restoreClientMock, backupClient: backupClientMock, maintenanceModeSwitch: maintenanceModeMock}

		// when
		err := sut.create(testCtx, restore)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to trigger provider: assert.AnError general error for testing\nfailed to update restore status to 'failed':")
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("should return error on failing setting completed status", func(t *testing.T) {
		// given
		restore := &v1.Restore{ObjectMeta: metav1.ObjectMeta{Name: "restore", Namespace: testNamespace}, Spec: v1.RestoreSpec{BackupName: "backup"}}
		backup := &v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "backup", Namespace: testNamespace}, Spec: v1.BackupSpec{Provider: "velero"}}

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(restore, corev1.EventTypeNormal, v1.CreateEventReason, "Start restore process")

		restoreClientMock := newMockEcosystemRestoreInterface(t)
		restoreClientMock.EXPECT().UpdateStatusInProgress(testCtx, restore).Return(restore, nil)
		restoreClientMock.EXPECT().AddFinalizer(testCtx, restore, "cloudogu-restore-finalizer").Return(restore, nil)
		restoreClientMock.EXPECT().UpdateStatusCompleted(testCtx, restore).Return(nil, assert.AnError)
		backupClientMock := newMockEcosystemBackupInterface(t)
		backupClientMock.EXPECT().Get(testCtx, "backup", metav1.GetOptions{}).Return(backup, nil)

		providerMock := newMockRestoreProvider(t)
		providerMock.EXPECT().CheckReady(testCtx).Return(nil)
		providerMock.EXPECT().CreateRestore(testCtx, restore).Return(nil)
		oldNewVeleroProvider := provider.NewVeleroProvider
		provider.NewVeleroProvider = func(recorder provider.EventRecorder, namespace string) (provider.Provider, error) {
			return providerMock, nil
		}
		defer func() { provider.NewVeleroProvider = oldNewVeleroProvider }()

		maintenanceModeMock := newMockMaintenanceModeSwitch(t)
		maintenanceModeMock.EXPECT().ActivateMaintenanceMode("Service temporary unavailable", "Restore in progress").Return(nil)
		maintenanceModeMock.EXPECT().DeactivateMaintenanceMode().Return(nil)

		sut := &defaultCreateManager{recorder: recorderMock, restoreClient: restoreClientMock, backupClient: backupClientMock, maintenanceModeSwitch: maintenanceModeMock}

		// when
		err := sut.create(testCtx, restore)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to set status [completed] in restore resource [restore]")
		assert.ErrorIs(t, err, assert.AnError)
	})
}
