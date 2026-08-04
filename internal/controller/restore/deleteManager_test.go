package restore

import (
	"context"
	"strings"
	"testing"
	"time"

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
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

// deletableParent returns the parent as the delete manager actually sees it: marked for deletion and
// still held by the finalizer. Without the finalizer its removal would be a no-op and the tests would
// pass for the wrong reason; without the deletion timestamp the derived status phase would not be
// "deleting" and the status write would be a no-op.
func deletableParent() *v1.Restore {
	restore := newParentRestore()
	restore.Finalizers = []string{v1.RestoreFinalizer}
	restore.DeletionTimestamp = &metav1.Time{Time: time.Now()}

	return restore
}

// assertParentDeleted asserts that removing the finalizer let the deletion of the parent finish.
func assertParentDeleted(t *testing.T, testClient client.Client, name string) {
	t.Helper()

	err := testClient.Get(context.Background(), client.ObjectKey{Namespace: testNamespace, Name: name}, &v1.Restore{})
	assert.True(t, apierrors.IsNotFound(err), "the parent must be gone once its finalizer is removed, got %v", err)
}

func Test_defaultDeleteManager_delete(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		recorderMock := newMockEventRecorder(t)
		clientMock := newMockK8sClient(t)

		// when
		manager := newDeleteManager(clientMock, testNamespace, recorderMock)

		// then
		require.NotEmpty(t, manager)
	})
}

func Test_newDeleteManager(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		restore := deletableParent()

		recorderMock := newMockEventRecorder(t)

		providerMock := newMockRestoreProvider(t)
		providerMock.EXPECT().CheckReady(testCtx).Return(nil)

		oldVeleroProviderGetter := provider.NewVeleroProvider
		provider.NewVeleroProvider = func(client provider.K8sClient, recorder provider.EventRecorder, namespace string) provider.Provider {
			return providerMock
		}
		defer func() { provider.NewVeleroProvider = oldVeleroProviderGetter }()

		writes := &clientWrites{}
		k8sClient := newTestClientWithParent(t, writes.interceptor(), restore, velero.BuildRestore(restore))
		sut := &defaultDeleteManager{k8sClient: k8sClient, namespace: testNamespace, recorder: recorderMock}

		// when
		err := sut.delete(testCtx, restore)

		// then
		require.NoError(t, err)
		assert.Equal(t, 1, writes.child.deletes)
		getErr := k8sClient.Get(testCtx, types.NamespacedName{Namespace: testNamespace, Name: testRestore}, &velerov1.Restore{})
		assert.True(t, apierrors.IsNotFound(getErr))
		assertParentDeleted(t, k8sClient, restore.Name)
	})

	t.Run("tolerates an already deleted velero child", func(t *testing.T) {
		// given
		restore := deletableParent()

		recorderMock := newMockEventRecorder(t)

		providerMock := newMockRestoreProvider(t)
		providerMock.EXPECT().CheckReady(testCtx).Return(nil)

		oldVeleroProviderGetter := provider.NewVeleroProvider
		provider.NewVeleroProvider = func(client provider.K8sClient, recorder provider.EventRecorder, namespace string) provider.Provider {
			return providerMock
		}
		defer func() { provider.NewVeleroProvider = oldVeleroProviderGetter }()

		k8sClient := newTestClientWithParent(t, interceptor.Funcs{}, restore)
		sut := &defaultDeleteManager{k8sClient: k8sClient, namespace: testNamespace, recorder: recorderMock}

		// when
		err := sut.delete(testCtx, restore)

		// then
		require.NoError(t, err)
		assertParentDeleted(t, k8sClient, restore.Name)
	})

	t.Run("should return error on status update error", func(t *testing.T) {
		// given
		restore := deletableParent()

		recorderMock := newMockEventRecorder(t)

		sut := &defaultDeleteManager{k8sClient: newTestClientWithParent(t, failingStatusUpdate(assert.AnError), restore), namespace: testNamespace, recorder: recorderMock}

		// when
		err := sut.delete(testCtx, restore)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to update status [deleting] on restore [test-restore]")
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("should return error on velero child delete error", func(t *testing.T) {
		// given
		restore := deletableParent()

		recorderMock := newMockEventRecorder(t)

		providerMock := newMockRestoreProvider(t)
		providerMock.EXPECT().CheckReady(testCtx).Return(nil)

		oldVeleroProviderGetter := provider.NewVeleroProvider
		provider.NewVeleroProvider = func(client provider.K8sClient, recorder provider.EventRecorder, namespace string) provider.Provider {
			return providerMock
		}
		defer func() { provider.NewVeleroProvider = oldVeleroProviderGetter }()

		sut := &defaultDeleteManager{k8sClient: newTestClientWithParent(t, failingDelete(assert.AnError), restore, velero.BuildRestore(restore)), recorder: recorderMock, namespace: testNamespace}

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
		restore := deletableParent()

		recorderMock := newMockEventRecorder(t)

		providerMock := newMockRestoreProvider(t)
		providerMock.EXPECT().CheckReady(testCtx).Return(nil)

		oldVeleroProviderGetter := provider.NewVeleroProvider
		provider.NewVeleroProvider = func(client provider.K8sClient, recorder provider.EventRecorder, namespace string) provider.Provider {
			return providerMock
		}
		defer func() { provider.NewVeleroProvider = oldVeleroProviderGetter }()

		sut := &defaultDeleteManager{k8sClient: newTestClientWithParent(t, failingUpdate(assert.AnError), restore, velero.BuildRestore(restore)), recorder: recorderMock, namespace: testNamespace}

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
		restore := deletableParent()
		restore.Status.Status = v1.RestoreStatusCompleted // written before conditions existed

		return restore
	}
	reportedConflict := func() *v1.Restore {
		restore := deletableParent()
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
			restore:     deletableParent(),
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
			restore:     deletableParent(),
			child:       unownedChild(),
			wantDeleted: false,
		},
		"a child controlled by another restore survives": {
			restore:     deletableParent(),
			child:       foreignlyOwnedChild(),
			wantDeleted: false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			// given
			recorderMock := newMockEventRecorder(t)
			if !test.wantDeleted {
				recorderMock.EXPECT().Event(matchesRestoreNamed(test.restore.Name), corev1.EventTypeWarning, v1.DeleteEventReason, mock.MatchedBy(func(message string) bool {
					return strings.Contains(message, "not owned by this restore")
				}))
			}

			providerMock := newMockRestoreProvider(t)
			providerMock.EXPECT().CheckReady(testCtx).Return(nil)
			oldVeleroProviderGetter := provider.NewVeleroProvider
			provider.NewVeleroProvider = func(client provider.K8sClient, recorder provider.EventRecorder, namespace string) provider.Provider {
				return providerMock
			}
			t.Cleanup(func() { provider.NewVeleroProvider = oldVeleroProviderGetter })

			writes := &clientWrites{}
			k8sClient := newTestClientWithParent(t, writes.interceptor(), test.restore, test.child)
			sut := &defaultDeleteManager{k8sClient: k8sClient, namespace: testNamespace, recorder: recorderMock}

			// when
			err := sut.delete(testCtx, test.restore)

			// then the finalizer is removed either way, so deletion never wedges on a foreign child
			require.NoError(t, err)
			assertParentDeleted(t, k8sClient, test.restore.Name)
			getErr := k8sClient.Get(testCtx, types.NamespacedName{Namespace: testNamespace, Name: testRestore}, &velerov1.Restore{})
			if test.wantDeleted {
				assert.Equal(t, 1, writes.child.deletes)
				assert.True(t, apierrors.IsNotFound(getErr), "the own child must be gone")
			} else {
				assert.Equal(t, 0, writes.child.total(), "a child that is not ours must not be written to at all")
				assert.NoError(t, getErr, "the foreign child must survive its namesake")
			}
		})
	}
}
