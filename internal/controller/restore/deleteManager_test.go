package restore

import (
	"context"
	"testing"

	v1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/k8s-backup-operator/internal/provider/velero"
	"github.com/cloudogu/k8s-backup-operator/pkg/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

// newFailingDeleteClient returns a client whose Delete always fails, to cover the
// non-NotFound error path of the typed child deletion.
func newFailingDeleteClient(t *testing.T) client.WithWatch {
	t.Helper()

	testScheme := runtime.NewScheme()
	require.NoError(t, velerov1.AddToScheme(testScheme))

	return fake.NewClientBuilder().WithScheme(testScheme).WithInterceptorFuncs(interceptor.Funcs{
		Delete: func(_ context.Context, _ client.WithWatch, _ client.Object, _ ...client.DeleteOption) error {
			return assert.AnError
		},
	}).Build()
}

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

		oldVeleroProviderGetter := provider.NewVeleroProvider
		provider.NewVeleroProvider = func(client provider.K8sClient, recorder provider.EventRecorder, namespace string) provider.Provider {
			return providerMock
		}
		defer func() { provider.NewVeleroProvider = oldVeleroProviderGetter }()

		v1Alpha1Client := newMockEcosystemV1Alpha1Interface(t)
		v1Alpha1Client.EXPECT().Restores(testNamespace).Return(restoreClientMock)
		clientSetMock := newMockEcosystemInterface(t)
		clientSetMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Client)

		writes := &childWriteCounter{}
		k8sClient := newChildTestClient(t, writes, velero.BuildRestore(restore))
		sut := &defaultDeleteManager{k8sClient: k8sClient, clientSet: clientSetMock, namespace: testNamespace, recorder: recorderMock}

		// when
		err := sut.delete(testCtx, restore)

		// then
		require.NoError(t, err)
		assert.Equal(t, 1, writes.deletes)
		getErr := k8sClient.Get(testCtx, types.NamespacedName{Namespace: testNamespace, Name: "restore"}, &velerov1.Restore{})
		assert.True(t, apierrors.IsNotFound(getErr))
	})

	t.Run("tolerates an already deleted velero child", func(t *testing.T) {
		// given
		restore := &v1.Restore{ObjectMeta: metav1.ObjectMeta{Name: "restore", Namespace: testNamespace}, Spec: v1.RestoreSpec{BackupName: "backup", Provider: "velero"}}

		recorderMock := newMockEventRecorder(t)
		restoreClientMock := newMockEcosystemRestoreInterface(t)
		restoreClientMock.EXPECT().UpdateStatusDeleting(testCtx, restore).Return(restore, nil)
		restoreClientMock.EXPECT().RemoveFinalizer(testCtx, restore, "cloudogu-restore-finalizer").Return(nil, nil)

		providerMock := newMockRestoreProvider(t)
		providerMock.EXPECT().CheckReady(testCtx).Return(nil)

		oldVeleroProviderGetter := provider.NewVeleroProvider
		provider.NewVeleroProvider = func(client provider.K8sClient, recorder provider.EventRecorder, namespace string) provider.Provider {
			return providerMock
		}
		defer func() { provider.NewVeleroProvider = oldVeleroProviderGetter }()

		v1Alpha1Client := newMockEcosystemV1Alpha1Interface(t)
		v1Alpha1Client.EXPECT().Restores(testNamespace).Return(restoreClientMock)
		clientSetMock := newMockEcosystemInterface(t)
		clientSetMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Client)

		sut := &defaultDeleteManager{k8sClient: newChildTestClient(t, &childWriteCounter{}), clientSet: clientSetMock, namespace: testNamespace, recorder: recorderMock}

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

	t.Run("should return error on velero child delete error", func(t *testing.T) {
		// given
		restore := &v1.Restore{ObjectMeta: metav1.ObjectMeta{Name: "restore", Namespace: testNamespace}, Spec: v1.RestoreSpec{BackupName: "backup", Provider: "velero"}}

		recorderMock := newMockEventRecorder(t)
		restoreClientMock := newMockEcosystemRestoreInterface(t)
		restoreClientMock.EXPECT().UpdateStatusDeleting(testCtx, restore).Return(restore, nil)

		providerMock := newMockRestoreProvider(t)
		providerMock.EXPECT().CheckReady(testCtx).Return(nil)

		oldVeleroProviderGetter := provider.NewVeleroProvider
		provider.NewVeleroProvider = func(client provider.K8sClient, recorder provider.EventRecorder, namespace string) provider.Provider {
			return providerMock
		}
		defer func() { provider.NewVeleroProvider = oldVeleroProviderGetter }()
		v1Alpha1Client := newMockEcosystemV1Alpha1Interface(t)
		v1Alpha1Client.EXPECT().Restores(testNamespace).Return(restoreClientMock)
		clientSetMock := newMockEcosystemInterface(t)
		clientSetMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Client)

		sut := &defaultDeleteManager{k8sClient: newFailingDeleteClient(t), clientSet: clientSetMock, recorder: recorderMock, namespace: testNamespace}

		// when
		err := sut.delete(testCtx, restore)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to delete restore")
		assert.ErrorContains(t, err, "failed to delete velero restore [restore]")
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

		oldVeleroProviderGetter := provider.NewVeleroProvider
		provider.NewVeleroProvider = func(client provider.K8sClient, recorder provider.EventRecorder, namespace string) provider.Provider {
			return providerMock
		}
		defer func() { provider.NewVeleroProvider = oldVeleroProviderGetter }()
		v1Alpha1Client := newMockEcosystemV1Alpha1Interface(t)
		v1Alpha1Client.EXPECT().Restores(testNamespace).Return(restoreClientMock)
		clientSetMock := newMockEcosystemInterface(t)
		clientSetMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Client)

		sut := &defaultDeleteManager{k8sClient: newChildTestClient(t, &childWriteCounter{}, velero.BuildRestore(restore)), clientSet: clientSetMock, recorder: recorderMock, namespace: testNamespace}

		// when
		err := sut.delete(testCtx, restore)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to delete finalizer [cloudogu-restore-finalizer]")
		assert.ErrorIs(t, err, assert.AnError)
	})
}
