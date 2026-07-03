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

	t.Run("should create a provider backup", func(t *testing.T) {
		backupCr := createBackup("ns1", "name1")

		backupApiMock := newMockEcosystemBackupInterface(t)
		backupApiMock.EXPECT().Get(testCtx, "name1", metav1.GetOptions{}).Return(backupCr, nil)

		backupProviderMock := newMockBackupProvider(t)
		backupProviderMock.EXPECT().CreateBackup(context.TODO(), backupCr).Return(nil)

		reconciler := NewReconciler(backupApiMock, backupProviderMock)

		request := ctrl.Request{NamespacedName: types.NamespacedName{
			Namespace: "ns1",
			Name:      "name1",
		}}

		_, err := reconciler.Reconcile(context.TODO(), request)

		assert.NoError(t, err)
	})

	t.Run("should delete the provider backup", func(t *testing.T) {
		now := metav1.NewTime(time.Now())
		backupCr := createBackup("ns1", "name1")
		backupCr.ObjectMeta.DeletionTimestamp = &now

		backupApiMock := newMockEcosystemBackupInterface(t)
		backupApiMock.EXPECT().Get(testCtx, "name1", metav1.GetOptions{}).Return(backupCr, nil)

		backupProviderMock := newMockBackupProvider(t)
		backupProviderMock.EXPECT().DeleteBackup(context.TODO(), backupCr).Return(nil)

		reconciler := NewReconciler(backupApiMock, backupProviderMock)

		request := ctrl.Request{NamespacedName: types.NamespacedName{
			Namespace: "ns1",
			Name:      "name1",
		}}

		_, err := reconciler.Reconcile(context.TODO(), request)

		assert.NoError(t, err)
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
