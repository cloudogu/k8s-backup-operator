package backup

import (
	"context"
	"testing"
	"time"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	blueprintv3 "github.com/cloudogu/k8s-blueprint-lib/v3/api/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

const requeueAfterTest = time.Duration(5) * time.Second

func TestControllerReconcile(t *testing.T) {

	t.Run("If there is no backup do nothing", func(t *testing.T) {
		fakeClient := newFakeClientBuilder(t).Build()
		// We set the service to nil to check if the controller calls any method of the reconciler.
		controller := NewController(fakeClient, nil, requeueAfterTest)

		result, err := controller.Reconcile(context.Background(), newReconcilerRequest("ns", "backup"))

		assert.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, result)
	})

	t.Run("synchronize a provider backup before the normal backup process", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		backup.Spec.SyncedFromProvider = true
		fakeClient := newFakeClientBuilder(t).WithObjects(backup).Build()
		reconcilerMock := newMockReconciler(t)
		controller := NewController(fakeClient, reconcilerMock, requeueAfterTest)
		reconcilerMock.EXPECT().
			ensureProviderBackupDeleted(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureVeleroStatusSynced(context.Background(), mock.Anything, mock.Anything).
			Return(Abort, nil)

		result, err := controller.Reconcile(context.Background(), newReconcilerRequest("ns", "backup"))

		assert.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, result)
	})

	t.Run("retry while synchronizing the provider backup status", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		backup.Spec.SyncedFromProvider = true
		fakeClient := newFakeClientBuilder(t).WithObjects(backup).Build()
		reconcilerMock := newMockReconciler(t)
		controller := NewController(fakeClient, reconcilerMock, requeueAfterTest)
		reconcilerMock.EXPECT().
			ensureProviderBackupDeleted(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureVeleroStatusSynced(context.Background(), mock.Anything, mock.Anything).
			Return(Retry, assert.AnError)

		result, err := controller.Reconcile(context.Background(), newReconcilerRequest("ns", "backup"))

		assert.ErrorIs(t, err, assert.AnError)
		assert.Equal(t, ctrl.Result{RequeueAfter: requeueAfterTest}, result)
	})

	t.Run("check backup deletion and retry", func(t *testing.T) {
		reconcilerMock, controller := newTestFixtureForControllerTest(t)
		reconcilerMock.EXPECT().
			ensureProviderBackupDeleted(context.Background(), mock.Anything, mock.Anything).
			Return(Retry, assert.AnError)

		result, err := controller.Reconcile(context.Background(), newReconcilerRequest("ns", "backup"))

		assert.Error(t, err)
		assert.Equal(t, ctrl.Result{RequeueAfter: requeueAfterTest}, result)
	})

	t.Run("check backup deletion and proceed to the next step", func(t *testing.T) {
		reconcilerMock, controller := newTestFixtureForControllerTest(t)
		reconcilerMock.EXPECT().
			ensureProviderBackupDeleted(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		// The next step was called.
		reconcilerMock.EXPECT().
			ensureCompletedBackupIsIgnored(context.Background(), mock.Anything, mock.Anything).
			Return(Abort, nil)

		result, err := controller.Reconcile(context.Background(), newReconcilerRequest("ns", "backup"))

		assert.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, result)
	})

	t.Run("check backup deletion and abort", func(t *testing.T) {
		reconcilerMock, controller := newTestFixtureForControllerTest(t)
		reconcilerMock.EXPECT().
			ensureProviderBackupDeleted(context.Background(), mock.Anything, mock.Anything).
			Return(Abort, nil)

		result, err := controller.Reconcile(context.Background(), newReconcilerRequest("ns", "backup"))

		assert.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, result)
	})

	t.Run("check backup completion and abort", func(t *testing.T) {
		reconcilerMock, controller := newTestFixtureForControllerTest(t)
		reconcilerMock.EXPECT().
			ensureProviderBackupDeleted(context.Background(), mock.Anything, mock.Anything).
			Return(Abort, nil)

		result, err := controller.Reconcile(context.Background(), newReconcilerRequest("ns", "backup"))

		assert.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, result)
	})

	t.Run("check backup completion and proceed to the next step", func(t *testing.T) {
		reconcilerMock, controller := newTestFixtureForControllerTest(t)
		reconcilerMock.EXPECT().
			ensureProviderBackupDeleted(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureCompletedBackupIsIgnored(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		// The next step was called.
		reconcilerMock.EXPECT().
			ensureBackupIsCanceledAfterTimeWindowExpired(context.Background(), mock.Anything, mock.Anything).
			Return(Abort, nil)

		result, err := controller.Reconcile(context.Background(), newReconcilerRequest("ns", "backup"))

		assert.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, result)
	})

	t.Run("check backup cancellation and abort", func(t *testing.T) {
		reconcilerMock, controller := newTestFixtureForControllerTest(t)
		reconcilerMock.EXPECT().
			ensureProviderBackupDeleted(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureCompletedBackupIsIgnored(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureBackupIsCanceledAfterTimeWindowExpired(context.Background(), mock.Anything, mock.Anything).
			Return(Abort, nil)

		result, err := controller.Reconcile(context.Background(), newReconcilerRequest("ns", "backup"))

		assert.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, result)
	})

	t.Run("check backup cancellation and proceed to the next step", func(t *testing.T) {
		reconcilerMock, controller := newTestFixtureForControllerTest(t)
		reconcilerMock.EXPECT().
			ensureProviderBackupDeleted(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureCompletedBackupIsIgnored(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureBackupIsCanceledAfterTimeWindowExpired(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		// The next step was called.
		reconcilerMock.EXPECT().
			ensureBackupIsPrepared(context.Background(), mock.Anything, mock.Anything).
			Return(Abort, nil)

		result, err := controller.Reconcile(context.Background(), newReconcilerRequest("ns", "backup"))

		assert.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, result)
	})

	t.Run("check if the velero backup storage is available and retry", func(t *testing.T) {
		reconcilerMock, controller := newTestFixtureForControllerTest(t)
		reconcilerMock.EXPECT().
			ensureProviderBackupDeleted(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureCompletedBackupIsIgnored(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureBackupIsCanceledAfterTimeWindowExpired(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureBackupIsPrepared(context.Background(), mock.Anything, mock.Anything).
			Return(Retry, assert.AnError)

		result, err := controller.Reconcile(context.Background(), newReconcilerRequest("ns", "backup"))

		assert.Equal(t, err, assert.AnError)
		assert.Equal(t, ctrl.Result{RequeueAfter: requeueAfterTest}, result)
	})

	t.Run("check if the velero backup storage is available and abort", func(t *testing.T) {
		reconcilerMock, controller := newTestFixtureForControllerTest(t)
		reconcilerMock.EXPECT().
			ensureProviderBackupDeleted(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureCompletedBackupIsIgnored(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureBackupIsCanceledAfterTimeWindowExpired(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureBackupIsPrepared(context.Background(), mock.Anything, mock.Anything).
			Return(Abort, assert.AnError)

		result, err := controller.Reconcile(context.Background(), newReconcilerRequest("ns", "backup"))

		assert.Equal(t, err, assert.AnError)
		assert.Equal(t, ctrl.Result{}, result)
	})

	t.Run("check if the velero backup storage is available and proceed to the next step", func(t *testing.T) {
		reconcilerMock, controller := newTestFixtureForControllerTest(t)
		reconcilerMock.EXPECT().
			ensureProviderBackupDeleted(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureCompletedBackupIsIgnored(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureBackupIsCanceledAfterTimeWindowExpired(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureBackupIsPrepared(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		// The next step was called.
		reconcilerMock.EXPECT().
			ensureMaintenanceActivated(context.Background(), mock.Anything, mock.Anything).
			Return(Abort, nil)

		result, err := controller.Reconcile(context.Background(), newReconcilerRequest("ns", "backup"))

		assert.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, result)
	})

	t.Run("check if the maintenance mode is active and retry", func(t *testing.T) {
		reconcilerMock, controller := newTestFixtureForControllerTest(t)
		reconcilerMock.EXPECT().
			ensureProviderBackupDeleted(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureCompletedBackupIsIgnored(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureBackupIsCanceledAfterTimeWindowExpired(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureBackupIsPrepared(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureMaintenanceActivated(context.Background(), mock.Anything, mock.Anything).
			Return(Retry, assert.AnError)

		result, err := controller.Reconcile(context.Background(), newReconcilerRequest("ns", "backup"))

		assert.Equal(t, assert.AnError, err)
		assert.Equal(t, ctrl.Result{RequeueAfter: requeueAfterTest}, result)
	})

	t.Run("check if the maintenance mode is active and abort", func(t *testing.T) {
		reconcilerMock, controller := newTestFixtureForControllerTest(t)
		reconcilerMock.EXPECT().
			ensureProviderBackupDeleted(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureCompletedBackupIsIgnored(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureBackupIsCanceledAfterTimeWindowExpired(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureBackupIsPrepared(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureMaintenanceActivated(context.Background(), mock.Anything, mock.Anything).
			Return(Abort, assert.AnError)

		result, err := controller.Reconcile(context.Background(), newReconcilerRequest("ns", "backup"))

		assert.Equal(t, assert.AnError, err)
		assert.Equal(t, ctrl.Result{}, result)
	})

	t.Run("check if the maintenance mode is active and proceed to the next step", func(t *testing.T) {
		reconcilerMock, controller := newTestFixtureForControllerTest(t)
		reconcilerMock.EXPECT().
			ensureProviderBackupDeleted(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureCompletedBackupIsIgnored(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureBackupIsCanceledAfterTimeWindowExpired(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureBackupIsPrepared(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureMaintenanceActivated(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		// The next step was called.
		reconcilerMock.EXPECT().
			ensureProviderBackupCreated(context.Background(), mock.Anything, mock.Anything).
			Return(Abort, nil)

		result, err := controller.Reconcile(context.Background(), newReconcilerRequest("ns", "backup"))

		assert.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, result)
	})

	t.Run("check velero backup resource and retry", func(t *testing.T) {
		reconcilerMock, controller := newTestFixtureForControllerTest(t)
		reconcilerMock.EXPECT().
			ensureProviderBackupDeleted(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureCompletedBackupIsIgnored(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureBackupIsCanceledAfterTimeWindowExpired(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureBackupIsPrepared(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureMaintenanceActivated(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureProviderBackupCreated(context.Background(), mock.Anything, mock.Anything).
			Return(Retry, assert.AnError)

		result, err := controller.Reconcile(context.Background(), newReconcilerRequest("ns", "backup"))

		assert.Error(t, err)
		assert.Equal(t, ctrl.Result{RequeueAfter: requeueAfterTest}, result)
	})

	t.Run("check velero backup resource and abort", func(t *testing.T) {
		reconcilerMock, controller := newTestFixtureForControllerTest(t)
		reconcilerMock.EXPECT().
			ensureProviderBackupDeleted(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureCompletedBackupIsIgnored(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureBackupIsCanceledAfterTimeWindowExpired(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureBackupIsPrepared(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureMaintenanceActivated(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureProviderBackupCreated(context.Background(), mock.Anything, mock.Anything).
			Return(Abort, nil)

		result, err := controller.Reconcile(context.Background(), newReconcilerRequest("ns", "backup"))

		assert.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, result)
	})

	t.Run("check velero backup resource and proceed to the next step", func(t *testing.T) {
		reconcilerMock, controller := newTestFixtureForControllerTest(t)
		reconcilerMock.EXPECT().
			ensureProviderBackupDeleted(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureCompletedBackupIsIgnored(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureBackupIsCanceledAfterTimeWindowExpired(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureBackupIsPrepared(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureMaintenanceActivated(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureProviderBackupCreated(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		// The next step was called.
		reconcilerMock.EXPECT().
			ensureProviderBackupCompleted(context.Background(), mock.Anything, mock.Anything).
			Return(Abort, nil)

		result, err := controller.Reconcile(context.Background(), newReconcilerRequest("ns", "backup"))

		assert.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, result)
	})

	t.Run("check velero backup completion and retry", func(t *testing.T) {
		reconcilerMock, controller := newTestFixtureForControllerTest(t)
		reconcilerMock.EXPECT().
			ensureProviderBackupDeleted(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureCompletedBackupIsIgnored(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureBackupIsCanceledAfterTimeWindowExpired(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureBackupIsPrepared(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureMaintenanceActivated(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureProviderBackupCreated(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureProviderBackupCompleted(context.Background(), mock.Anything, mock.Anything).
			Return(Retry, nil)

		result, err := controller.Reconcile(context.Background(), newReconcilerRequest("ns", "backup"))

		assert.NoError(t, err)
		assert.Equal(t, ctrl.Result{RequeueAfter: requeueAfterTest}, result)
	})

	t.Run("check velero backup completion and abort", func(t *testing.T) {
		reconcilerMock, controller := newTestFixtureForControllerTest(t)
		reconcilerMock.EXPECT().
			ensureProviderBackupDeleted(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureCompletedBackupIsIgnored(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureBackupIsCanceledAfterTimeWindowExpired(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureBackupIsPrepared(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureMaintenanceActivated(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureProviderBackupCreated(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureProviderBackupCompleted(context.Background(), mock.Anything, mock.Anything).
			Return(Abort, assert.AnError)

		result, err := controller.Reconcile(context.Background(), newReconcilerRequest("ns", "backup"))

		assert.Error(t, err)
		assert.Equal(t, ctrl.Result{}, result)
	})

	t.Run("check velero backup completion and proceed to the next step", func(t *testing.T) {
		reconcilerMock, controller := newTestFixtureForControllerTest(t)
		reconcilerMock.EXPECT().
			ensureProviderBackupDeleted(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureCompletedBackupIsIgnored(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureBackupIsCanceledAfterTimeWindowExpired(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureBackupIsPrepared(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureMaintenanceActivated(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureProviderBackupCreated(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureProviderBackupCompleted(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		// The next step was called.
		reconcilerMock.EXPECT().
			ensureMaintenanceDeactivated(context.Background(), mock.Anything, mock.Anything).
			Return(Abort, nil)

		result, err := controller.Reconcile(context.Background(), newReconcilerRequest("ns", "backup"))

		assert.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, result)
	})

	t.Run("check maintenance mode active after backup and retry", func(t *testing.T) {
		reconcilerMock, controller := newTestFixtureForControllerTest(t)
		reconcilerMock.EXPECT().
			ensureProviderBackupDeleted(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureCompletedBackupIsIgnored(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureBackupIsCanceledAfterTimeWindowExpired(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureBackupIsPrepared(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureMaintenanceActivated(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureProviderBackupCreated(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureProviderBackupCompleted(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureMaintenanceDeactivated(context.Background(), mock.Anything, mock.Anything).
			Return(Retry, nil)

		result, err := controller.Reconcile(context.Background(), newReconcilerRequest("ns", "backup"))

		assert.NoError(t, err)
		assert.Equal(t, ctrl.Result{RequeueAfter: requeueAfterTest}, result)
	})

	t.Run("check maintenance mode active after backup and abort", func(t *testing.T) {
		reconcilerMock, controller := newTestFixtureForControllerTest(t)
		reconcilerMock.EXPECT().
			ensureProviderBackupDeleted(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureCompletedBackupIsIgnored(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureBackupIsCanceledAfterTimeWindowExpired(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureBackupIsPrepared(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureMaintenanceActivated(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureProviderBackupCreated(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureProviderBackupCompleted(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureMaintenanceDeactivated(context.Background(), mock.Anything, mock.Anything).
			Return(Abort, assert.AnError)

		result, err := controller.Reconcile(context.Background(), newReconcilerRequest("ns", "backup"))

		assert.Error(t, err)
		assert.Equal(t, ctrl.Result{}, result)
	})

	t.Run("check maintenance mode active after backup and proceed to the next step", func(t *testing.T) {
		reconcilerMock, controller := newTestFixtureForControllerTest(t)
		reconcilerMock.EXPECT().
			ensureProviderBackupDeleted(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureCompletedBackupIsIgnored(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureBackupIsCanceledAfterTimeWindowExpired(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureBackupIsPrepared(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureMaintenanceActivated(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureProviderBackupCreated(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureProviderBackupCompleted(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			ensureMaintenanceDeactivated(context.Background(), mock.Anything, mock.Anything).
			Return(Next, nil)

		result, err := controller.Reconcile(context.Background(), newReconcilerRequest("ns", "backup"))

		assert.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, result)
	})

}

func newBackupForTest(namespace string, name string) *backupv1.Backup {
	return &backupv1.Backup{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Spec: backupv1.BackupSpec{
			Provider: "velero",
		},
	}
}

func newFakeClientBuilder(t *testing.T) *fake.ClientBuilder {
	scheme := runtime.NewScheme()
	require.NoError(t, backupv1.AddToScheme(scheme))
	require.NoError(t, blueprintv3.AddToScheme(scheme))
	require.NoError(t, velerov1.AddToScheme(scheme))
	require.NoError(t, corev1.AddToScheme(scheme))

	return fake.NewClientBuilder().WithScheme(scheme)
}

func newFakeClientBuilderWithCounter(t *testing.T, callCounter *callCounter) *fake.ClientBuilder {
	return newFakeClientBuilder(t).
		WithInterceptorFuncs(interceptor.Funcs{
			Get:              callCounter.getCall,
			SubResourcePatch: callCounter.subResourcePatchCall,
			Create:           callCounter.createCall,
		})
}

func newReconcilerRequest(namespace string, name string) ctrl.Request {
	return ctrl.Request{NamespacedName: types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}}
}

func newTestFixtureForControllerTest(t *testing.T) (*mockReconciler, *Controller) {
	backup := newBackupForTest("ns", "backup")
	fakeClient := newFakeClientBuilder(t).
		WithObjects(backup).
		Build()

	reconcilerMock := newMockReconciler(t)
	reconcilerMock.EXPECT().
		ensureVeleroStatusSynced(context.Background(), mock.Anything, mock.Anything).
		Return(Next, nil).Maybe()
	reconcilerMock.EXPECT().
		ensureBackupSetup(context.Background(), mock.Anything, mock.Anything).
		Return(Next, nil).Maybe()
	controller := NewController(fakeClient, reconcilerMock, time.Duration(5)*time.Second)
	return reconcilerMock, controller
}
