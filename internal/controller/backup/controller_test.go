package backup

import (
	"context"
	"testing"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	blueprintv3 "github.com/cloudogu/k8s-blueprint-lib/v3/api/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestControllerReconcile(t *testing.T) {
	t.Run("should configure backup", func(t *testing.T) {
		backup := newBackupForControllerReconcileTest("ns1", "backup")
		blueprintList := &blueprintv3.BlueprintList{Items: make([]blueprintv3.Blueprint, 0)}

		scheme := runtime.NewScheme()
		require.NoError(t, backupv1.AddToScheme(scheme))
		require.NoError(t, blueprintv3.AddToScheme(scheme))

		updateCalled := false
		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(backup).
			WithLists(blueprintList).
			WithInterceptorFuncs(interceptor.Funcs{
				Update: func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
					updateCalled = true
					return client.Update(ctx, obj, opts...)
				},
			}).
			Build()

		serviceMock := newMockService(t)
		serviceMock.EXPECT().configureBackup(context.Background(), mock.Anything)
		serviceMock.EXPECT().reconcileBackup(context.Background(), mock.Anything).Return(nil)

		request := ctrl.Request{NamespacedName: types.NamespacedName{
			Namespace: "ns1",
			Name:      "backup",
		}}
		controller := NewController(fakeClient, serviceMock)

		result, err := controller.Reconcile(context.Background(), request)

		assert.True(t, updateCalled, "backup resource should be updated")
		assert.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, result)
	})

	t.Run("If there is a blueprint, add an annotation with blueprint infos", func(t *testing.T) {
		scheme := runtime.NewScheme()
		require.NoError(t, backupv1.AddToScheme(scheme))
		require.NoError(t, blueprintv3.AddToScheme(scheme))

		backup := newBackupForControllerReconcileTest("ns", "backup")
		blueprintList := newBlueprintListForControllerReconcilerTest(
			"ns",
			"blueprint",
			"blueprint-display-name-01",
			[]blueprintv3.Dogu{
				{Name: "dogu01"},
				{Name: "dogu02"},
			},
		)

		updateCalled := false
		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(backup).
			WithLists(blueprintList).
			WithInterceptorFuncs(interceptor.Funcs{
				Update: func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
					updateCalled = true
					return client.Update(ctx, obj, opts...)
				},
			}).
			Build()

		serviceMock := newMockService(t)
		serviceMock.EXPECT().
			configureBackup(context.Background(), mock.Anything)
		serviceMock.EXPECT().
			addBlueprintAnnotation(context.Background(), mock.Anything, mock.Anything, mock.Anything).
			Run(func(ctx context.Context, backup *backupv1.Backup, displayName string, dogus []blueprintv3.Dogu) {
				assert.Equal(t, "blueprint-display-name-01", displayName)
				assert.ElementsMatch(t, []blueprintv3.Dogu{{Name: "dogu01"}, {Name: "dogu02"}}, dogus)
			})
		serviceMock.EXPECT().
			reconcileBackup(context.Background(), mock.Anything).
			Return(nil)

		controller := NewController(fakeClient, serviceMock)

		result, err := controller.Reconcile(context.Background(), newReconcileRequestForTest("ns", "backup"))

		assert.True(t, updateCalled, "backup resource should be updated")
		assert.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, result)
	})

	t.Run("If a backup resource has been created a backup should be created", func(t *testing.T) {
		t.Skip("TODO")
		/*
			backup := createBackup("ns1", "name1")

			scheme := runtime.NewScheme()
			require.NoError(t, backupv1.AddToScheme(scheme))

			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				Build()

			serviceMock := newMockService(t)
			serviceMock.EXPECT().reconcileBackup(context.Background(), backup)

			request := ctrl.Request{NamespacedName: types.NamespacedName{
				Namespace: "ns1",
				Name:      "name1",
			}}
			controller := NewController(fakeClient, serviceMock)

			result, err := controller.Reconcile(context.Background(), request)

			assert.NoError(t, err)
			assert.Equal(t, ctrl.Result{}, result)

		*/
	})

	t.Run("If a backup resource has been deleted the corresponding backup should be deleted", func(t *testing.T) {
		t.Skip("TODO")

		/*
			var deletionTime = metav1.Now()
			backupCr := &backupv1.Backup{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "ns2",
					Namespace:         "name2",
					DeletionTimestamp: &deletionTime,
				},
				Spec: backupv1.BackupSpec{
					Provider: "velero",
				},
			}

			fakeClient := fake.NewClientBuilder().Build()

			serviceMock := newMockService(t)
			serviceMock.EXPECT().deleteBackup(context.Background(), backupCr).Return(nil)

			request := ctrl.Request{NamespacedName: types.NamespacedName{
				Namespace: "ns2",
				Name:      "name2",
			}}
			reconciler := NewController(fakeClient, serviceMock)

			result, err := reconciler.Reconcile(context.Background(), request)

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

func newBackupForControllerReconcileTest(namespace string, name string) *backupv1.Backup {
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

func newReconcileRequestForTest(namespace string, name string) ctrl.Request {
	return ctrl.Request{NamespacedName: types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}}
}

func newBlueprintListForControllerReconcilerTest(
	namespace string,
	name string,
	displayName string,
	dogus []blueprintv3.Dogu,
) *blueprintv3.BlueprintList {
	blueprint := blueprintv3.Blueprint{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Spec: blueprintv3.BlueprintSpec{
			DisplayName: displayName,
			Blueprint: blueprintv3.BlueprintManifest{
				Dogus: dogus,
			},
		},
	}
	return &blueprintv3.BlueprintList{
		Items: []blueprintv3.Blueprint{blueprint},
	}
}
