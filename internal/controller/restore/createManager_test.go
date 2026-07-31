package restore

import (
	"testing"

	v1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/k8s-backup-operator/pkg/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

// installProvider makes restoreprovider.Get return the given provider instead of a real one.
func installProvider(t *testing.T, providerMock *mockRestoreProvider) {
	t.Helper()

	oldNewVeleroProvider := provider.NewVeleroProvider
	provider.NewVeleroProvider = func(_ provider.K8sClient, _ provider.EventRecorder, _ string) provider.Provider {
		return providerMock
	}
	t.Cleanup(func() { provider.NewVeleroProvider = oldNewVeleroProvider })
}

// expectReadinessCheck installs a provider whose readiness check returns checkReadyErr
func expectReadinessCheck(t *testing.T, checkReadyErr error) {
	providerMock := newMockRestoreProvider(t)
	providerMock.EXPECT().CheckReady(testCtx).Return(checkReadyErr)

	installProvider(t, providerMock)
}

// expectBackupSynchronization installs a ready provider whose backup synchronization returns syncErr.
func expectBackupSynchronization(t *testing.T, syncErr error) {
	providerMock := newMockRestoreProvider(t)
	providerMock.EXPECT().CheckReady(testCtx).Return(nil)
	providerMock.EXPECT().SyncBackups(testCtx).Return(syncErr)

	installProvider(t, providerMock)
}

func Test_newCreateManager(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		clientMock := newMockK8sClient(t)

		// when
		manager := newCreateManager(clientMock, testNamespace, nil, nil)

		// then
		require.NotEmpty(t, manager)
	})
}

func Test_defaultCreateManager_create(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		restore := withPreparation(withInitializedConditions(withMetadata(newParentRestore())))

		expectBackupSynchronization(t, nil)

		maintenanceModeMock := newMockMaintenanceModeSwitch(t)
		maintenanceModeMock.EXPECT().Deactivate(testCtx, false).Return(nil)

		scaleMock := newMockScaleManager(t)
		scaleMock.EXPECT().ScaleUp(testCtx).Return(nil)

		writes := &clientWrites{}
		parentClient := newTestClientWithParent(t, writes.interceptor(), restore)

		sut := &defaultCreateManager{k8sClient: parentClient, recorder: newMockEventRecorder(t), maintenanceModeSwitch: maintenanceModeMock, scaleManager: scaleMock, namespace: testNamespace}

		// when
		err := sut.create(testCtx, restore)

		// then
		require.NoError(t, err)
		assert.Equal(t, 0, writes.parent.updates, "the metadata is the metadata stage's, so create must not write the object itself")
		assert.Equal(t, 1, writes.parent.statusUpdates, "the milestones the manager reaches must be written together")
		assert.Equal(t, 0, writes.child.total(), "the child belongs to the provider, so the manager must not touch it")
		assertPersistedCondition(t, parentClient, v1.ConditionBackupsSynchronized, metav1.ConditionTrue, ReasonBackupSynchronizationCompleted)
		assertPersistedCondition(t, parentClient, v1.ConditionWorkloadsRecovered, metav1.ConditionTrue, ReasonWorkloadRecoveryCompleted)
		assertSuccessfulCondition(t, parentClient, restore.Name, metav1.ConditionTrue, ReasonRestoreCompleted)
	})

	t.Run("should fail to sync backups after restore", func(t *testing.T) {
		// given
		restore := withPreparation(withInitializedConditions(withMetadata(newParentRestore())))

		expectBackupSynchronization(t, assert.AnError)

		maintenanceModeMock := newMockMaintenanceModeSwitch(t)
		maintenanceModeMock.EXPECT().Deactivate(testCtx, false).Return(nil)

		// the workloads must not be scaled up when the synchronization failed
		scaleMock := newMockScaleManager(t)

		sut := &defaultCreateManager{k8sClient: newTestClientWithParent(t, interceptor.Funcs{}, restore), recorder: newMockEventRecorder(t), maintenanceModeSwitch: maintenanceModeMock, scaleManager: scaleMock, namespace: testNamespace}

		// when
		err := sut.create(testCtx, restore)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to sync backups with provider")
	})

	t.Run("should return error before touching the restore when the provider is not ready", func(t *testing.T) {
		// given
		restore := withPreparation(withInitializedConditions(withMetadata(newParentRestore())))

		expectReadinessCheck(t, assert.AnError)

		writes := &clientWrites{}
		parentClient := newTestClientWithParent(t, writes.interceptor(), restore)

		sut := &defaultCreateManager{k8sClient: parentClient, recorder: newMockEventRecorder(t), namespace: testNamespace}

		// when
		err := sut.create(testCtx, restore)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get restore provider [velero]: provider velero is not ready")
		assert.ErrorIs(t, err, assert.AnError)
		assert.Equal(t, 0, writes.total(), "an unready provider must leave the restore untouched")
	})

	t.Run("should return error on failing setting completed status", func(t *testing.T) {
		// given
		restore := withPreparation(withInitializedConditions(withMetadata(newParentRestore())))

		expectBackupSynchronization(t, nil)

		maintenanceModeMock := newMockMaintenanceModeSwitch(t)
		maintenanceModeMock.EXPECT().Deactivate(testCtx, false).Return(nil)

		scaleMock := newMockScaleManager(t)
		scaleMock.EXPECT().ScaleUp(testCtx).Return(nil)

		parentClient := newTestClientWithParent(t, failingStatusUpdate(assert.AnError), restore)

		sut := &defaultCreateManager{k8sClient: parentClient, recorder: newMockEventRecorder(t), maintenanceModeSwitch: maintenanceModeMock, scaleManager: scaleMock, namespace: testNamespace}

		// when
		err := sut.create(testCtx, restore)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to set status [completed] in restore resource [test-restore]")
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("should return error on scaleup error", func(t *testing.T) {
		// given
		restore := withPreparation(withInitializedConditions(withMetadata(newParentRestore())))

		expectBackupSynchronization(t, nil)

		maintenanceModeMock := newMockMaintenanceModeSwitch(t)
		maintenanceModeMock.EXPECT().Deactivate(testCtx, false).Return(nil)

		scaleMock := newMockScaleManager(t)
		scaleMock.EXPECT().ScaleUp(testCtx).Return(assert.AnError)

		sut := &defaultCreateManager{k8sClient: newTestClientWithParent(t, interceptor.Funcs{}, restore), recorder: newMockEventRecorder(t), maintenanceModeSwitch: maintenanceModeMock, scaleManager: scaleMock, namespace: testNamespace}

		// when
		err := sut.create(testCtx, restore)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to scale up workloads after restore")
		assert.ErrorIs(t, err, assert.AnError)
	})
}
