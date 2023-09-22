package backup

import (
	"context"
	"github.com/cloudogu/k8s-backup-operator/pkg/api/ecosystem"
	v1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
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

		providerMock := NewMockProvider(t)
		providerMock.EXPECT().CreateBackup(context.TODO(), backup).Return(nil)
		oldVeleroProvider := newVeleroProvider
		newVeleroProvider = func(client ecosystem.BackupInterface, recorder eventRecorder) Provider {
			return providerMock
		}
		defer func() { newVeleroProvider = oldVeleroProvider }()

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(backup, corev1.EventTypeNormal, CreateEventReason, "Start backup process")
		recorderMock.EXPECT().Event(backup, corev1.EventTypeNormal, CreateEventReason, "Use velero as backup provider")
		clientMock := newMockEcosystemBackupInterface(t)
		clientMock.EXPECT().UpdateStatusInProgress(context.TODO(), backup).Return(backup, nil)
		clientMock.EXPECT().UpdateStatusCompleted(context.TODO(), backup).Return(backup, nil)
		maintenanceModeMock := NewMockMaintenanceModeSwitch(t)
		maintenanceModeMock.EXPECT().ActivateMaintenanceMode("Service temporary unavailable", "Backup in progress").Return(nil)
		maintenanceModeMock.EXPECT().DeactivateMaintenanceMode().Return(nil)

		sut := &backupCreateManager{recorder: recorderMock, client: clientMock, maintenanceModeSwitch: maintenanceModeMock}

		// when
		err := sut.create(context.TODO(), backup)

		// then
		require.NoError(t, err)
	})

	t.Run("should return error on update status in progress error", func(t *testing.T) {
		// given
		backupName := "backup"
		backup := &v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: backupName, Namespace: testNamespace}}

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(backup, corev1.EventTypeNormal, CreateEventReason, "Start backup process")
		clientMock := newMockEcosystemBackupInterface(t)
		clientMock.EXPECT().UpdateStatusInProgress(context.TODO(), backup).Return(nil, assert.AnError)

		sut := &backupCreateManager{recorder: recorderMock, client: clientMock}

		// when
		err := sut.create(context.TODO(), backup)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to set status [in progress] in backup resource")
	})

	t.Run("should return error activate maintenance mode error", func(t *testing.T) {
		// given
		backupName := "backup"
		backup := &v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: backupName, Namespace: testNamespace}}

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(backup, corev1.EventTypeNormal, CreateEventReason, "Start backup process")
		clientMock := newMockEcosystemBackupInterface(t)
		clientMock.EXPECT().UpdateStatusInProgress(context.TODO(), backup).Return(backup, nil)
		maintenanceModeMock := NewMockMaintenanceModeSwitch(t)
		maintenanceModeMock.EXPECT().ActivateMaintenanceMode("Service temporary unavailable", "Backup in progress").Return(assert.AnError)

		sut := &backupCreateManager{recorder: recorderMock, client: clientMock, maintenanceModeSwitch: maintenanceModeMock}

		// when
		err := sut.create(context.TODO(), backup)

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
		recorderMock.EXPECT().Event(backup, corev1.EventTypeNormal, CreateEventReason, "Start backup process")
		clientMock := newMockEcosystemBackupInterface(t)
		clientMock.EXPECT().UpdateStatusInProgress(context.TODO(), backup).Return(backup, nil)
		maintenanceModeMock := NewMockMaintenanceModeSwitch(t)
		maintenanceModeMock.EXPECT().ActivateMaintenanceMode("Service temporary unavailable", "Backup in progress").Return(nil)

		sut := &backupCreateManager{recorder: recorderMock, client: clientMock, maintenanceModeSwitch: maintenanceModeMock}

		// when
		err := sut.create(context.TODO(), backup)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to trigger backup provider: unknown backup provider unknown123")
	})

	t.Run("should return error on provider error", func(t *testing.T) {
		// given
		backupName := "backup"
		backup := &v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: backupName, Namespace: testNamespace}, Spec: v1.BackupSpec{Provider: "velero"}}

		providerMock := NewMockProvider(t)
		providerMock.EXPECT().CreateBackup(context.TODO(), backup).Return(assert.AnError)
		oldVeleroProvider := newVeleroProvider
		newVeleroProvider = func(client ecosystem.BackupInterface, recorder eventRecorder) Provider {
			return providerMock
		}
		defer func() { newVeleroProvider = oldVeleroProvider }()

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(backup, corev1.EventTypeNormal, CreateEventReason, "Start backup process")
		recorderMock.EXPECT().Event(backup, corev1.EventTypeNormal, CreateEventReason, "Use velero as backup provider")
		clientMock := newMockEcosystemBackupInterface(t)
		clientMock.EXPECT().UpdateStatusInProgress(context.TODO(), backup).Return(backup, nil)
		maintenanceModeMock := NewMockMaintenanceModeSwitch(t)
		maintenanceModeMock.EXPECT().ActivateMaintenanceMode("Service temporary unavailable", "Backup in progress").Return(nil)

		sut := &backupCreateManager{recorder: recorderMock, client: clientMock, maintenanceModeSwitch: maintenanceModeMock}

		// when
		err := sut.create(context.TODO(), backup)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to trigger backup provider")
	})

	t.Run("should return error on deactivating maintenance mode", func(t *testing.T) {
		// given
		backupName := "backup"
		backup := &v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: backupName, Namespace: testNamespace}, Spec: v1.BackupSpec{Provider: "velero"}}

		providerMock := NewMockProvider(t)
		providerMock.EXPECT().CreateBackup(context.TODO(), backup).Return(nil)
		oldVeleroProvider := newVeleroProvider
		newVeleroProvider = func(client ecosystem.BackupInterface, recorder eventRecorder) Provider {
			return providerMock
		}
		defer func() { newVeleroProvider = oldVeleroProvider }()

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(backup, corev1.EventTypeNormal, CreateEventReason, "Start backup process")
		recorderMock.EXPECT().Event(backup, corev1.EventTypeNormal, CreateEventReason, "Use velero as backup provider")
		clientMock := newMockEcosystemBackupInterface(t)
		clientMock.EXPECT().UpdateStatusInProgress(context.TODO(), backup).Return(backup, nil)
		maintenanceModeMock := NewMockMaintenanceModeSwitch(t)
		maintenanceModeMock.EXPECT().ActivateMaintenanceMode("Service temporary unavailable", "Backup in progress").Return(nil)
		maintenanceModeMock.EXPECT().DeactivateMaintenanceMode().Return(assert.AnError)

		sut := &backupCreateManager{recorder: recorderMock, client: clientMock, maintenanceModeSwitch: maintenanceModeMock}

		// when
		err := sut.create(context.TODO(), backup)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to deactivate maintenance mode")
	})

	t.Run("should return error on set status completed error", func(t *testing.T) {
		// given
		backupName := "backup"
		backup := &v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: backupName, Namespace: testNamespace}, Spec: v1.BackupSpec{Provider: "velero"}}

		providerMock := NewMockProvider(t)
		providerMock.EXPECT().CreateBackup(context.TODO(), backup).Return(nil)
		oldVeleroProvider := newVeleroProvider
		newVeleroProvider = func(client ecosystem.BackupInterface, recorder eventRecorder) Provider {
			return providerMock
		}
		defer func() { newVeleroProvider = oldVeleroProvider }()

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(backup, corev1.EventTypeNormal, CreateEventReason, "Start backup process")
		recorderMock.EXPECT().Event(backup, corev1.EventTypeNormal, CreateEventReason, "Use velero as backup provider")
		clientMock := newMockEcosystemBackupInterface(t)
		clientMock.EXPECT().UpdateStatusInProgress(context.TODO(), backup).Return(backup, nil)
		clientMock.EXPECT().UpdateStatusCompleted(context.TODO(), backup).Return(nil, assert.AnError)
		maintenanceModeMock := NewMockMaintenanceModeSwitch(t)
		maintenanceModeMock.EXPECT().ActivateMaintenanceMode("Service temporary unavailable", "Backup in progress").Return(nil)
		maintenanceModeMock.EXPECT().DeactivateMaintenanceMode().Return(nil)

		sut := &backupCreateManager{recorder: recorderMock, client: clientMock, maintenanceModeSwitch: maintenanceModeMock}

		// when
		err := sut.create(context.TODO(), backup)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to set status [completed] in backup resource")
	})
}
