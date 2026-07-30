package restore

import (
	"testing"

	v1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/k8s-backup-operator/internal/provider/velero"
	"github.com/cloudogu/k8s-backup-operator/pkg/provider"
	"github.com/cloudogu/k8s-registry-lib/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

// expectReadinessCheck installs a provider whose readiness check returns checkReadyErr, for tests
// that only care about the readiness gate and not about the rest of the provider.
func expectReadinessCheck(t *testing.T, checkReadyErr error) {
	providerMock := newMockRestoreProvider(t)
	providerMock.EXPECT().CheckReady(testCtx).Return(checkReadyErr)

	oldNewVeleroProvider := provider.NewVeleroProvider
	provider.NewVeleroProvider = func(_ provider.K8sClient, _ provider.EventRecorder, _ string) provider.Provider {
		return providerMock
	}
	t.Cleanup(func() { provider.NewVeleroProvider = oldNewVeleroProvider })
}

func Test_newCreateManager(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		clientMock := newMockK8sClient(t)

		// when
		manager := newCreateManager(clientMock, testNamespace, nil, nil, nil)

		// then
		require.NotEmpty(t, manager)
	})
}

func Test_defaultCreateManager_create(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		restore := &v1.Restore{ObjectMeta: metav1.ObjectMeta{Name: "restore", Namespace: testNamespace}, Spec: v1.RestoreSpec{BackupName: "backup", Provider: "velero"}}

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(restore, corev1.EventTypeNormal, v1.CreateEventReason, "Start restore process")

		providerMock := newMockRestoreProvider(t)
		providerMock.EXPECT().CheckReady(testCtx).Return(nil)
		providerMock.EXPECT().WaitForRestore(testCtx, matchesRestoreNamed(restore.Name)).Return(nil)
		providerMock.EXPECT().SyncBackups(testCtx).Return(nil)
		oldNewVeleroProvider := provider.NewVeleroProvider
		provider.NewVeleroProvider = func(client provider.K8sClient, recorder provider.EventRecorder, namespace string) provider.Provider {
			return providerMock
		}
		defer func() { provider.NewVeleroProvider = oldNewVeleroProvider }()

		maintenanceModeMock := newMockMaintenanceModeSwitch(t)
		maintenanceModeMock.EXPECT().Activate(testCtx, repository.MaintenanceModeDescription{Title: "Service temporary unavailable", Text: "Restore in progress"}, false).Return(nil)
		maintenanceModeMock.EXPECT().Deactivate(testCtx, false).Return(nil)

		cleanupMock := newMockCleanupManager(t)
		cleanupMock.EXPECT().Cleanup(testCtx).Return(nil)

		scaleMock := newMockScaleManager(t)
		scaleMock.EXPECT().ScaleDown(testCtx).Return(nil)
		scaleMock.EXPECT().ScaleUp(testCtx).Return(nil)

		parentClient := newTestClientWithParent(t, interceptor.Funcs{}, restore)

		sut := &defaultCreateManager{k8sClient: parentClient, recorder: recorderMock, maintenanceModeSwitch: maintenanceModeMock, cleanup: cleanupMock, scaleManager: scaleMock, namespace: testNamespace}

		// when
		err := sut.create(testCtx, restore)

		// then
		require.NoError(t, err)
		assertSuccessfulCondition(t, parentClient, restore.Name, metav1.ConditionTrue, ReasonRestoreCompleted)
	})

	t.Run("writes the parent only once when its status is already converged", func(t *testing.T) {
		restore := &v1.Restore{
			ObjectMeta: metav1.ObjectMeta{
				Name:       "restore",
				Namespace:  testNamespace,
				Finalizers: []string{v1.RestoreFinalizer},
				Labels:     restoreLabels(),
			},
			Spec:   v1.RestoreSpec{BackupName: "backup", Provider: "velero"},
			Status: v1.RestoreStatus{Status: v1.RestoreStatusInProgress},
		}

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(restore, corev1.EventTypeNormal, v1.CreateEventReason, "Start restore process")

		providerMock := newMockRestoreProvider(t)
		providerMock.EXPECT().CheckReady(testCtx).Return(nil)
		providerMock.EXPECT().WaitForRestore(testCtx, matchesRestoreNamed(restore.Name)).Return(nil)
		providerMock.EXPECT().SyncBackups(testCtx).Return(nil)
		oldNewVeleroProvider := provider.NewVeleroProvider
		provider.NewVeleroProvider = func(client provider.K8sClient, recorder provider.EventRecorder, namespace string) provider.Provider {
			return providerMock
		}
		defer func() { provider.NewVeleroProvider = oldNewVeleroProvider }()

		maintenanceModeMock := newMockMaintenanceModeSwitch(t)
		maintenanceModeMock.EXPECT().Activate(testCtx, repository.MaintenanceModeDescription{Title: "Service temporary unavailable", Text: "Restore in progress"}, false).Return(nil)
		maintenanceModeMock.EXPECT().Deactivate(testCtx, false).Return(nil)

		cleanupMock := newMockCleanupManager(t)
		cleanupMock.EXPECT().Cleanup(testCtx).Return(nil)

		scaleMock := newMockScaleManager(t)
		scaleMock.EXPECT().ScaleDown(testCtx).Return(nil)
		scaleMock.EXPECT().ScaleUp(testCtx).Return(nil)

		writes := &clientWrites{}
		parentClient := newTestClientWithParent(t, writes.interceptor(), restore)

		sut := &defaultCreateManager{k8sClient: parentClient, recorder: recorderMock, maintenanceModeSwitch: maintenanceModeMock, cleanup: cleanupMock, scaleManager: scaleMock, namespace: testNamespace}

		// when
		err := sut.create(testCtx, restore)

		// then
		require.NoError(t, err)
		assert.Equal(t, 0, writes.parent.updates, "the metadata is the metadata stage's, so create must not write the object itself")
		assert.Equal(t, 1, writes.parent.statusUpdates, "only the final Successful condition may be written")
		assertSuccessfulCondition(t, parentClient, restore.Name, metav1.ConditionTrue, ReasonRestoreCompleted)
	})

	t.Run("should fail to sync backups after restore", func(t *testing.T) {
		// given
		restore := &v1.Restore{ObjectMeta: metav1.ObjectMeta{Name: "restore", Namespace: testNamespace}, Spec: v1.RestoreSpec{BackupName: "backup", Provider: "velero"}}

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(restore, corev1.EventTypeNormal, v1.CreateEventReason, "Start restore process")

		providerMock := newMockRestoreProvider(t)
		providerMock.EXPECT().CheckReady(testCtx).Return(nil)
		providerMock.EXPECT().WaitForRestore(testCtx, matchesRestoreNamed(restore.Name)).Return(nil)
		providerMock.EXPECT().SyncBackups(testCtx).Return(assert.AnError)
		oldNewVeleroProvider := provider.NewVeleroProvider
		provider.NewVeleroProvider = func(client provider.K8sClient, recorder provider.EventRecorder, namespace string) provider.Provider {
			return providerMock
		}
		defer func() { provider.NewVeleroProvider = oldNewVeleroProvider }()

		maintenanceModeMock := newMockMaintenanceModeSwitch(t)
		maintenanceModeMock.EXPECT().Activate(testCtx, repository.MaintenanceModeDescription{Title: "Service temporary unavailable", Text: "Restore in progress"}, false).Return(nil)
		maintenanceModeMock.EXPECT().Deactivate(testCtx, false).Return(nil)

		cleanupMock := newMockCleanupManager(t)
		cleanupMock.EXPECT().Cleanup(testCtx).Return(nil)

		scaleMock := newMockScaleManager(t)
		scaleMock.EXPECT().ScaleDown(testCtx).Return(nil)

		sut := &defaultCreateManager{k8sClient: newTestClientWithParent(t, interceptor.Funcs{}, restore), recorder: recorderMock, maintenanceModeSwitch: maintenanceModeMock, cleanup: cleanupMock, scaleManager: scaleMock, namespace: testNamespace}

		// when
		err := sut.create(testCtx, restore)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to sync backups with provider")
	})

	t.Run("should return error before touching the restore when the provider is not ready", func(t *testing.T) {
		// given
		restore := &v1.Restore{ObjectMeta: metav1.ObjectMeta{Name: "restore", Namespace: testNamespace}, Spec: v1.RestoreSpec{BackupName: "backup", Provider: "velero"}}

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(restore, corev1.EventTypeNormal, v1.CreateEventReason, "Start restore process")

		expectReadinessCheck(t, assert.AnError)

		writes := &clientWrites{}
		parentClient := newTestClientWithParent(t, writes.interceptor(), restore)

		sut := &defaultCreateManager{k8sClient: parentClient, recorder: recorderMock, namespace: testNamespace}

		// when
		err := sut.create(testCtx, restore)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get restore provider [velero]: provider velero is not ready")
		assert.ErrorIs(t, err, assert.AnError)
		assert.Equal(t, 0, writes.total(), "an unready provider must leave the restore untouched")
	})

	t.Run("should return error on failing update status in progress", func(t *testing.T) {
		// given
		restore := &v1.Restore{ObjectMeta: metav1.ObjectMeta{Name: "restore", Namespace: testNamespace}, Spec: v1.RestoreSpec{BackupName: "backup", Provider: "velero"}}

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(restore, corev1.EventTypeNormal, v1.CreateEventReason, "Start restore process")

		expectReadinessCheck(t, nil)

		sut := &defaultCreateManager{k8sClient: newTestClientWithParent(t, failingStatusUpdate(assert.AnError), restore), recorder: recorderMock, namespace: testNamespace}

		// when
		err := sut.create(testCtx, restore)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to set status [in progress] in restore resource [restore]")
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("should continue with restore when failing ti activate maintenance mode", func(t *testing.T) {
		// given
		restore := &v1.Restore{ObjectMeta: metav1.ObjectMeta{Name: "restore", Namespace: testNamespace}, Spec: v1.RestoreSpec{BackupName: "backup", Provider: "velero"}}

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(restore, corev1.EventTypeNormal, v1.CreateEventReason, "Start restore process")

		providerMock := newMockRestoreProvider(t)
		providerMock.EXPECT().CheckReady(testCtx).Return(nil)
		providerMock.EXPECT().WaitForRestore(testCtx, matchesRestoreNamed(restore.Name)).Return(nil)
		providerMock.EXPECT().SyncBackups(testCtx).Return(nil)
		oldNewVeleroProvider := provider.NewVeleroProvider
		provider.NewVeleroProvider = func(client provider.K8sClient, recorder provider.EventRecorder, namespace string) provider.Provider {
			return providerMock
		}
		defer func() { provider.NewVeleroProvider = oldNewVeleroProvider }()

		maintenanceModeMock := newMockMaintenanceModeSwitch(t)
		maintenanceModeMock.EXPECT().Activate(testCtx, repository.MaintenanceModeDescription{Title: "Service temporary unavailable", Text: "Restore in progress"}, false).Return(assert.AnError)
		maintenanceModeMock.EXPECT().Deactivate(testCtx, false).Return(nil)

		cleanupMock := newMockCleanupManager(t)
		cleanupMock.EXPECT().Cleanup(testCtx).Return(nil)

		scaleMock := newMockScaleManager(t)
		scaleMock.EXPECT().ScaleDown(testCtx).Return(nil)
		scaleMock.EXPECT().ScaleUp(testCtx).Return(nil)

		parentClient := newTestClientWithParent(t, interceptor.Funcs{}, restore)

		sut := &defaultCreateManager{k8sClient: parentClient, recorder: recorderMock, maintenanceModeSwitch: maintenanceModeMock, cleanup: cleanupMock, scaleManager: scaleMock, namespace: testNamespace}

		// when
		err := sut.create(testCtx, restore)

		// then
		require.NoError(t, err)
		assertSuccessfulCondition(t, parentClient, restore.Name, metav1.ConditionTrue, ReasonRestoreCompleted)
	})

	t.Run("should return error on cleanup error", func(t *testing.T) {
		// given
		restore := &v1.Restore{ObjectMeta: metav1.ObjectMeta{Name: "restore", Namespace: testNamespace}, Spec: v1.RestoreSpec{BackupName: "backup", Provider: "velero"}}

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(restore, corev1.EventTypeNormal, v1.CreateEventReason, "Start restore process")

		expectReadinessCheck(t, nil)

		maintenanceModeMock := newMockMaintenanceModeSwitch(t)
		maintenanceModeMock.EXPECT().Activate(testCtx, repository.MaintenanceModeDescription{Title: "Service temporary unavailable", Text: "Restore in progress"}, false).Return(nil)
		maintenanceModeMock.EXPECT().Deactivate(testCtx, false).Return(nil)

		scaleManagerMock := newMockScaleManager(t)
		scaleManagerMock.EXPECT().ScaleDown(testCtx).Return(nil)

		cleanupMock := newMockCleanupManager(t)
		cleanupMock.EXPECT().Cleanup(testCtx).Return(assert.AnError)

		sut := &defaultCreateManager{k8sClient: newTestClientWithParent(t, interceptor.Funcs{}, restore), recorder: recorderMock, maintenanceModeSwitch: maintenanceModeMock, cleanup: cleanupMock, scaleManager: scaleManagerMock, namespace: testNamespace}

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

		providerMock := newMockRestoreProvider(t)
		providerMock.EXPECT().CheckReady(testCtx).Return(nil)
		providerMock.EXPECT().WaitForRestore(testCtx, matchesRestoreNamed(restore.Name)).Return(assert.AnError)
		oldNewVeleroProvider := provider.NewVeleroProvider
		provider.NewVeleroProvider = func(client provider.K8sClient, recorder provider.EventRecorder, namespace string) provider.Provider {
			return providerMock
		}
		defer func() { provider.NewVeleroProvider = oldNewVeleroProvider }()

		maintenanceModeMock := newMockMaintenanceModeSwitch(t)
		maintenanceModeMock.EXPECT().Activate(testCtx, repository.MaintenanceModeDescription{Title: "Service temporary unavailable", Text: "Restore in progress"}, false).Return(nil)
		maintenanceModeMock.EXPECT().Deactivate(testCtx, false).Return(nil)

		cleanupMock := newMockCleanupManager(t)
		cleanupMock.EXPECT().Cleanup(testCtx).Return(nil)

		scaleMock := newMockScaleManager(t)
		scaleMock.EXPECT().ScaleDown(testCtx).Return(nil)

		parentClient := newTestClientWithParent(t, interceptor.Funcs{}, restore)

		sut := &defaultCreateManager{k8sClient: parentClient, recorder: recorderMock, maintenanceModeSwitch: maintenanceModeMock, cleanup: cleanupMock, scaleManager: scaleMock, namespace: testNamespace}

		// when
		err := sut.create(testCtx, restore)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to trigger provider")
		assert.ErrorIs(t, err, assert.AnError)
		assertSuccessfulCondition(t, parentClient, restore.Name, metav1.ConditionFalse, ReasonProviderRestoreFailed)
	})

	t.Run("should wrap status error failing calling provider and update status", func(t *testing.T) {
		// given
		restore := &v1.Restore{ObjectMeta: metav1.ObjectMeta{Name: "restore", Namespace: testNamespace}, Spec: v1.RestoreSpec{BackupName: "backup", Provider: "velero"}}

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(restore, corev1.EventTypeNormal, v1.CreateEventReason, "Start restore process")

		providerMock := newMockRestoreProvider(t)
		providerMock.EXPECT().CheckReady(testCtx).Return(nil)
		providerMock.EXPECT().WaitForRestore(testCtx, matchesRestoreNamed(restore.Name)).Return(assert.AnError)
		oldNewVeleroProvider := provider.NewVeleroProvider
		provider.NewVeleroProvider = func(client provider.K8sClient, recorder provider.EventRecorder, namespace string) provider.Provider {
			return providerMock
		}
		defer func() { provider.NewVeleroProvider = oldNewVeleroProvider }()

		maintenanceModeMock := newMockMaintenanceModeSwitch(t)
		maintenanceModeMock.EXPECT().Activate(testCtx, repository.MaintenanceModeDescription{Title: "Service temporary unavailable", Text: "Restore in progress"}, false).Return(nil)
		maintenanceModeMock.EXPECT().Deactivate(testCtx, false).Return(nil)

		cleanupMock := newMockCleanupManager(t)
		cleanupMock.EXPECT().Cleanup(testCtx).Return(nil)

		scaleMock := newMockScaleManager(t)
		scaleMock.EXPECT().ScaleDown(testCtx).Return(nil)

		// the in-progress status still has to persist, only the terminal condition must fail
		parentClient := newTestClientWithParent(t, failingStatusUpdateFrom(2, assert.AnError), restore)

		sut := &defaultCreateManager{k8sClient: parentClient, recorder: recorderMock, maintenanceModeSwitch: maintenanceModeMock, cleanup: cleanupMock, scaleManager: scaleMock, namespace: testNamespace}

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

		providerMock := newMockRestoreProvider(t)
		providerMock.EXPECT().CheckReady(testCtx).Return(nil)
		providerMock.EXPECT().WaitForRestore(testCtx, matchesRestoreNamed(restore.Name)).Return(nil)
		providerMock.EXPECT().SyncBackups(testCtx).Return(nil)
		oldNewVeleroProvider := provider.NewVeleroProvider
		provider.NewVeleroProvider = func(client provider.K8sClient, recorder provider.EventRecorder, namespace string) provider.Provider {
			return providerMock
		}
		defer func() { provider.NewVeleroProvider = oldNewVeleroProvider }()

		maintenanceModeMock := newMockMaintenanceModeSwitch(t)
		maintenanceModeMock.EXPECT().Activate(testCtx, repository.MaintenanceModeDescription{Title: "Service temporary unavailable", Text: "Restore in progress"}, false).Return(nil)
		maintenanceModeMock.EXPECT().Deactivate(testCtx, false).Return(nil)

		cleanupMock := newMockCleanupManager(t)
		cleanupMock.EXPECT().Cleanup(testCtx).Return(nil)

		scaleMock := newMockScaleManager(t)
		scaleMock.EXPECT().ScaleDown(testCtx).Return(nil)
		scaleMock.EXPECT().ScaleUp(testCtx).Return(nil)

		// the in-progress status still has to persist, only the terminal condition must fail
		parentClient := newTestClientWithParent(t, failingStatusUpdateFrom(2, assert.AnError), restore)

		sut := &defaultCreateManager{k8sClient: parentClient, recorder: recorderMock, maintenanceModeSwitch: maintenanceModeMock, cleanup: cleanupMock, scaleManager: scaleMock, namespace: testNamespace}

		// when
		err := sut.create(testCtx, restore)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to set status [completed] in restore resource [restore]")
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("should return error on scaledown error", func(t *testing.T) {
		// given
		restore := &v1.Restore{ObjectMeta: metav1.ObjectMeta{Name: "restore", Namespace: testNamespace}, Spec: v1.RestoreSpec{BackupName: "backup", Provider: "velero"}}

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(restore, corev1.EventTypeNormal, v1.CreateEventReason, "Start restore process")

		expectReadinessCheck(t, nil)

		maintenanceModeMock := newMockMaintenanceModeSwitch(t)
		maintenanceModeMock.EXPECT().Activate(testCtx, repository.MaintenanceModeDescription{Title: "Service temporary unavailable", Text: "Restore in progress"}, false).Return(nil)
		maintenanceModeMock.EXPECT().Deactivate(testCtx, false).Return(nil)

		cleanupMock := newMockCleanupManager(t)

		scaleMock := newMockScaleManager(t)
		scaleMock.EXPECT().ScaleDown(testCtx).Return(assert.AnError)

		sut := &defaultCreateManager{k8sClient: newTestClientWithParent(t, interceptor.Funcs{}, restore), recorder: recorderMock, maintenanceModeSwitch: maintenanceModeMock, cleanup: cleanupMock, scaleManager: scaleMock, namespace: testNamespace}

		// when
		err := sut.create(testCtx, restore)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to scale down workloads before restore")
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("should return error on scaleup error", func(t *testing.T) {
		// given
		restore := &v1.Restore{ObjectMeta: metav1.ObjectMeta{Name: "restore", Namespace: testNamespace}, Spec: v1.RestoreSpec{BackupName: "backup", Provider: "velero"}}

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(restore, corev1.EventTypeNormal, v1.CreateEventReason, "Start restore process")

		providerMock := newMockRestoreProvider(t)
		providerMock.EXPECT().CheckReady(testCtx).Return(nil)
		providerMock.EXPECT().WaitForRestore(testCtx, matchesRestoreNamed(restore.Name)).Return(nil)
		providerMock.EXPECT().SyncBackups(testCtx).Return(nil)
		oldNewVeleroProvider := provider.NewVeleroProvider
		provider.NewVeleroProvider = func(client provider.K8sClient, recorder provider.EventRecorder, namespace string) provider.Provider {
			return providerMock
		}
		defer func() { provider.NewVeleroProvider = oldNewVeleroProvider }()

		maintenanceModeMock := newMockMaintenanceModeSwitch(t)
		maintenanceModeMock.EXPECT().Activate(testCtx, repository.MaintenanceModeDescription{Title: "Service temporary unavailable", Text: "Restore in progress"}, false).Return(nil)
		maintenanceModeMock.EXPECT().Deactivate(testCtx, false).Return(nil)

		cleanupMock := newMockCleanupManager(t)
		cleanupMock.EXPECT().Cleanup(testCtx).Return(nil)

		scaleMock := newMockScaleManager(t)
		scaleMock.EXPECT().ScaleDown(testCtx).Return(nil)
		scaleMock.EXPECT().ScaleUp(testCtx).Return(assert.AnError)

		sut := &defaultCreateManager{k8sClient: newTestClientWithParent(t, interceptor.Funcs{}, restore), recorder: recorderMock, maintenanceModeSwitch: maintenanceModeMock, cleanup: cleanupMock, scaleManager: scaleMock, namespace: testNamespace}

		// when
		err := sut.create(testCtx, restore)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to scale up workloads after restore")
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("reuses an existing own velero child without creating another one", func(t *testing.T) {
		// given
		restore := newParentRestore()

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(restore, corev1.EventTypeNormal, v1.CreateEventReason, "Start restore process")

		providerMock := newMockRestoreProvider(t)
		providerMock.EXPECT().CheckReady(testCtx).Return(nil)
		providerMock.EXPECT().WaitForRestore(testCtx, matchesRestoreNamed(restore.Name)).Return(nil)
		providerMock.EXPECT().SyncBackups(testCtx).Return(nil)
		oldNewVeleroProvider := provider.NewVeleroProvider
		provider.NewVeleroProvider = func(client provider.K8sClient, recorder provider.EventRecorder, namespace string) provider.Provider {
			return providerMock
		}
		defer func() { provider.NewVeleroProvider = oldNewVeleroProvider }()

		maintenanceModeMock := newMockMaintenanceModeSwitch(t)
		maintenanceModeMock.EXPECT().Activate(testCtx, repository.MaintenanceModeDescription{Title: "Service temporary unavailable", Text: "Restore in progress"}, false).Return(nil)
		maintenanceModeMock.EXPECT().Deactivate(testCtx, false).Return(nil)

		cleanupMock := newMockCleanupManager(t)
		cleanupMock.EXPECT().Cleanup(testCtx).Return(nil)

		scaleMock := newMockScaleManager(t)
		scaleMock.EXPECT().ScaleDown(testCtx).Return(nil)
		scaleMock.EXPECT().ScaleUp(testCtx).Return(nil)

		writes := &clientWrites{}
		k8sClient := newTestClientWithParent(t, writes.interceptor(), restore, velero.BuildRestore(restore))
		sut := &defaultCreateManager{k8sClient: k8sClient, recorder: recorderMock, maintenanceModeSwitch: maintenanceModeMock, cleanup: cleanupMock, scaleManager: scaleMock, namespace: testNamespace}

		// when
		err := sut.create(testCtx, restore)

		// then
		require.NoError(t, err)
		assert.Equal(t, 0, writes.child.total(), "the existing own child must be reused without any write")
		assertSuccessfulCondition(t, k8sClient, restore.Name, metav1.ConditionTrue, ReasonRestoreCompleted)
	})

	t.Run("fails terminally with a conflict on a foreign velero restore of the expected name", func(t *testing.T) {
		// given
		restore := newParentRestore()
		foreign := &velerov1.Restore{
			ObjectMeta: metav1.ObjectMeta{Name: restore.Name, Namespace: restore.Namespace},
			Spec:       velerov1.RestoreSpec{BackupName: restore.Spec.BackupName},
		}

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(restore, corev1.EventTypeNormal, v1.CreateEventReason, "Start restore process")
		recorderMock.EXPECT().Event(matchesRestoreNamed(restore.Name), corev1.EventTypeWarning, v1.ErrorOnCreateEventReason, mock.Anything)

		// the provider is never asked to wait, and SyncBackups and ScaleUp never run
		expectReadinessCheck(t, nil)

		maintenanceModeMock := newMockMaintenanceModeSwitch(t)
		maintenanceModeMock.EXPECT().Activate(testCtx, repository.MaintenanceModeDescription{Title: "Service temporary unavailable", Text: "Restore in progress"}, false).Return(nil)
		maintenanceModeMock.EXPECT().Deactivate(testCtx, false).Return(nil)

		cleanupMock := newMockCleanupManager(t)
		cleanupMock.EXPECT().Cleanup(testCtx).Return(nil)

		scaleMock := newMockScaleManager(t)
		scaleMock.EXPECT().ScaleDown(testCtx).Return(nil)

		writes := &clientWrites{}
		k8sClient := newTestClientWithParent(t, writes.interceptor(), restore, foreign)
		sut := &defaultCreateManager{k8sClient: k8sClient, recorder: recorderMock, maintenanceModeSwitch: maintenanceModeMock, cleanup: cleanupMock, scaleManager: scaleMock, namespace: testNamespace}

		// when
		err := sut.create(testCtx, restore)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to ensure the provider restore")
		var conflictErr *velero.ConflictError
		require.ErrorAs(t, err, &conflictErr)
		assert.Equal(t, 0, writes.child.total(), "the foreign velero restore must neither be deleted nor modified")
		persisted := &velerov1.Restore{}
		require.NoError(t, k8sClient.Get(testCtx, types.NamespacedName{Namespace: testNamespace, Name: restore.Name}, persisted))
		assert.Empty(t, persisted.OwnerReferences, "the foreign velero restore must never be claimed")
		assertSuccessfulCondition(t, k8sClient, restore.Name, metav1.ConditionFalse, ReasonProviderRestoreConflict)
	})
}
