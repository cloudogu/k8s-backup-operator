package backup

import (
	"context"
	"testing"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	blueprintv3 "github.com/cloudogu/k8s-blueprint-lib/v3/api/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestControllerReconcile(t *testing.T) {
	t.Run("If there is no backup do nothing", func(t *testing.T) {
		fakeClient := newFakeClientBuilder(t).Build()
		// We set the service to nil to check if the controller calls any method of the reconciler.
		controller := NewController(fakeClient, nil)

		result, err := controller.Reconcile(context.Background(), newReconcilerRequest("ns", "backup"))

		assert.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, result)
	})

	t.Run("check if the velero backup storage is available and retry", func(t *testing.T) {
		reconcilerMock, controller := newTestFixtureForControllerTest(t)
		reconcilerMock.EXPECT().
			checkVeleroBackupStorage(context.Background(), mock.Anything, "ns", mock.Anything).
			Return(Retry, assert.AnError)

		result, err := controller.Reconcile(context.Background(), newReconcilerRequest("ns", "backup"))

		assert.Equal(t, err, assert.AnError)
		assert.Equal(t, ctrl.Result{RequeueAfter: defaultRequeueAfterTime}, result)
	})

	t.Run("check if the velero backup storage is available and abort", func(t *testing.T) {
		reconcilerMock, controller := newTestFixtureForControllerTest(t)
		reconcilerMock.EXPECT().
			checkVeleroBackupStorage(context.Background(), mock.Anything, "ns", mock.Anything).
			Return(Abort, assert.AnError)

		result, err := controller.Reconcile(context.Background(), newReconcilerRequest("ns", "backup"))

		assert.Equal(t, err, assert.AnError)
		assert.Equal(t, ctrl.Result{}, result)
	})

	t.Run("check if the velero backup storage is available and proceed to the next step", func(t *testing.T) {
		reconcilerMock, controller := newTestFixtureForControllerTest(t)
		reconcilerMock.EXPECT().
			checkVeleroBackupStorage(context.Background(), mock.Anything, "ns", mock.Anything).
			Return(Next, nil)
		// The next step was called.
		reconcilerMock.EXPECT().
			checkMaintenanceModeIsActive(context.Background(), mock.Anything, "ns", mock.Anything).
			Return(Abort, nil)

		result, err := controller.Reconcile(context.Background(), newReconcilerRequest("ns", "backup"))

		assert.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, result)
	})

	t.Run("check if the maintenance mode is active and retry", func(t *testing.T) {
		reconcilerMock, controller := newTestFixtureForControllerTest(t)
		reconcilerMock.EXPECT().
			checkVeleroBackupStorage(context.Background(), mock.Anything, "ns", mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			checkMaintenanceModeIsActive(context.Background(), mock.Anything, "ns", mock.Anything).
			Return(Retry, assert.AnError)

		result, err := controller.Reconcile(context.Background(), newReconcilerRequest("ns", "backup"))

		assert.Equal(t, assert.AnError, err)
		assert.Equal(t, ctrl.Result{RequeueAfter: defaultRequeueAfterTime}, result)
	})

	t.Run("check if the maintenance mode is active and abort", func(t *testing.T) {
		reconcilerMock, controller := newTestFixtureForControllerTest(t)
		reconcilerMock.EXPECT().
			checkVeleroBackupStorage(context.Background(), mock.Anything, "ns", mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			checkMaintenanceModeIsActive(context.Background(), mock.Anything, "ns", mock.Anything).
			Return(Abort, assert.AnError)

		result, err := controller.Reconcile(context.Background(), newReconcilerRequest("ns", "backup"))

		assert.Equal(t, assert.AnError, err)
		assert.Equal(t, ctrl.Result{}, result)
	})

	t.Run("check if the maintenance mode is active and proceed to the next step", func(t *testing.T) {
		reconcilerMock, controller := newTestFixtureForControllerTest(t)
		reconcilerMock.EXPECT().
			checkVeleroBackupStorage(context.Background(), mock.Anything, "ns", mock.Anything).
			Return(Next, nil)
		reconcilerMock.EXPECT().
			checkMaintenanceModeIsActive(context.Background(), mock.Anything, "ns", mock.Anything).
			Return(Next, nil)
		// The next step was called.
		reconcilerMock.EXPECT().
			checkVeleroBackup(context.Background(), mock.Anything, "ns", mock.Anything).
			Return(Abort, nil)

		result, err := controller.Reconcile(context.Background(), newReconcilerRequest("ns", "backup"))

		assert.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, result)
	})

	t.Run("If the backup is about to delete, delete also the provider backup", func(t *testing.T) {
		t.Skip("TODO")
		/*
			backup := newBackupForControllerTest("ns", "backup")
			deletionTime := metav1.Now()
			backup.Finalizers = []string{"fakeFinalizer"}
			backup.DeletionTimestamp = &deletionTime

			fakeClient := newFakeClientBuilder(t).
				WithObjects(backup).
				Build()

			serviceMock := newMockReconciler(t)
			serviceMock.EXPECT().deleteBackup(mock.Anything, mock.Anything).Return(nil)

			controller := NewController(fakeClient, serviceMock)

			result, err := controller.Reconcile(context.Background(), newReconcilerRequest("ns", "backup"))

			assert.NoError(t, err)
			assert.Equal(t, ctrl.Result{}, result)

		*/
	})

	t.Run("If the backup does not complete in time and is still running it should be canceled", func(t *testing.T) {
		t.Skip("TODO: It is not really possible to cancel a velero backup. Should we let it finish?")
	})

	t.Run("If the backup does not complete in time and remains in an error state it should be canceled", func(t *testing.T) {
		t.Skip("TODO")
	})

	t.Run("If the backup is in an error state and has still time to complete we check it later again", func(t *testing.T) {
		t.Skip("TODO")
	})

	t.Run("If the backup is completed we do nothing", func(t *testing.T) {
		t.Skip("TODO")
		// We could configure the reconciler mock without a Service (=nil) and if any function of
		// that service was called the test will fail.
	})
}

func newBackupForControllerTest(namespace string, name string) *backupv1.Backup {
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

	return fake.NewClientBuilder().WithScheme(scheme)
}

func newReconcilerRequest(namespace string, name string) ctrl.Request {
	return ctrl.Request{NamespacedName: types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}}
}

func newTestFixtureForControllerTest(t *testing.T) (*mockReconciler, *Controller) {
	backup := newBackupForControllerTest("ns", "backup")
	fakeClient := newFakeClientBuilder(t).
		WithObjects(backup).
		Build()

	reconcilerMock := newMockReconciler(t)
	controller := NewController(fakeClient, reconcilerMock)
	return reconcilerMock, controller
}
