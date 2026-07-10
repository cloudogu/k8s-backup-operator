package backup

import (
	"context"
	"testing"
	"time"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestReconciler(t *testing.T) {
	var testCtx = context.TODO()

	t.Run("If a backup resource has been created a backup should be created", func(t *testing.T) {
		backupCr := createBackup("ns1", "name1")

		backupApiMock := newMockEcosystemBackupInterface(t)
		backupApiMock.EXPECT().Get(testCtx, "name1", metav1.GetOptions{}).Return(backupCr, nil)

		service := NewMockService(t)
		service.EXPECT().createBackup(testCtx, Backup{Name: "name1"}).Return(nil)

		request := ctrl.Request{NamespacedName: types.NamespacedName{
			Namespace: "ns1",
			Name:      "name1",
		}}
		reconciler := NewReconciler(backupApiMock, service, nil)

		result, err := reconciler.Reconcile(testCtx, request)

		assert.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, result)
	})

	t.Run("If a backup resource has been deleted the corresponding backup should be deleted", func(t *testing.T) {
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
		backupApiMock := newMockEcosystemBackupInterface(t)
		backupApiMock.EXPECT().Get(testCtx, "name2", metav1.GetOptions{}).Return(backupCr, nil)

		service := NewMockService(t)
		service.EXPECT().deleteBackup(testCtx, Backup{Name: "name2"}).Return(nil)

		request := ctrl.Request{NamespacedName: types.NamespacedName{
			Namespace: "ns2",
			Name:      "name2",
		}}
		reconciler := NewReconciler(backupApiMock, service, nil)

		result, err := reconciler.Reconcile(testCtx, request)

		assert.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, result)
	})

	t.Run("If the backup does not complete in time and is still running it should be canceled", func(t *testing.T) {
		t.Skip("TODO: It is not really possible to cancel a velero backup. Should we let it finish?")

		var backupStartTime = metav1.Now()
		var lastTransitionTime = backupStartTime.Add(time.Minute * 10)

		backupCr := &backupv1.Backup{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ns3",
				Namespace: "name3",
			},
			Spec: backupv1.BackupSpec{
				Provider: "velero",
			},
			Status: backupv1.BackupStatus{
				StartTimestamp: backupStartTime,
				Conditions: []metav1.Condition{
					{
						Type:               "InProgress",
						Status:             metav1.ConditionTrue,
						LastTransitionTime: metav1.Time{lastTransitionTime},
					},
				},
			},
		}
		backupApiMock := newMockEcosystemBackupInterface(t)
		backupApiMock.EXPECT().Get(testCtx, "name3", metav1.GetOptions{}).Return(backupCr, nil)

		serviceMock := NewMockService(t)
		serviceMock.EXPECT().cancelBackup(testCtx, Backup{Name: "name3"}).Return(nil)

		configGatewayMock := newMockConfigGateway(t)
		configGatewayMock.EXPECT().RetryLimit(testCtx).Return(10, nil)

		request := ctrl.Request{NamespacedName: types.NamespacedName{
			Namespace: "ns3",
			Name:      "name3",
		}}
		reconciler := NewReconciler(backupApiMock, serviceMock, nil)

		result, err := reconciler.Reconcile(testCtx, request)

		assert.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, result)
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

	t.Run("It should convert a backupCr to a backup", func(t *testing.T) {
		t.Skip("TODO")
	})
}

func createBackup(namespace string, name string) *backupv1.Backup {
	return &backupv1.Backup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      namespace,
			Namespace: name,
		},
		Spec: backupv1.BackupSpec{
			Provider: "velero",
		},
	}
}
