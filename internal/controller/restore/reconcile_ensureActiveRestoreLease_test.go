package restore

import (
	"context"
	"testing"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	coordinationv1 "k8s.io/api/coordination/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestReconcileEnsureActiveRestoreLease(t *testing.T) {
	t.Run("should skip lease handling when the restore is terminal", func(t *testing.T) {
		restore := newParentRestore()
		applyConditions(restore, []metav1.Condition{{
			Type: backupv1.ConditionSuccessful, Status: metav1.ConditionTrue, Reason: ReasonRestoreCompleted,
		}})
		clientMock := newMockK8sClient(t)
		sut := &restoreReconciler{k8sClient: clientMock, namespace: testNamespace}

		actualRestore, actualOutcome := sut.ensureActiveRestoreLease(testCtx, restore)

		assert.Same(t, restore, actualRestore)
		assert.Equal(t, next(), actualOutcome)
	})

	t.Run("should create a lease for a restore when none exists", func(t *testing.T) {
		restore := newParentRestore()
		clientMock := newMockK8sClient(t)
		clientMock.EXPECT().Get(testCtx, leaseKey(), mock.Anything).Return(leaseNotFound())
		clientMock.EXPECT().Create(testCtx, mock.Anything).
			Run(func(_ context.Context, object client.Object, _ ...client.CreateOption) {
				lease, ok := object.(*coordinationv1.Lease)
				require.True(t, ok)
				assert.Equal(t, restoreLeaseName, lease.Name)
				assert.Equal(t, testNamespace, lease.Namespace)
				assert.Equal(t, string(restore.UID), ptr.Deref(lease.Spec.HolderIdentity, ""))
				assert.Equal(t, restore.Name, lease.Annotations[restoreLeaseHolderNameAnnotation])
				assert.NotNil(t, lease.Spec.AcquireTime)
				assert.NotNil(t, lease.Spec.RenewTime)
				assert.Equal(t, int32(1), ptr.Deref(lease.Spec.LeaseTransitions, 0))
			}).
			Return(nil)
		sut := &restoreReconciler{k8sClient: clientMock, namespace: testNamespace}

		actualRestore, actualOutcome := sut.ensureActiveRestoreLease(testCtx, restore)

		assert.Same(t, restore, actualRestore)
		assert.Equal(t, retryAfter(defaultRequeueDelay), actualOutcome)
	})

	for _, test := range []struct {
		name          string
		getError      error
		createError   error
		wantRetry     bool
		wantErrorText string
	}{
		{
			name:        "should retry when another restore creates the lease first",
			getError:    leaseNotFound(),
			createError: apierrors.NewAlreadyExists(leaseResource(), restoreLeaseName),
			wantRetry:   true,
		},
		{
			name:          "should return an error when reading the lease fails",
			getError:      assert.AnError,
			wantErrorText: "failed to get restore lease",
		},
		{
			name:          "should return an error when creating the lease fails",
			getError:      leaseNotFound(),
			createError:   assert.AnError,
			wantErrorText: "failed to create restore lease",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			restore := newParentRestore()
			clientMock := newMockK8sClient(t)
			clientMock.EXPECT().Get(testCtx, leaseKey(), mock.Anything).Return(test.getError)
			if apierrors.IsNotFound(test.getError) {
				clientMock.EXPECT().Create(testCtx, mock.Anything).Return(test.createError)
			}
			sut := &restoreReconciler{k8sClient: clientMock, namespace: testNamespace}

			actualRestore, actualOutcome := sut.ensureActiveRestoreLease(testCtx, restore)

			assert.Same(t, restore, actualRestore)
			if test.wantRetry {
				assert.Equal(t, retryAfter(defaultRequeueDelay), actualOutcome)
			} else {
				require.Error(t, actualOutcome.err)
				assert.ErrorIs(t, actualOutcome.err, assert.AnError)
				assert.ErrorContains(t, actualOutcome.err, test.wantErrorText)
			}
		})
	}

	t.Run("should continue when the restore already holds the lease", func(t *testing.T) {
		restore := newParentRestore()
		clientMock := newMockK8sClient(t)
		expectLeaseRead(t, clientMock, newRestoreLease(restore))
		sut := &restoreReconciler{k8sClient: clientMock, namespace: testNamespace}

		actualRestore, actualOutcome := sut.ensureActiveRestoreLease(testCtx, restore)

		assert.Same(t, restore, actualRestore)
		assert.Equal(t, next(), actualOutcome)
	})

	t.Run("should wait and report the active restore holding the lease", func(t *testing.T) {
		restore := newParentRestore()
		holder := restoreWithIdentity("active-restore", types.UID("active-uid"))
		clientMock := newMockK8sClient(t)
		expectLeaseRead(t, clientMock, newRestoreLease(holder))
		expectRestoreRead(t, clientMock, holder)
		statusClient := newTestClient(t, interceptor.Funcs{}, restore.DeepCopy())
		err := statusClient.Get(testCtx, client.ObjectKeyFromObject(restore), restore)
		require.NoError(t, err)
		clientMock.EXPECT().Status().Return(statusClient.Status())
		sut := &restoreReconciler{k8sClient: clientMock, namespace: testNamespace}

		actualRestore, actualOutcome := sut.ensureActiveRestoreLease(testCtx, restore)

		assert.Equal(t, retryAfter(defaultRequeueDelay), actualOutcome)
		condition := findSuccessfulCondition(actualRestore)
		require.NotNil(t, condition)
		assert.Equal(t, metav1.ConditionUnknown, condition.Status)
		assert.Equal(t, ReasonWaitingForActiveRestore, condition.Reason)
		assert.Contains(t, condition.Message, holder.Name)
	})

	for _, test := range []struct {
		name        string
		prepareMock func(*testing.T, *mockK8sClient, *backupv1.Restore)
	}{
		{
			name: "holder no longer exists",
			prepareMock: func(t *testing.T, clientMock *mockK8sClient, holder *backupv1.Restore) {
				key := client.ObjectKey{Namespace: testNamespace, Name: holder.Name}
				clientMock.EXPECT().Get(testCtx, key, mock.Anything).
					Return(apierrors.NewNotFound(restoreResource(), holder.Name))
			},
		},
		{
			name: "holder name was reused with another UID",
			prepareMock: func(t *testing.T, clientMock *mockK8sClient, holder *backupv1.Restore) {
				replacement := holder.DeepCopy()
				replacement.UID = types.UID("replacement-uid")
				expectRestoreRead(t, clientMock, replacement)
			},
		},
		{
			name: "holder reached a terminal state",
			prepareMock: func(t *testing.T, clientMock *mockK8sClient, holder *backupv1.Restore) {
				terminal := holder.DeepCopy()
				applyConditions(terminal, []metav1.Condition{{
					Type: backupv1.ConditionSuccessful, Status: metav1.ConditionFalse,
					Reason: ReasonProviderRestoreFailed,
				}})
				expectRestoreRead(t, clientMock, terminal)
			},
		},
	} {
		t.Run("should take over the lease when the "+test.name, func(t *testing.T) {
			restore := newParentRestore()
			holder := restoreWithIdentity("previous-restore", types.UID("previous-uid"))
			lease := newRestoreLease(holder)
			lease.ResourceVersion = "7"
			clientMock := newMockK8sClient(t)
			expectLeaseRead(t, clientMock, lease)
			test.prepareMock(t, clientMock, holder)
			clientMock.EXPECT().Update(testCtx, mock.Anything).
				Run(func(_ context.Context, object client.Object, _ ...client.UpdateOption) {
					updatedLease, ok := object.(*coordinationv1.Lease)
					require.True(t, ok)
					assert.Equal(t, "7", updatedLease.ResourceVersion)
					assert.Equal(t, string(restore.UID), ptr.Deref(updatedLease.Spec.HolderIdentity, ""))
					assert.Equal(t, restore.Name, updatedLease.Annotations[restoreLeaseHolderNameAnnotation])
					assert.Equal(t, int32(2), ptr.Deref(updatedLease.Spec.LeaseTransitions, 0))
				}).
				Return(nil)
			sut := &restoreReconciler{k8sClient: clientMock, namespace: testNamespace}

			actualRestore, actualOutcome := sut.ensureActiveRestoreLease(testCtx, restore)

			assert.Same(t, restore, actualRestore)
			assert.Equal(t, retryAfter(defaultRequeueDelay), actualOutcome)
		})
	}

	t.Run("should return an error when checking the holder fails", func(t *testing.T) {
		restore := newParentRestore()
		holder := restoreWithIdentity("previous-restore", types.UID("previous-uid"))
		clientMock := newMockK8sClient(t)
		expectLeaseRead(t, clientMock, newRestoreLease(holder))
		holderKey := client.ObjectKey{Namespace: testNamespace, Name: holder.Name}
		clientMock.EXPECT().Get(testCtx, holderKey, mock.Anything).Return(assert.AnError)
		sut := &restoreReconciler{k8sClient: clientMock, namespace: testNamespace}

		actualRestore, actualOutcome := sut.ensureActiveRestoreLease(testCtx, restore)

		assert.Same(t, restore, actualRestore)
		require.Error(t, actualOutcome.err)
		assert.ErrorIs(t, actualOutcome.err, assert.AnError)
		assert.ErrorContains(t, actualOutcome.err, "failed to verify holder")
	})

	for _, test := range []struct {
		name        string
		updateError error
		wantError   bool
	}{
		{
			name:        "should retry on a conflicting takeover",
			updateError: apierrors.NewConflict(leaseResource(), restoreLeaseName, assert.AnError),
		},
		{
			name:        "should return an error when taking over the lease fails",
			updateError: assert.AnError,
			wantError:   true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			restore := newParentRestore()
			holder := restoreWithIdentity("previous-restore", types.UID("previous-uid"))
			clientMock := newMockK8sClient(t)
			expectLeaseRead(t, clientMock, newRestoreLease(holder))
			holderKey := client.ObjectKey{Namespace: testNamespace, Name: holder.Name}
			clientMock.EXPECT().Get(testCtx, holderKey, mock.Anything).
				Return(apierrors.NewNotFound(restoreResource(), holder.Name))
			clientMock.EXPECT().Update(testCtx, mock.Anything).Return(test.updateError)
			sut := &restoreReconciler{k8sClient: clientMock, namespace: testNamespace}

			actualRestore, actualOutcome := sut.ensureActiveRestoreLease(testCtx, restore)

			assert.Same(t, restore, actualRestore)
			if test.wantError {
				require.Error(t, actualOutcome.err)
				assert.ErrorIs(t, actualOutcome.err, test.updateError)
				assert.ErrorContains(t, actualOutcome.err, "failed to take over stale restore lease")
			} else {
				assert.Equal(t, retryAfter(defaultRequeueDelay), actualOutcome)
			}
		})
	}
}

func expectLeaseRead(t *testing.T, clientMock *mockK8sClient, lease *coordinationv1.Lease) {
	t.Helper()
	clientMock.EXPECT().Get(testCtx, leaseKey(), mock.Anything).
		Run(func(_ context.Context, _ types.NamespacedName, object client.Object, _ ...client.GetOption) {
			actual, ok := object.(*coordinationv1.Lease)
			require.True(t, ok)
			lease.DeepCopyInto(actual)
		}).
		Return(nil)
}

func expectRestoreRead(t *testing.T, clientMock *mockK8sClient, restore *backupv1.Restore) {
	t.Helper()
	key := client.ObjectKey{Namespace: testNamespace, Name: restore.Name}
	clientMock.EXPECT().Get(testCtx, key, mock.Anything).
		Run(func(_ context.Context, _ types.NamespacedName, object client.Object, _ ...client.GetOption) {
			actual, ok := object.(*backupv1.Restore)
			require.True(t, ok)
			restore.DeepCopyInto(actual)
		}).
		Return(nil)
}

func restoreWithIdentity(name string, uid types.UID) *backupv1.Restore {
	restore := newParentRestore()
	restore.Name = name
	restore.UID = uid
	return restore
}

func leaseKey() client.ObjectKey {
	return client.ObjectKey{Namespace: testNamespace, Name: restoreLeaseName}
}

func leaseNotFound() error {
	return apierrors.NewNotFound(leaseResource(), restoreLeaseName)
}

func leaseResource() schema.GroupResource {
	return schema.GroupResource{Group: coordinationv1.GroupName, Resource: "leases"}
}

func restoreResource() schema.GroupResource {
	return schema.GroupResource{Group: backupv1.GroupVersion.Group, Resource: "restores"}
}
