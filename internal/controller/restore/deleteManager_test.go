package restore

import (
	"strings"
	"testing"

	v1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/k8s-backup-operator/internal/provider/velero"
	"github.com/cloudogu/k8s-backup-operator/pkg/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
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
		restore := newParentRestore()

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

		writes := &clientWrites{}
		k8sClient := newTestClient(t, writes.interceptor(), velero.BuildRestore(restore))
		sut := &defaultDeleteManager{k8sClient: k8sClient, clientSet: clientSetMock, namespace: testNamespace, recorder: recorderMock}

		// when
		err := sut.delete(testCtx, restore)

		// then
		require.NoError(t, err)
		assert.Equal(t, 1, writes.child.deletes)
		getErr := k8sClient.Get(testCtx, types.NamespacedName{Namespace: testNamespace, Name: testRestore}, &velerov1.Restore{})
		assert.True(t, apierrors.IsNotFound(getErr))
	})

	t.Run("tolerates an already deleted velero child", func(t *testing.T) {
		// given
		restore := newParentRestore()

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

		sut := &defaultDeleteManager{k8sClient: newTestClient(t, interceptor.Funcs{}), clientSet: clientSetMock, namespace: testNamespace, recorder: recorderMock}

		// when
		err := sut.delete(testCtx, restore)

		// then
		require.NoError(t, err)
	})

	t.Run("should return error on status update error", func(t *testing.T) {
		// given
		restore := newParentRestore()

		recorderMock := newMockEventRecorder(t)
		restoreClientMock := newMockEcosystemRestoreInterface(t)
		restoreClientMock.EXPECT().UpdateStatusDeleting(testCtx, restore).Return(nil, assert.AnError)

		v1Alpha1Client := newMockEcosystemV1Alpha1Interface(t)
		v1Alpha1Client.EXPECT().Restores(testNamespace).Return(restoreClientMock)
		clientSetMock := newMockEcosystemInterface(t)
		clientSetMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Client)

		sut := &defaultDeleteManager{k8sClient: newTestClient(t, interceptor.Funcs{}), clientSet: clientSetMock, namespace: testNamespace, recorder: recorderMock}

		// when
		err := sut.delete(testCtx, restore)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to update status [deleting] on restore [test-restore]")
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("should return error on velero child delete error", func(t *testing.T) {
		// given
		restore := newParentRestore()

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

		sut := &defaultDeleteManager{k8sClient: newTestClient(t, failingDelete(assert.AnError), velero.BuildRestore(restore)), clientSet: clientSetMock, recorder: recorderMock, namespace: testNamespace}

		// when
		err := sut.delete(testCtx, restore)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to delete restore")
		assert.ErrorContains(t, err, "failed to delete velero restore [test-restore]")
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("should return error on finalizer remove error", func(t *testing.T) {
		// given
		restore := newParentRestore()

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

		sut := &defaultDeleteManager{k8sClient: newTestClient(t, interceptor.Funcs{}, velero.BuildRestore(restore)), clientSet: clientSetMock, recorder: recorderMock, namespace: testNamespace}

		// when
		err := sut.delete(testCtx, restore)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to delete finalizer [cloudogu-restore-finalizer]")
		assert.ErrorIs(t, err, assert.AnError)
	})
}

func Test_defaultDeleteManager_deleteOnlyDeletesItsOwnProviderRestore(t *testing.T) {
	legacyCompleted := func() *v1.Restore {
		restore := newParentRestore()
		restore.Status.Status = v1.RestoreStatusCompleted // written before conditions existed

		return restore
	}
	reportedConflict := func() *v1.Restore {
		restore := newParentRestore()
		restore.Status.Conditions = []metav1.Condition{{
			Type:               v1.ConditionSuccessful,
			Status:             metav1.ConditionFalse,
			Reason:             ReasonProviderRestoreConflict,
			LastTransitionTime: metav1.Now(),
		}}

		return restore
	}
	unownedChild := func() *velerov1.Restore {
		return &velerov1.Restore{
			ObjectMeta: metav1.ObjectMeta{Name: testRestore, Namespace: testNamespace},
			Spec:       velerov1.RestoreSpec{BackupName: testBackup},
		}
	}
	foreignlyOwnedChild := func() *velerov1.Restore {
		foreignParent := newParentRestore()
		foreignParent.UID = "22222222-2222-2222-2222-222222222222"

		return velero.BuildRestore(foreignParent)
	}

	tests := map[string]struct {
		restore     *v1.Restore
		child       *velerov1.Restore
		wantDeleted bool
	}{
		"our own child is deleted": {
			restore:     newParentRestore(),
			child:       velero.BuildRestore(newParentRestore()),
			wantDeleted: true,
		},
		"the child of a restore created before conditions existed is deleted": {
			restore:     legacyCompleted(),
			child:       unownedChild(),
			wantDeleted: true,
		},
		"a namesake this restore was reported as conflicting with survives": {
			restore:     reportedConflict(),
			child:       unownedChild(),
			wantDeleted: false,
		},
		"a namesake of a restore that never ran survives": {
			restore:     newParentRestore(),
			child:       unownedChild(),
			wantDeleted: false,
		},
		"a child controlled by another restore survives": {
			restore:     newParentRestore(),
			child:       foreignlyOwnedChild(),
			wantDeleted: false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			// given
			recorderMock := newMockEventRecorder(t)
			if !test.wantDeleted {
				recorderMock.EXPECT().Event(test.restore, corev1.EventTypeWarning, v1.DeleteEventReason, mock.MatchedBy(func(message string) bool {
					return strings.Contains(message, "not owned by this restore")
				}))
			}

			restoreClientMock := newMockEcosystemRestoreInterface(t)
			restoreClientMock.EXPECT().UpdateStatusDeleting(testCtx, test.restore).Return(test.restore, nil)
			restoreClientMock.EXPECT().RemoveFinalizer(testCtx, test.restore, "cloudogu-restore-finalizer").Return(nil, nil)

			providerMock := newMockRestoreProvider(t)
			providerMock.EXPECT().CheckReady(testCtx).Return(nil)
			oldVeleroProviderGetter := provider.NewVeleroProvider
			provider.NewVeleroProvider = func(client provider.K8sClient, recorder provider.EventRecorder, namespace string) provider.Provider {
				return providerMock
			}
			t.Cleanup(func() { provider.NewVeleroProvider = oldVeleroProviderGetter })

			v1Alpha1Client := newMockEcosystemV1Alpha1Interface(t)
			v1Alpha1Client.EXPECT().Restores(testNamespace).Return(restoreClientMock)
			clientSetMock := newMockEcosystemInterface(t)
			clientSetMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Client)

			writes := &clientWrites{}
			k8sClient := newTestClient(t, writes.interceptor(), test.child)
			sut := &defaultDeleteManager{k8sClient: k8sClient, clientSet: clientSetMock, namespace: testNamespace, recorder: recorderMock}

			// when
			err := sut.delete(testCtx, test.restore)

			// then the finalizer is removed either way, so deletion never wedges on a foreign child
			require.NoError(t, err)
			getErr := k8sClient.Get(testCtx, types.NamespacedName{Namespace: testNamespace, Name: testRestore}, &velerov1.Restore{})
			if test.wantDeleted {
				assert.Equal(t, 1, writes.child.deletes)
				assert.True(t, apierrors.IsNotFound(getErr), "the own child must be gone")
			} else {
				assert.Equal(t, 0, writes.total(), "a child that is not ours must not be written to at all")
				assert.NoError(t, getErr, "the foreign child must survive its namesake")
			}
		})
	}
}
