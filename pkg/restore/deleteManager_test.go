package restore

import (
	v1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
	"github.com/cloudogu/k8s-backup-operator/pkg/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func Test_defaultDeleteManager_delete(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		recorderMock := newMockEventRecorder(t)
		clientSetMock := newMockEcosystemInterface(t)
		clientMock := newMockK8sClient(t)

		// when
		manager := newDeleteManager(clientMock, clientSetMock, testNamespace, recorderMock)

		// then
		require.NotEmpty(t, manager)
	})
}

func Test_newDeleteManager(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		restore := &v1.Restore{ObjectMeta: metav1.ObjectMeta{Name: "restore", Namespace: testNamespace}, Spec: v1.RestoreSpec{BackupName: "backup", Provider: "velero"}}

		recorderMock := newMockEventRecorder(t)
		restoreClientMock := newMockEcosystemRestoreInterface(t)
		restoreClientMock.EXPECT().UpdateStatusDeleting(testCtx, restore).Return(restore, nil)
		restoreClientMock.EXPECT().RemoveFinalizer(testCtx, restore, "cloudogu-restore-finalizer").Return(nil, nil)

		providerMock := newMockRestoreProvider(t)
		providerMock.EXPECT().CheckReady(testCtx).Return(nil)
		providerMock.EXPECT().DeleteRestore(testCtx, restore).Return(nil)

		oldVeleroProviderGetter := provider.NewVeleroProvider
		provider.NewVeleroProvider = func(client provider.K8sClient, recorder provider.EventRecorder, namespace string) (provider.Provider, error) {
			return providerMock, nil
		}
		defer func() { provider.NewVeleroProvider = oldVeleroProviderGetter }()

		v1Alpha1Client := newMockEcosystemV1Alpha1Interface(t)
		v1Alpha1Client.EXPECT().Restores(testNamespace).Return(restoreClientMock)
		clientSetMock := newMockEcosystemInterface(t)
		clientSetMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Client)

		sut := &defaultDeleteManager{clientSet: clientSetMock, namespace: testNamespace, recorder: recorderMock}

		// when
		err := sut.delete(testCtx, restore)

		// then
		require.NoError(t, err)
	})

	t.Run("should return error on status update error", func(t *testing.T) {
		// given
		restore := &v1.Restore{ObjectMeta: metav1.ObjectMeta{Name: "restore", Namespace: testNamespace}, Spec: v1.RestoreSpec{BackupName: "backup", Provider: "velero"}}

		recorderMock := newMockEventRecorder(t)
		restoreClientMock := newMockEcosystemRestoreInterface(t)
		restoreClientMock.EXPECT().UpdateStatusDeleting(testCtx, restore).Return(nil, assert.AnError)

		v1Alpha1Client := newMockEcosystemV1Alpha1Interface(t)
		v1Alpha1Client.EXPECT().Restores(testNamespace).Return(restoreClientMock)
		clientSetMock := newMockEcosystemInterface(t)
		clientSetMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Client)

		sut := &defaultDeleteManager{clientSet: clientSetMock, namespace: testNamespace, recorder: recorderMock}

		// when
		err := sut.delete(testCtx, restore)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to update status [deleting] on restore [restore]")
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("should return error on provider delete error", func(t *testing.T) {
		// given
		restore := &v1.Restore{ObjectMeta: metav1.ObjectMeta{Name: "restore", Namespace: testNamespace}, Spec: v1.RestoreSpec{BackupName: "backup", Provider: "velero"}}

		recorderMock := newMockEventRecorder(t)
		restoreClientMock := newMockEcosystemRestoreInterface(t)
		restoreClientMock.EXPECT().UpdateStatusDeleting(testCtx, restore).Return(restore, nil)

		providerMock := newMockRestoreProvider(t)
		providerMock.EXPECT().CheckReady(testCtx).Return(nil)
		providerMock.EXPECT().DeleteRestore(testCtx, restore).Return(assert.AnError)

		oldVeleroProviderGetter := provider.NewVeleroProvider
		provider.NewVeleroProvider = func(client provider.K8sClient, recorder provider.EventRecorder, namespace string) (provider.Provider, error) {
			return providerMock, nil
		}
		defer func() { provider.NewVeleroProvider = oldVeleroProviderGetter }()
		v1Alpha1Client := newMockEcosystemV1Alpha1Interface(t)
		v1Alpha1Client.EXPECT().Restores(testNamespace).Return(restoreClientMock)
		clientSetMock := newMockEcosystemInterface(t)
		clientSetMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Client)

		sut := &defaultDeleteManager{clientSet: clientSetMock, recorder: recorderMock, namespace: testNamespace}

		// when
		err := sut.delete(testCtx, restore)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to delete restore")
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("should return error on finalizer remove error", func(t *testing.T) {
		// given
		restore := &v1.Restore{ObjectMeta: metav1.ObjectMeta{Name: "restore", Namespace: testNamespace}, Spec: v1.RestoreSpec{BackupName: "backup", Provider: "velero"}}

		recorderMock := newMockEventRecorder(t)
		restoreClientMock := newMockEcosystemRestoreInterface(t)
		restoreClientMock.EXPECT().UpdateStatusDeleting(testCtx, restore).Return(restore, nil)
		restoreClientMock.EXPECT().RemoveFinalizer(testCtx, restore, "cloudogu-restore-finalizer").Return(nil, assert.AnError)

		providerMock := newMockRestoreProvider(t)
		providerMock.EXPECT().CheckReady(testCtx).Return(nil)
		providerMock.EXPECT().DeleteRestore(testCtx, restore).Return(nil)

		oldVeleroProviderGetter := provider.NewVeleroProvider
		provider.NewVeleroProvider = func(client provider.K8sClient, recorder provider.EventRecorder, namespace string) (provider.Provider, error) {
			return providerMock, nil
		}
		defer func() { provider.NewVeleroProvider = oldVeleroProviderGetter }()
		v1Alpha1Client := newMockEcosystemV1Alpha1Interface(t)
		v1Alpha1Client.EXPECT().Restores(testNamespace).Return(restoreClientMock)
		clientSetMock := newMockEcosystemInterface(t)
		clientSetMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Client)

		sut := &defaultDeleteManager{clientSet: clientSetMock, recorder: recorderMock, namespace: testNamespace}

		// when
		err := sut.delete(testCtx, restore)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to delete finalizer [cloudogu-restore-finalizer]")
		assert.ErrorIs(t, err, assert.AnError)
	})
}
