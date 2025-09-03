package backup

import (
	"context"
	"github.com/cloudogu/k8s-registry-lib/repository"
	"github.com/stretchr/testify/mock"
	"testing"

	v1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/k8s-backup-operator/pkg/provider"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var testCtx = context.TODO()

func TestNewBackupCreateManager(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		ownerReferenceBackupMock := newMockOwnerReferenceBackup(t)
		globalConfigRepositoryMock := newMockGlobalConfigRepository(t)
		configMapMock := newMockBackupConfigMapInterface(t)
		corev1Client := newMockBackupCoreV1Interface(t)
		corev1Client.EXPECT().ConfigMaps(mock.Anything).Return(configMapMock)
		clientSetMock := newMockEcosystemInterface(t)
		clientSetMock.EXPECT().CoreV1().Return(corev1Client)
		clientMock := newMockK8sClient(t)

		// when
		manager := newBackupCreateManager(clientMock, clientSetMock, "", nil, globalConfigRepositoryMock, ownerReferenceBackupMock)

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
		provider.NewVeleroProvider = func(client provider.K8sClient, ecosystemClientSet provider.EcosystemClientSet, recorder provider.EventRecorder, namespace string) provider.Provider {
			return providerMock
		}
		defer func() { provider.NewVeleroProvider = oldVeleroProvider }()

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(backup, corev1.EventTypeNormal, v1.CreateEventReason, "Start backup process")
		clientMock := newMockEcosystemBackupInterface(t)
		clientMock.EXPECT().UpdateStatus(testCtx, backup, metav1.UpdateOptions{}).Run(func(ctx context.Context, backup *v1.Backup, opts metav1.UpdateOptions) {
			assert.NotEmpty(t, backup.Status.StartTimestamp)
		}).Return(backup, nil)
		clientMock.EXPECT().AddFinalizer(testCtx, backup, v1.BackupFinalizer).Return(backup, nil)
		clientMock.EXPECT().AddLabels(testCtx, backup).Return(backup, nil)
		clientMock.EXPECT().UpdateStatusInProgress(testCtx, backup).Return(backup, nil)
		clientMock.EXPECT().UpdateStatusCompleted(testCtx, backup).Return(backup, nil)
		clientMock.EXPECT().Get(testCtx, backup.Name, metav1.GetOptions{}).Return(backup, nil)
		clientMock.EXPECT().UpdateStatus(testCtx, backup, metav1.UpdateOptions{}).Run(func(ctx context.Context, backup *v1.Backup, opts metav1.UpdateOptions) {
			assert.NotEmpty(t, backup.Status.CompletionTimestamp)
		}).Return(backup, nil)
		maintenanceModeMock := NewMockMaintenanceModeSwitch(t)
		maintenanceModeMock.EXPECT().Activate(testCtx, repository.MaintenanceModeDescription{Title: "Service temporary unavailable", Text: "Backup in progress"}).Return(nil)
		maintenanceModeMock.EXPECT().Deactivate(testCtx).Return(nil)

		v1Alpha1Client := newMockBackupV1Alpha1Interface(t)
		v1Alpha1Client.EXPECT().Backups(testNamespace).Return(clientMock)
		clientSetMock := newMockEcosystemInterface(t)
		clientSetMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Client)

		ownerReferenceBackupMock := newMockOwnerReferenceBackup(t)
		ownerReferenceBackupMock.EXPECT().BackupOwnerReferences(testCtx).Return(nil)

		sut := &backupCreateManager{recorder: recorderMock, clientSet: clientSetMock, maintenanceModeSwitch: maintenanceModeMock, namespace: testNamespace, ownerRefBackuper: ownerReferenceBackupMock}

		// when
		err := sut.create(testCtx, backup)

		// then
		require.NoError(t, err)
	})

	t.Run("should select velero provider as default with empty provider", func(t *testing.T) {
		// given
		backupName := "backup"
		backup := &v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: backupName, Namespace: testNamespace}, Spec: v1.BackupSpec{Provider: ""}}

		providerMock := newMockBackupProvider(t)
		providerMock.EXPECT().CheckReady(testCtx).Return(nil)
		providerMock.EXPECT().CreateBackup(testCtx, backup).Return(nil)
		oldVeleroProvider := provider.NewVeleroProvider
		provider.NewVeleroProvider = func(client provider.K8sClient, ecosystemClientSet provider.EcosystemClientSet, recorder provider.EventRecorder, namespace string) provider.Provider {
			return providerMock
		}
		defer func() { provider.NewVeleroProvider = oldVeleroProvider }()

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(backup, corev1.EventTypeNormal, v1.CreateEventReason, "Start backup process")
		recorderMock.EXPECT().Event(backup, corev1.EventTypeNormal, v1.ProviderSelectEventReason, "No provider given. Select velero as default provider.")
		clientMock := newMockEcosystemBackupInterface(t)
		clientMock.EXPECT().UpdateStatus(testCtx, backup, metav1.UpdateOptions{}).Run(func(ctx context.Context, backup *v1.Backup, opts metav1.UpdateOptions) {
			assert.NotEmpty(t, backup.Status.StartTimestamp)
		}).Return(backup, nil)
		clientMock.EXPECT().AddFinalizer(testCtx, backup, v1.BackupFinalizer).Return(backup, nil)
		clientMock.EXPECT().AddLabels(testCtx, backup).Return(backup, nil)
		clientMock.EXPECT().UpdateStatusInProgress(testCtx, backup).Return(backup, nil)
		clientMock.EXPECT().UpdateStatusCompleted(testCtx, backup).Return(backup, nil)
		clientMock.EXPECT().Get(testCtx, backup.Name, metav1.GetOptions{}).Return(backup, nil)
		clientMock.EXPECT().UpdateStatus(testCtx, backup, metav1.UpdateOptions{}).Run(func(ctx context.Context, backup *v1.Backup, opts metav1.UpdateOptions) {
			assert.NotEmpty(t, backup.Status.CompletionTimestamp)
		}).Return(backup, nil)
		maintenanceModeMock := NewMockMaintenanceModeSwitch(t)
		maintenanceModeMock.EXPECT().Activate(testCtx, repository.MaintenanceModeDescription{Title: "Service temporary unavailable", Text: "Backup in progress"}).Return(nil)
		maintenanceModeMock.EXPECT().Deactivate(testCtx).Return(nil)

		v1Alpha1Client := newMockBackupV1Alpha1Interface(t)
		v1Alpha1Client.EXPECT().Backups(testNamespace).Return(clientMock)
		clientSetMock := newMockEcosystemInterface(t)
		clientSetMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Client)

		ownerReferenceBackupMock := newMockOwnerReferenceBackup(t)
		ownerReferenceBackupMock.EXPECT().BackupOwnerReferences(testCtx).Return(nil)

		sut := &backupCreateManager{recorder: recorderMock, clientSet: clientSetMock, maintenanceModeSwitch: maintenanceModeMock, namespace: testNamespace, ownerRefBackuper: ownerReferenceBackupMock}

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

		v1Alpha1Client := newMockBackupV1Alpha1Interface(t)
		v1Alpha1Client.EXPECT().Backups(testNamespace).Return(clientMock)
		clientSetMock := newMockEcosystemInterface(t)
		clientSetMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Client)

		sut := &backupCreateManager{recorder: recorderMock, clientSet: clientSetMock, namespace: testNamespace}

		// when
		err := sut.create(testCtx, backup)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to set status [in progress] in backup resource")
	})

	t.Run("should return error on updating start timestamp", func(t *testing.T) {
		// given
		backupName := "backup"
		backup := &v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: backupName, Namespace: testNamespace}}

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(backup, corev1.EventTypeNormal, v1.CreateEventReason, "Start backup process")
		clientMock := newMockEcosystemBackupInterface(t)
		clientMock.EXPECT().UpdateStatusInProgress(testCtx, backup).Return(backup, nil)
		clientMock.EXPECT().UpdateStatus(testCtx, backup, metav1.UpdateOptions{}).Run(func(ctx context.Context, backup *v1.Backup, opts metav1.UpdateOptions) {
			assert.NotEmpty(t, backup.Status.StartTimestamp)
		}).Return(nil, assert.AnError)

		v1Alpha1Client := newMockBackupV1Alpha1Interface(t)
		v1Alpha1Client.EXPECT().Backups(testNamespace).Return(clientMock)
		clientSetMock := newMockEcosystemInterface(t)
		clientSetMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Client)

		sut := &backupCreateManager{recorder: recorderMock, clientSet: clientSetMock, namespace: testNamespace}

		// when
		err := sut.create(testCtx, backup)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to update start time in status of backup resource")
	})

	t.Run("should return error on finalizer update", func(t *testing.T) {
		// given
		backupName := "backup"
		backup := &v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: backupName, Namespace: testNamespace}}

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(backup, corev1.EventTypeNormal, v1.CreateEventReason, "Start backup process")
		clientMock := newMockEcosystemBackupInterface(t)
		clientMock.EXPECT().UpdateStatusInProgress(testCtx, backup).Return(backup, nil)
		clientMock.EXPECT().UpdateStatus(testCtx, backup, metav1.UpdateOptions{}).Run(func(ctx context.Context, backup *v1.Backup, opts metav1.UpdateOptions) {
			assert.NotEmpty(t, backup.Status.StartTimestamp)
		}).Return(backup, nil)
		clientMock.EXPECT().Get(testCtx, backup.Name, metav1.GetOptions{}).Return(backup, nil)
		clientMock.EXPECT().UpdateStatus(testCtx, backup, metav1.UpdateOptions{}).Run(func(ctx context.Context, backup *v1.Backup, opts metav1.UpdateOptions) {
			assert.NotEmpty(t, backup.Status.CompletionTimestamp)
		}).Return(backup, nil)
		clientMock.EXPECT().AddFinalizer(testCtx, backup, v1.BackupFinalizer).Return(nil, assert.AnError)

		v1Alpha1Client := newMockBackupV1Alpha1Interface(t)
		v1Alpha1Client.EXPECT().Backups(testNamespace).Return(clientMock)
		clientSetMock := newMockEcosystemInterface(t)
		clientSetMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Client)

		ownerReferenceBackupMock := newMockOwnerReferenceBackup(t)
		ownerReferenceBackupMock.EXPECT().BackupOwnerReferences(testCtx).Return(nil)

		sut := &backupCreateManager{recorder: recorderMock, clientSet: clientSetMock, namespace: testNamespace, ownerRefBackuper: ownerReferenceBackupMock}

		// when
		err := sut.create(testCtx, backup)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to set finalizer cloudogu-backup-finalizer to backup resource")
	})

	t.Run("should return error on adding app=ces label", func(t *testing.T) {
		// given
		backupName := "backup"
		backup := &v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: backupName, Namespace: testNamespace}}

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(backup, corev1.EventTypeNormal, v1.CreateEventReason, "Start backup process")
		clientMock := newMockEcosystemBackupInterface(t)
		clientMock.EXPECT().UpdateStatusInProgress(testCtx, backup).Return(backup, nil)
		clientMock.EXPECT().UpdateStatus(testCtx, backup, metav1.UpdateOptions{}).Run(func(ctx context.Context, backup *v1.Backup, opts metav1.UpdateOptions) {
			assert.NotEmpty(t, backup.Status.StartTimestamp)
		}).Return(backup, nil)
		clientMock.EXPECT().AddFinalizer(testCtx, backup, v1.BackupFinalizer).Return(backup, nil)
		clientMock.EXPECT().Get(testCtx, backup.Name, metav1.GetOptions{}).Return(backup, nil)
		clientMock.EXPECT().UpdateStatus(testCtx, backup, metav1.UpdateOptions{}).Run(func(ctx context.Context, backup *v1.Backup, opts metav1.UpdateOptions) {
			assert.NotEmpty(t, backup.Status.CompletionTimestamp)
		}).Return(backup, nil)
		clientMock.EXPECT().AddLabels(testCtx, backup).Return(nil, assert.AnError)

		v1Alpha1Client := newMockBackupV1Alpha1Interface(t)
		v1Alpha1Client.EXPECT().Backups(testNamespace).Return(clientMock)
		clientSetMock := newMockEcosystemInterface(t)
		clientSetMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Client)

		ownerReferenceBackupMock := newMockOwnerReferenceBackup(t)
		ownerReferenceBackupMock.EXPECT().BackupOwnerReferences(testCtx).Return(nil)

		sut := &backupCreateManager{recorder: recorderMock, clientSet: clientSetMock, namespace: testNamespace, ownerRefBackuper: ownerReferenceBackupMock}

		// when
		err := sut.create(testCtx, backup)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to add labels to backup resource")
	})

	t.Run("should return error activate maintenance mode error", func(t *testing.T) {
		// given
		backupName := "backup"
		backup := &v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: backupName, Namespace: testNamespace}}

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(backup, corev1.EventTypeNormal, v1.CreateEventReason, "Start backup process")
		clientMock := newMockEcosystemBackupInterface(t)
		clientMock.EXPECT().UpdateStatus(testCtx, backup, metav1.UpdateOptions{}).Run(func(ctx context.Context, backup *v1.Backup, opts metav1.UpdateOptions) {
			assert.NotEmpty(t, backup.Status.StartTimestamp)
		}).Return(backup, nil)
		clientMock.EXPECT().AddFinalizer(testCtx, backup, v1.BackupFinalizer).Return(backup, nil)
		clientMock.EXPECT().AddLabels(testCtx, backup).Return(backup, nil)
		clientMock.EXPECT().UpdateStatusInProgress(testCtx, backup).Return(backup, nil)
		clientMock.EXPECT().Get(testCtx, backup.Name, metav1.GetOptions{}).Return(backup, nil)
		clientMock.EXPECT().UpdateStatus(testCtx, backup, metav1.UpdateOptions{}).Run(func(ctx context.Context, backup *v1.Backup, opts metav1.UpdateOptions) {
			assert.NotEmpty(t, backup.Status.CompletionTimestamp)
		}).Return(backup, nil)
		maintenanceModeMock := NewMockMaintenanceModeSwitch(t)
		maintenanceModeMock.EXPECT().Activate(testCtx, repository.MaintenanceModeDescription{Title: "Service temporary unavailable", Text: "Backup in progress"}).Return(assert.AnError)

		v1Alpha1Client := newMockBackupV1Alpha1Interface(t)
		v1Alpha1Client.EXPECT().Backups(testNamespace).Return(clientMock)
		clientSetMock := newMockEcosystemInterface(t)
		clientSetMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Client)

		ownerReferenceBackupMock := newMockOwnerReferenceBackup(t)
		ownerReferenceBackupMock.EXPECT().BackupOwnerReferences(testCtx).Return(nil)

		sut := &backupCreateManager{recorder: recorderMock, clientSet: clientSetMock, maintenanceModeSwitch: maintenanceModeMock, namespace: testNamespace, ownerRefBackuper: ownerReferenceBackupMock}

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
		clientMock.EXPECT().UpdateStatus(testCtx, backup, metav1.UpdateOptions{}).Run(func(ctx context.Context, backup *v1.Backup, opts metav1.UpdateOptions) {
			assert.NotEmpty(t, backup.Status.StartTimestamp)
		}).Return(backup, nil)
		clientMock.EXPECT().AddFinalizer(testCtx, backup, v1.BackupFinalizer).Return(backup, nil)
		clientMock.EXPECT().AddLabels(testCtx, backup).Return(backup, nil)
		clientMock.EXPECT().UpdateStatusInProgress(testCtx, backup).Return(backup, nil)
		clientMock.EXPECT().UpdateStatusFailed(testCtx, backup).Return(backup, nil)
		clientMock.EXPECT().Get(testCtx, backup.Name, metav1.GetOptions{}).Return(backup, nil)
		clientMock.EXPECT().UpdateStatus(testCtx, backup, metav1.UpdateOptions{}).Run(func(ctx context.Context, backup *v1.Backup, opts metav1.UpdateOptions) {
			assert.NotEmpty(t, backup.Status.CompletionTimestamp)
		}).Return(backup, nil)
		maintenanceModeMock := NewMockMaintenanceModeSwitch(t)
		maintenanceModeMock.EXPECT().Activate(testCtx, repository.MaintenanceModeDescription{Title: "Service temporary unavailable", Text: "Backup in progress"}).Return(nil)
		maintenanceModeMock.EXPECT().Deactivate(testCtx).Return(nil)
		v1Alpha1Client := newMockBackupV1Alpha1Interface(t)
		v1Alpha1Client.EXPECT().Backups(testNamespace).Return(clientMock)
		clientSetMock := newMockEcosystemInterface(t)
		clientSetMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Client)

		ownerReferenceBackupMock := newMockOwnerReferenceBackup(t)
		ownerReferenceBackupMock.EXPECT().BackupOwnerReferences(testCtx).Return(nil)

		sut := &backupCreateManager{recorder: recorderMock, clientSet: clientSetMock, maintenanceModeSwitch: maintenanceModeMock, namespace: testNamespace, ownerRefBackuper: ownerReferenceBackupMock}

		// when
		err := sut.create(testCtx, backup)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to trigger backup provider: failed to get backup provider: unknown provider unknown123")
	})

	t.Run("should return error on provider readiness check", func(t *testing.T) {
		// given
		backupName := "backup"
		backup := &v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: backupName, Namespace: testNamespace}, Spec: v1.BackupSpec{Provider: "velero"}}

		providerMock := newMockBackupProvider(t)
		providerMock.EXPECT().CheckReady(testCtx).Return(assert.AnError)
		oldVeleroProvider := provider.NewVeleroProvider
		provider.NewVeleroProvider = func(client provider.K8sClient, ecosystemClientSet provider.EcosystemClientSet, recorder provider.EventRecorder, namespace string) provider.Provider {
			return providerMock
		}
		defer func() { provider.NewVeleroProvider = oldVeleroProvider }()

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(backup, corev1.EventTypeNormal, v1.CreateEventReason, "Start backup process")
		clientMock := newMockEcosystemBackupInterface(t)
		clientMock.EXPECT().UpdateStatus(testCtx, backup, metav1.UpdateOptions{}).Run(func(ctx context.Context, backup *v1.Backup, opts metav1.UpdateOptions) {
			assert.NotEmpty(t, backup.Status.StartTimestamp)
		}).Return(backup, nil)
		clientMock.EXPECT().AddFinalizer(testCtx, backup, v1.BackupFinalizer).Return(backup, nil)
		clientMock.EXPECT().AddLabels(testCtx, backup).Return(backup, nil)
		clientMock.EXPECT().UpdateStatusInProgress(testCtx, backup).Return(backup, nil)
		clientMock.EXPECT().UpdateStatusFailed(testCtx, backup).Return(backup, nil)
		clientMock.EXPECT().Get(testCtx, backup.Name, metav1.GetOptions{}).Return(backup, nil)
		clientMock.EXPECT().UpdateStatus(testCtx, backup, metav1.UpdateOptions{}).Run(func(ctx context.Context, backup *v1.Backup, opts metav1.UpdateOptions) {
			assert.NotEmpty(t, backup.Status.CompletionTimestamp)
		}).Return(backup, nil)
		maintenanceModeMock := NewMockMaintenanceModeSwitch(t)
		maintenanceModeMock.EXPECT().Activate(testCtx, repository.MaintenanceModeDescription{Title: "Service temporary unavailable", Text: "Backup in progress"}).Return(nil)
		maintenanceModeMock.EXPECT().Deactivate(testCtx).Return(nil)
		v1Alpha1Client := newMockBackupV1Alpha1Interface(t)
		v1Alpha1Client.EXPECT().Backups(testNamespace).Return(clientMock)
		clientSetMock := newMockEcosystemInterface(t)
		clientSetMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Client)

		ownerReferenceBackupMock := newMockOwnerReferenceBackup(t)
		ownerReferenceBackupMock.EXPECT().BackupOwnerReferences(testCtx).Return(nil)

		sut := &backupCreateManager{recorder: recorderMock, clientSet: clientSetMock, maintenanceModeSwitch: maintenanceModeMock, namespace: testNamespace, ownerRefBackuper: ownerReferenceBackupMock}

		// when
		err := sut.create(testCtx, backup)

		// then
		require.Error(t, err)
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
		provider.NewVeleroProvider = func(client provider.K8sClient, ecosystemClientSet provider.EcosystemClientSet, recorder provider.EventRecorder, namespace string) provider.Provider {
			return providerMock
		}
		defer func() { provider.NewVeleroProvider = oldVeleroProvider }()

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(backup, corev1.EventTypeNormal, v1.CreateEventReason, "Start backup process")
		clientMock := newMockEcosystemBackupInterface(t)
		clientMock.EXPECT().UpdateStatus(testCtx, backup, metav1.UpdateOptions{}).Run(func(ctx context.Context, backup *v1.Backup, opts metav1.UpdateOptions) {
			assert.NotEmpty(t, backup.Status.StartTimestamp)
		}).Return(backup, nil)
		clientMock.EXPECT().AddFinalizer(testCtx, backup, v1.BackupFinalizer).Return(backup, nil)
		clientMock.EXPECT().AddLabels(testCtx, backup).Return(backup, nil)
		clientMock.EXPECT().UpdateStatusInProgress(testCtx, backup).Return(backup, nil)
		clientMock.EXPECT().UpdateStatusFailed(testCtx, backup).Return(backup, nil)
		clientMock.EXPECT().Get(testCtx, backup.Name, metav1.GetOptions{}).Return(backup, nil)
		clientMock.EXPECT().UpdateStatus(testCtx, backup, metav1.UpdateOptions{}).Run(func(ctx context.Context, backup *v1.Backup, opts metav1.UpdateOptions) {
			assert.NotEmpty(t, backup.Status.CompletionTimestamp)
		}).Return(backup, nil)
		maintenanceModeMock := NewMockMaintenanceModeSwitch(t)
		maintenanceModeMock.EXPECT().Activate(testCtx, repository.MaintenanceModeDescription{Title: "Service temporary unavailable", Text: "Backup in progress"}).Return(nil)
		maintenanceModeMock.EXPECT().Deactivate(testCtx).Return(nil)
		v1Alpha1Client := newMockBackupV1Alpha1Interface(t)
		v1Alpha1Client.EXPECT().Backups(testNamespace).Return(clientMock)
		clientSetMock := newMockEcosystemInterface(t)
		clientSetMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Client)

		ownerReferenceBackupMock := newMockOwnerReferenceBackup(t)
		ownerReferenceBackupMock.EXPECT().BackupOwnerReferences(testCtx).Return(nil)

		sut := &backupCreateManager{recorder: recorderMock, clientSet: clientSetMock, maintenanceModeSwitch: maintenanceModeMock, namespace: testNamespace, ownerRefBackuper: ownerReferenceBackupMock}

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
		provider.NewVeleroProvider = func(client provider.K8sClient, ecosystemClientSet provider.EcosystemClientSet, recorder provider.EventRecorder, namespace string) provider.Provider {
			return providerMock
		}
		defer func() { provider.NewVeleroProvider = oldVeleroProvider }()

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(backup, corev1.EventTypeNormal, v1.CreateEventReason, "Start backup process")
		clientMock := newMockEcosystemBackupInterface(t)
		clientMock.EXPECT().UpdateStatus(testCtx, backup, metav1.UpdateOptions{}).Run(func(ctx context.Context, backup *v1.Backup, opts metav1.UpdateOptions) {
			assert.NotEmpty(t, backup.Status.StartTimestamp)
		}).Return(backup, nil)
		clientMock.EXPECT().AddFinalizer(testCtx, backup, v1.BackupFinalizer).Return(backup, nil)
		clientMock.EXPECT().AddLabels(testCtx, backup).Return(backup, nil)
		clientMock.EXPECT().UpdateStatusInProgress(testCtx, backup).Return(backup, nil)
		clientMock.EXPECT().Get(testCtx, backup.Name, metav1.GetOptions{}).Return(backup, nil)
		clientMock.EXPECT().UpdateStatus(testCtx, backup, metav1.UpdateOptions{}).Run(func(ctx context.Context, backup *v1.Backup, opts metav1.UpdateOptions) {
			assert.NotEmpty(t, backup.Status.CompletionTimestamp)
		}).Return(backup, nil)
		clientMock.EXPECT().UpdateStatusFailed(testCtx, backup).Return(nil, assert.AnError)
		maintenanceModeMock := NewMockMaintenanceModeSwitch(t)
		maintenanceModeMock.EXPECT().Activate(testCtx, repository.MaintenanceModeDescription{Title: "Service temporary unavailable", Text: "Backup in progress"}).Return(nil)
		maintenanceModeMock.EXPECT().Deactivate(testCtx).Return(nil)
		v1Alpha1Client := newMockBackupV1Alpha1Interface(t)
		v1Alpha1Client.EXPECT().Backups(testNamespace).Return(clientMock)
		clientSetMock := newMockEcosystemInterface(t)
		clientSetMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Client)

		ownerReferenceBackupMock := newMockOwnerReferenceBackup(t)
		ownerReferenceBackupMock.EXPECT().BackupOwnerReferences(testCtx).Return(nil)

		sut := &backupCreateManager{recorder: recorderMock, clientSet: clientSetMock, maintenanceModeSwitch: maintenanceModeMock, namespace: testNamespace, ownerRefBackuper: ownerReferenceBackupMock}

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
		provider.NewVeleroProvider = func(client provider.K8sClient, ecosystemClientSet provider.EcosystemClientSet, recorder provider.EventRecorder, namespace string) provider.Provider {
			return providerMock
		}
		defer func() { provider.NewVeleroProvider = oldVeleroProvider }()

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(backup, corev1.EventTypeNormal, v1.CreateEventReason, "Start backup process")
		clientMock := newMockEcosystemBackupInterface(t)
		clientMock.EXPECT().UpdateStatus(testCtx, backup, metav1.UpdateOptions{}).Run(func(ctx context.Context, backup *v1.Backup, opts metav1.UpdateOptions) {
			assert.NotEmpty(t, backup.Status.StartTimestamp)
		}).Return(backup, nil)
		clientMock.EXPECT().AddFinalizer(testCtx, backup, v1.BackupFinalizer).Return(backup, nil)
		clientMock.EXPECT().AddLabels(testCtx, backup).Return(backup, nil)
		clientMock.EXPECT().UpdateStatusInProgress(testCtx, backup).Return(backup, nil)
		clientMock.EXPECT().UpdateStatusCompleted(testCtx, backup).Return(nil, nil)
		clientMock.EXPECT().Get(testCtx, backup.Name, metav1.GetOptions{}).Return(backup, nil)
		clientMock.EXPECT().UpdateStatus(testCtx, backup, metav1.UpdateOptions{}).Run(func(ctx context.Context, backup *v1.Backup, opts metav1.UpdateOptions) {
			assert.NotEmpty(t, backup.Status.CompletionTimestamp)
		}).Return(backup, nil)
		maintenanceModeMock := NewMockMaintenanceModeSwitch(t)
		maintenanceModeMock.EXPECT().Activate(testCtx, repository.MaintenanceModeDescription{Title: "Service temporary unavailable", Text: "Backup in progress"}).Return(nil)
		maintenanceModeMock.EXPECT().Deactivate(testCtx).Return(assert.AnError)
		v1Alpha1Client := newMockBackupV1Alpha1Interface(t)
		v1Alpha1Client.EXPECT().Backups(testNamespace).Return(clientMock)
		clientSetMock := newMockEcosystemInterface(t)
		clientSetMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Client)

		ownerReferenceBackupMock := newMockOwnerReferenceBackup(t)
		ownerReferenceBackupMock.EXPECT().BackupOwnerReferences(testCtx).Return(nil)

		sut := &backupCreateManager{recorder: recorderMock, clientSet: clientSetMock, maintenanceModeSwitch: maintenanceModeMock, namespace: testNamespace, ownerRefBackuper: ownerReferenceBackupMock}

		// when
		err := sut.create(testCtx, backup)

		// then
		require.NoError(t, err)
	})

	t.Run("should log error on getting backup for setting completion timestamp", func(t *testing.T) {
		// given
		backupName := "backup"
		backup := &v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: backupName, Namespace: testNamespace}, Spec: v1.BackupSpec{Provider: "velero"}}

		providerMock := newMockBackupProvider(t)
		providerMock.EXPECT().CheckReady(testCtx).Return(nil)
		providerMock.EXPECT().CreateBackup(testCtx, backup).Return(nil)
		oldVeleroProvider := provider.NewVeleroProvider
		provider.NewVeleroProvider = func(client provider.K8sClient, ecosystemClientSet provider.EcosystemClientSet, recorder provider.EventRecorder, namespace string) provider.Provider {
			return providerMock
		}
		defer func() { provider.NewVeleroProvider = oldVeleroProvider }()

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(backup, corev1.EventTypeNormal, v1.CreateEventReason, "Start backup process")
		clientMock := newMockEcosystemBackupInterface(t)
		clientMock.EXPECT().UpdateStatus(testCtx, backup, metav1.UpdateOptions{}).Run(func(ctx context.Context, backup *v1.Backup, opts metav1.UpdateOptions) {
			assert.NotEmpty(t, backup.Status.StartTimestamp)
		}).Return(backup, nil)
		clientMock.EXPECT().AddFinalizer(testCtx, backup, v1.BackupFinalizer).Return(backup, nil)
		clientMock.EXPECT().AddLabels(testCtx, backup).Return(backup, nil)
		clientMock.EXPECT().UpdateStatusInProgress(testCtx, backup).Return(backup, nil)
		clientMock.EXPECT().UpdateStatusCompleted(testCtx, backup).Return(nil, nil)
		clientMock.EXPECT().Get(testCtx, backup.Name, metav1.GetOptions{}).Return(nil, assert.AnError)
		maintenanceModeMock := NewMockMaintenanceModeSwitch(t)
		maintenanceModeMock.EXPECT().Activate(testCtx, repository.MaintenanceModeDescription{Title: "Service temporary unavailable", Text: "Backup in progress"}).Return(nil)
		maintenanceModeMock.EXPECT().Deactivate(testCtx).Return(nil)
		v1Alpha1Client := newMockBackupV1Alpha1Interface(t)
		v1Alpha1Client.EXPECT().Backups(testNamespace).Return(clientMock)
		clientSetMock := newMockEcosystemInterface(t)
		clientSetMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Client)

		ownerReferenceBackupMock := newMockOwnerReferenceBackup(t)
		ownerReferenceBackupMock.EXPECT().BackupOwnerReferences(testCtx).Return(nil)

		sut := &backupCreateManager{recorder: recorderMock, clientSet: clientSetMock, maintenanceModeSwitch: maintenanceModeMock, namespace: testNamespace, ownerRefBackuper: ownerReferenceBackupMock}

		// when
		err := sut.create(testCtx, backup)

		// then
		require.NoError(t, err)
	})

	t.Run("should log error on updating completion timestamp", func(t *testing.T) {
		// given
		backupName := "backup"
		backup := &v1.Backup{ObjectMeta: metav1.ObjectMeta{Name: backupName, Namespace: testNamespace}, Spec: v1.BackupSpec{Provider: "velero"}}

		providerMock := newMockBackupProvider(t)
		providerMock.EXPECT().CheckReady(testCtx).Return(nil)
		providerMock.EXPECT().CreateBackup(testCtx, backup).Return(nil)
		oldVeleroProvider := provider.NewVeleroProvider
		provider.NewVeleroProvider = func(client provider.K8sClient, ecosystemClientSet provider.EcosystemClientSet, recorder provider.EventRecorder, namespace string) provider.Provider {
			return providerMock
		}
		defer func() { provider.NewVeleroProvider = oldVeleroProvider }()

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(backup, corev1.EventTypeNormal, v1.CreateEventReason, "Start backup process")
		clientMock := newMockEcosystemBackupInterface(t)
		clientMock.EXPECT().UpdateStatus(testCtx, backup, metav1.UpdateOptions{}).Run(func(ctx context.Context, backup *v1.Backup, opts metav1.UpdateOptions) {
			assert.NotEmpty(t, backup.Status.StartTimestamp)
		}).Return(backup, nil)
		clientMock.EXPECT().AddFinalizer(testCtx, backup, v1.BackupFinalizer).Return(backup, nil)
		clientMock.EXPECT().AddLabels(testCtx, backup).Return(backup, nil)
		clientMock.EXPECT().UpdateStatusInProgress(testCtx, backup).Return(backup, nil)
		clientMock.EXPECT().UpdateStatusCompleted(testCtx, backup).Return(nil, nil)
		clientMock.EXPECT().Get(testCtx, backup.Name, metav1.GetOptions{}).Return(backup, nil)
		clientMock.EXPECT().UpdateStatus(testCtx, backup, metav1.UpdateOptions{}).Run(func(ctx context.Context, backup *v1.Backup, opts metav1.UpdateOptions) {
			assert.NotEmpty(t, backup.Status.CompletionTimestamp)
		}).Return(nil, assert.AnError)
		maintenanceModeMock := NewMockMaintenanceModeSwitch(t)
		maintenanceModeMock.EXPECT().Activate(testCtx, repository.MaintenanceModeDescription{Title: "Service temporary unavailable", Text: "Backup in progress"}).Return(nil)
		maintenanceModeMock.EXPECT().Deactivate(testCtx).Return(nil)
		v1Alpha1Client := newMockBackupV1Alpha1Interface(t)
		v1Alpha1Client.EXPECT().Backups(testNamespace).Return(clientMock)
		clientSetMock := newMockEcosystemInterface(t)
		clientSetMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Client)

		ownerReferenceBackupMock := newMockOwnerReferenceBackup(t)
		ownerReferenceBackupMock.EXPECT().BackupOwnerReferences(testCtx).Return(nil)

		sut := &backupCreateManager{recorder: recorderMock, clientSet: clientSetMock, maintenanceModeSwitch: maintenanceModeMock, namespace: testNamespace, ownerRefBackuper: ownerReferenceBackupMock}

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
		provider.NewVeleroProvider = func(client provider.K8sClient, ecosystemClientSet provider.EcosystemClientSet, recorder provider.EventRecorder, namespace string) provider.Provider {
			return providerMock
		}
		defer func() { provider.NewVeleroProvider = oldVeleroProvider }()

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(backup, corev1.EventTypeNormal, v1.CreateEventReason, "Start backup process")
		clientMock := newMockEcosystemBackupInterface(t)
		clientMock.EXPECT().UpdateStatus(testCtx, backup, metav1.UpdateOptions{}).Run(func(ctx context.Context, backup *v1.Backup, opts metav1.UpdateOptions) {
			assert.NotEmpty(t, backup.Status.StartTimestamp)
		}).Return(backup, nil)
		clientMock.EXPECT().AddFinalizer(testCtx, backup, v1.BackupFinalizer).Return(backup, nil)
		clientMock.EXPECT().AddLabels(testCtx, backup).Return(backup, nil)
		clientMock.EXPECT().UpdateStatusInProgress(testCtx, backup).Return(backup, nil)
		clientMock.EXPECT().Get(testCtx, backup.Name, metav1.GetOptions{}).Return(backup, nil)
		clientMock.EXPECT().UpdateStatus(testCtx, backup, metav1.UpdateOptions{}).Run(func(ctx context.Context, backup *v1.Backup, opts metav1.UpdateOptions) {
			assert.NotEmpty(t, backup.Status.CompletionTimestamp)
		}).Return(backup, nil)
		clientMock.EXPECT().UpdateStatusCompleted(testCtx, backup).Return(nil, assert.AnError)
		maintenanceModeMock := NewMockMaintenanceModeSwitch(t)
		maintenanceModeMock.EXPECT().Activate(testCtx, repository.MaintenanceModeDescription{Title: "Service temporary unavailable", Text: "Backup in progress"}).Return(nil)
		maintenanceModeMock.EXPECT().Deactivate(testCtx).Return(nil)
		v1Alpha1Client := newMockBackupV1Alpha1Interface(t)
		v1Alpha1Client.EXPECT().Backups(testNamespace).Return(clientMock)
		clientSetMock := newMockEcosystemInterface(t)
		clientSetMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Client)

		ownerReferenceBackupMock := newMockOwnerReferenceBackup(t)
		ownerReferenceBackupMock.EXPECT().BackupOwnerReferences(testCtx).Return(nil)

		sut := &backupCreateManager{recorder: recorderMock, clientSet: clientSetMock, maintenanceModeSwitch: maintenanceModeMock, namespace: testNamespace, ownerRefBackuper: ownerReferenceBackupMock}

		// when
		err := sut.create(testCtx, backup)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to set status [completed] in backup resource")
	})
}
