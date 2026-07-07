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

	t.Run("should create backup", func(t *testing.T) {
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

	t.Run("should delete backup", func(t *testing.T) {
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

	t.Run("should mark backup as 'NotFinishedInTime' if it's in 'InProgress' and retry time limit has been reached", func(t *testing.T) {
		t.Skip("TODO")

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
		serviceMock.EXPECT().markBackupAsNotFinishedInTime(testCtx, Backup{Name: "name3"}).Return(nil)

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
