package backup

import (
	"github.com/cloudogu/k8s-backup-operator/pkg/provider"
	"testing"

	v1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewBackupCreateManager(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		registryMock := newMockEtcdRegistry(t)
		globalMock := newMockConfigurationContext(t)
		registryMock.EXPECT().GlobalConfig().Return(globalMock)

		// when
		manager := NewBackupCreateManager(nil, nil, registryMock)

		// then
		require.NotNil(t, manager)
	})
}

func Test_backupCreateManager_create(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		backupName := "backup"
		backup := &v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: backupName, Namespace: testNamespace}, Spec: v1.BackupSpec{Provider: "velero"}}

		providerMock := newMockBackupProvider(t)
		providerMock.EXPECT().CheckReady(testCtx).Return(nil)
		providerMock.EXPECT().CreateBackup(testCtx, backup).Return(nil)

		oldVeleroProvider := provider.NewVeleroProvider
		provider.NewVeleroProvider = func(recorder provider.EventRecorder, namespace string) (provider.Provider, error) {
			return providerMock, nil
		}
		defer func() { provider.NewVeleroProvider = oldVeleroProvider }()

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(backup, corev1.EventTypeNormal, v1.CreateEventReason, "Start backup process")
		clientMock := newMockEcosystemBackupInterface(t)
		clientMock.EXPECT().AddFinalizer(testCtx, backup, v1.BackupFinalizer).Return(backup, nil)
		clientMock.EXPECT().UpdateStatusInProgress(testCtx, backup).Return(backup, nil)
		clientMock.EXPECT().UpdateStatusCompleted(testCtx, backup).Return(backup, nil)
		maintenanceModeMock := NewMockMaintenanceModeSwitch(t)
		maintenanceModeMock.EXPECT().ActivateMaintenanceMode("Service temporary unavailable", "Backup in progress").Return(nil)
		maintenanceModeMock.EXPECT().DeactivateMaintenanceMode().Return(nil)

		sut := &backupCreateManager{recorder: recorderMock, client: clientMock, maintenanceModeSwitch: maintenanceModeMock}

		// when
		err := sut.create(testCtx, backup)

		// then
		require.NoError(t, err)
	})

	t.Run("should return error on update status in progress error", func(t *testing.T) {
		// given
		backupName := "backup"
		backup := &v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: backupName, Namespace: testNamespace}}

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(backup, corev1.EventTypeNormal, v1.CreateEventReason, "Start backup process")
		clientMock := newMockEcosystemBackupInterface(t)
		clientMock.EXPECT().UpdateStatusInProgress(testCtx, backup).Return(nil, assert.AnError)

		sut := &backupCreateManager{recorder: recorderMock, client: clientMock}

		// when
		err := sut.create(testCtx, backup)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to set status [in progress] in backup resource")
	})

	t.Run("should return error on finalizer update", func(t *testing.T) {
		// given
		backupName := "backup"
		backup := &v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: backupName, Namespace: testNamespace}}

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(backup, corev1.EventTypeNormal, v1.CreateEventReason, "Start backup process")
		clientMock := newMockEcosystemBackupInterface(t)
		clientMock.EXPECT().UpdateStatusInProgress(testCtx, backup).Return(backup, nil)
		clientMock.EXPECT().AddFinalizer(testCtx, backup, v1.BackupFinalizer).Return(nil, assert.AnError)

		sut := &backupCreateManager{recorder: recorderMock, client: clientMock}

		// when
		err := sut.create(testCtx, backup)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to set finalizer cloudogu-backup-finalizer to backup resource")
	})

	t.Run("should return error activate maintenance mode error", func(t *testing.T) {
		// given
		backupName := "backup"
		backup := &v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: backupName, Namespace: testNamespace}}

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(backup, corev1.EventTypeNormal, v1.CreateEventReason, "Start backup process")
		clientMock := newMockEcosystemBackupInterface(t)
		clientMock.EXPECT().AddFinalizer(testCtx, backup, v1.BackupFinalizer).Return(backup, nil)
		clientMock.EXPECT().UpdateStatusInProgress(testCtx, backup).Return(backup, nil)
		maintenanceModeMock := NewMockMaintenanceModeSwitch(t)
		maintenanceModeMock.EXPECT().ActivateMaintenanceMode("Service temporary unavailable", "Backup in progress").Return(assert.AnError)

		sut := &backupCreateManager{recorder: recorderMock, client: clientMock, maintenanceModeSwitch: maintenanceModeMock}

		// when
		err := sut.create(testCtx, backup)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to active maintenance mode")
	})

	t.Run("should return error with unknown provider type", func(t *testing.T) {
		// given
		backupName := "backup"
		backup := &v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: backupName, Namespace: testNamespace}, Spec: v1.BackupSpec{Provider: "unknown123"}}

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(backup, corev1.EventTypeNormal, v1.CreateEventReason, "Start backup process")
		clientMock := newMockEcosystemBackupInterface(t)
		clientMock.EXPECT().AddFinalizer(testCtx, backup, v1.BackupFinalizer).Return(backup, nil)
		clientMock.EXPECT().UpdateStatusInProgress(testCtx, backup).Return(backup, nil)
		clientMock.EXPECT().UpdateStatusFailed(testCtx, backup).Return(backup, nil)
		maintenanceModeMock := NewMockMaintenanceModeSwitch(t)
		maintenanceModeMock.EXPECT().ActivateMaintenanceMode("Service temporary unavailable", "Backup in progress").Return(nil)
		maintenanceModeMock.EXPECT().DeactivateMaintenanceMode().Return(nil)

		sut := &backupCreateManager{recorder: recorderMock, client: clientMock, maintenanceModeSwitch: maintenanceModeMock}

		// when
		err := sut.create(testCtx, backup)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to trigger backup provider: failed to get backup provider: unknown provider unknown123")
	})

	t.Run("should return error on velero provider creation", func(t *testing.T) {
		// given
		backupName := "backup"
		backup := &v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: backupName, Namespace: testNamespace}, Spec: v1.BackupSpec{Provider: "velero"}}

		oldVeleroProvider := provider.NewVeleroProvider
		provider.NewVeleroProvider = func(recorder provider.EventRecorder, namespace string) (provider.Provider, error) {
			return nil, assert.AnError
		}
		defer func() { provider.NewVeleroProvider = oldVeleroProvider }()

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(backup, corev1.EventTypeNormal, v1.CreateEventReason, "Start backup process")
		clientMock := newMockEcosystemBackupInterface(t)
		clientMock.EXPECT().AddFinalizer(testCtx, backup, v1.BackupFinalizer).Return(backup, nil)
		clientMock.EXPECT().UpdateStatusInProgress(testCtx, backup).Return(backup, nil)
		clientMock.EXPECT().UpdateStatusFailed(testCtx, backup).Return(backup, nil)
		maintenanceModeMock := NewMockMaintenanceModeSwitch(t)
		maintenanceModeMock.EXPECT().ActivateMaintenanceMode("Service temporary unavailable", "Backup in progress").Return(nil)
		maintenanceModeMock.EXPECT().DeactivateMaintenanceMode().Return(nil)

		sut := &backupCreateManager{recorder: recorderMock, client: clientMock, maintenanceModeSwitch: maintenanceModeMock}

		// when
		err := sut.create(testCtx, backup)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to create velero provider")
	})

	t.Run("should return error on provider readiness check", func(t *testing.T) {
		// given
		backupName := "backup"
		backup := &v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: backupName, Namespace: testNamespace}, Spec: v1.BackupSpec{Provider: "velero"}}

		providerMock := newMockBackupProvider(t)
		providerMock.EXPECT().CheckReady(testCtx).Return(assert.AnError)
		oldVeleroProvider := provider.NewVeleroProvider
		provider.NewVeleroProvider = func(recorder provider.EventRecorder, namespace string) (provider.Provider, error) {
			return providerMock, nil
		}
		defer func() { provider.NewVeleroProvider = oldVeleroProvider }()

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(backup, corev1.EventTypeNormal, v1.CreateEventReason, "Start backup process")
		clientMock := newMockEcosystemBackupInterface(t)
		clientMock.EXPECT().AddFinalizer(testCtx, backup, v1.BackupFinalizer).Return(backup, nil)
		clientMock.EXPECT().UpdateStatusInProgress(testCtx, backup).Return(backup, nil)
		clientMock.EXPECT().UpdateStatusFailed(testCtx, backup).Return(backup, nil)
		maintenanceModeMock := NewMockMaintenanceModeSwitch(t)
		maintenanceModeMock.EXPECT().ActivateMaintenanceMode("Service temporary unavailable", "Backup in progress").Return(nil)
		maintenanceModeMock.EXPECT().DeactivateMaintenanceMode().Return(nil)

		sut := &backupCreateManager{recorder: recorderMock, client: clientMock, maintenanceModeSwitch: maintenanceModeMock}

		// when
		err := sut.create(testCtx, backup)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "provider velero is not ready")
	})

	t.Run("should return error on provider error", func(t *testing.T) {
		// given
		backupName := "backup"
		backup := &v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: backupName, Namespace: testNamespace}, Spec: v1.BackupSpec{Provider: "velero"}}

		providerMock := newMockBackupProvider(t)
		providerMock.EXPECT().CheckReady(testCtx).Return(nil)
		providerMock.EXPECT().CreateBackup(testCtx, backup).Return(assert.AnError)
		oldVeleroProvider := provider.NewVeleroProvider
		provider.NewVeleroProvider = func(recorder provider.EventRecorder, namespace string) (provider.Provider, error) {
			return providerMock, nil
		}
		defer func() { provider.NewVeleroProvider = oldVeleroProvider }()

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(backup, corev1.EventTypeNormal, v1.CreateEventReason, "Start backup process")
		clientMock := newMockEcosystemBackupInterface(t)
		clientMock.EXPECT().AddFinalizer(testCtx, backup, v1.BackupFinalizer).Return(backup, nil)
		clientMock.EXPECT().UpdateStatusInProgress(testCtx, backup).Return(backup, nil)
		clientMock.EXPECT().UpdateStatusFailed(testCtx, backup).Return(backup, nil)
		maintenanceModeMock := NewMockMaintenanceModeSwitch(t)
		maintenanceModeMock.EXPECT().ActivateMaintenanceMode("Service temporary unavailable", "Backup in progress").Return(nil)
		maintenanceModeMock.EXPECT().DeactivateMaintenanceMode().Return(nil)

		sut := &backupCreateManager{recorder: recorderMock, client: clientMock, maintenanceModeSwitch: maintenanceModeMock}

		// when
		err := sut.create(testCtx, backup)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to trigger backup provider")
	})

	t.Run("should return error on provider error and fail to update status to 'Failed'", func(t *testing.T) {
		// given
		backupName := "backup"
		backup := &v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: backupName, Namespace: testNamespace}, Spec: v1.BackupSpec{Provider: "velero"}}

		providerMock := newMockBackupProvider(t)
		providerMock.EXPECT().CheckReady(testCtx).Return(nil)
		providerMock.EXPECT().CreateBackup(testCtx, backup).Return(assert.AnError)
		oldVeleroProvider := provider.NewVeleroProvider
		provider.NewVeleroProvider = func(recorder provider.EventRecorder, namespace string) (provider.Provider, error) {
			return providerMock, nil
		}
		defer func() { provider.NewVeleroProvider = oldVeleroProvider }()

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(backup, corev1.EventTypeNormal, v1.CreateEventReason, "Start backup process")
		clientMock := newMockEcosystemBackupInterface(t)
		clientMock.EXPECT().AddFinalizer(testCtx, backup, v1.BackupFinalizer).Return(backup, nil)
		clientMock.EXPECT().UpdateStatusInProgress(testCtx, backup).Return(backup, nil)
		clientMock.EXPECT().UpdateStatusFailed(testCtx, backup).Return(nil, assert.AnError)
		maintenanceModeMock := NewMockMaintenanceModeSwitch(t)
		maintenanceModeMock.EXPECT().ActivateMaintenanceMode("Service temporary unavailable", "Backup in progress").Return(nil)
		maintenanceModeMock.EXPECT().DeactivateMaintenanceMode().Return(nil)

		sut := &backupCreateManager{recorder: recorderMock, client: clientMock, maintenanceModeSwitch: maintenanceModeMock}

		// when
		err := sut.create(testCtx, backup)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to trigger backup provider")
		assert.ErrorContains(t, err, "failed to update backups status to 'Failed'")
	})

	t.Run("should log error on deactivating maintenance mode", func(t *testing.T) {
		// given
		backupName := "backup"
		backup := &v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: backupName, Namespace: testNamespace}, Spec: v1.BackupSpec{Provider: "velero"}}

		providerMock := newMockBackupProvider(t)
		providerMock.EXPECT().CheckReady(testCtx).Return(nil)
		providerMock.EXPECT().CreateBackup(testCtx, backup).Return(nil)
		oldVeleroProvider := provider.NewVeleroProvider
		provider.NewVeleroProvider = func(recorder provider.EventRecorder, namespace string) (provider.Provider, error) {
			return providerMock, nil
		}
		defer func() { provider.NewVeleroProvider = oldVeleroProvider }()

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(backup, corev1.EventTypeNormal, v1.CreateEventReason, "Start backup process")
		clientMock := newMockEcosystemBackupInterface(t)
		clientMock.EXPECT().AddFinalizer(testCtx, backup, v1.BackupFinalizer).Return(backup, nil)
		clientMock.EXPECT().UpdateStatusInProgress(testCtx, backup).Return(backup, nil)
		clientMock.EXPECT().UpdateStatusCompleted(testCtx, backup).Return(nil, nil)
		maintenanceModeMock := NewMockMaintenanceModeSwitch(t)
		maintenanceModeMock.EXPECT().ActivateMaintenanceMode("Service temporary unavailable", "Backup in progress").Return(nil)
		maintenanceModeMock.EXPECT().DeactivateMaintenanceMode().Return(assert.AnError)

		sut := &backupCreateManager{recorder: recorderMock, client: clientMock, maintenanceModeSwitch: maintenanceModeMock}

		// when
		err := sut.create(testCtx, backup)

		// then
		require.NoError(t, err)
	})

	t.Run("should return error on set status completed error", func(t *testing.T) {
		// given
		backupName := "backup"
		backup := &v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: backupName, Namespace: testNamespace}, Spec: v1.BackupSpec{Provider: "velero"}}

		providerMock := newMockBackupProvider(t)
		providerMock.EXPECT().CheckReady(testCtx).Return(nil)
		providerMock.EXPECT().CreateBackup(testCtx, backup).Return(nil)
		oldVeleroProvider := provider.NewVeleroProvider
		provider.NewVeleroProvider = func(recorder provider.EventRecorder, namespace string) (provider.Provider, error) {
			return providerMock, nil
		}
		defer func() { provider.NewVeleroProvider = oldVeleroProvider }()

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(backup, corev1.EventTypeNormal, v1.CreateEventReason, "Start backup process")
		clientMock := newMockEcosystemBackupInterface(t)
		clientMock.EXPECT().AddFinalizer(testCtx, backup, v1.BackupFinalizer).Return(backup, nil)
		clientMock.EXPECT().UpdateStatusInProgress(testCtx, backup).Return(backup, nil)
		clientMock.EXPECT().UpdateStatusCompleted(testCtx, backup).Return(nil, assert.AnError)
		maintenanceModeMock := NewMockMaintenanceModeSwitch(t)
		maintenanceModeMock.EXPECT().ActivateMaintenanceMode("Service temporary unavailable", "Backup in progress").Return(nil)
		maintenanceModeMock.EXPECT().DeactivateMaintenanceMode().Return(nil)

		sut := &backupCreateManager{recorder: recorderMock, client: clientMock, maintenanceModeSwitch: maintenanceModeMock}

		// when
		err := sut.create(testCtx, backup)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to set status [completed] in backup resource")
	})
}
