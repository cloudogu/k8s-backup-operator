package backup

import (
	"context"
	"testing"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/k8s-backup-operator/pkg/annotations"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestVeleroBackupController(t *testing.T) {
	t.Run("creates the corresponding cloudogu backup", func(t *testing.T) {
		veleroBackup := &velerov1.Backup{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "backup",
				Namespace: "ns",
				Annotations: map[string]string{
					annotations.BlueprintIdAnnotation: "blueprint",
					annotations.DogusAnnotation:       "dogus",
					"unrelated":                       "ignored",
				},
			},
		}
		fakeClient := newFakeClientBuilder(t).WithObjects(veleroBackup).Build()
		reconciler := NewVeleroBackupReconciler(fakeClient)
		controller := NewVeleroBackupController(fakeClient, reconciler)

		result, err := controller.Reconcile(context.Background(), newReconcilerRequest("ns", "backup"))

		require.NoError(t, err)
		assert.Zero(t, result)
		backup := &backupv1.Backup{}
		require.NoError(t, fakeClient.Get(context.Background(), client.ObjectKeyFromObject(veleroBackup), backup))
		assert.Equal(t, backupv1.Provider("velero"), backup.Spec.Provider)
		assert.True(t, backup.Spec.SyncedFromProvider)
		assert.Equal(t, defaultLabels, backup.Labels)
		assert.Equal(t, []string{backupv1.BackupFinalizer}, backup.Finalizers)
		assert.Equal(t, "blueprint", backup.Annotations[annotations.BlueprintIdAnnotation])
		assert.Equal(t, "dogus", backup.Annotations[annotations.DogusAnnotation])
		assert.NotContains(t, backup.Annotations, "unrelated")
	})

	t.Run("deletes the cloudogu backup after the Velero backup disappeared", func(t *testing.T) {
		backup := newBackupForControllerTest("ns", "backup")
		backup.Finalizers = []string{backupv1.BackupFinalizer}
		fakeClient := newFakeClientBuilder(t).WithObjects(backup).Build()
		reconciler := NewVeleroBackupReconciler(fakeClient)
		controller := NewVeleroBackupController(fakeClient, reconciler)

		result, err := controller.Reconcile(context.Background(), newReconcilerRequest("ns", "backup"))

		require.NoError(t, err)
		assert.Zero(t, result)
		deletingBackup := &backupv1.Backup{}
		err = fakeClient.Get(context.Background(), client.ObjectKeyFromObject(backup), deletingBackup)
		require.NoError(t, err)
		assert.False(t, deletingBackup.DeletionTimestamp.IsZero())
		assert.Contains(t, deletingBackup.Finalizers, backupv1.BackupFinalizer)
		deleteRequest := &velerov1.DeleteBackupRequest{}
		require.NoError(t, fakeClient.Get(context.Background(), client.ObjectKeyFromObject(backup), deleteRequest))
		assert.Equal(t, backup.Name, deleteRequest.Spec.BackupName)
	})

	t.Run("creates a delete request when the Velero backup starts deleting", func(t *testing.T) {
		deletionTimestamp := metav1.Now()
		veleroBackup := &velerov1.Backup{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "backup",
				Namespace:         "ns",
				DeletionTimestamp: &deletionTimestamp,
				Finalizers:        []string{"velero.io/test-finalizer"},
			},
		}
		backup := newBackupForControllerTest("ns", "backup")
		backup.Finalizers = []string{backupv1.BackupFinalizer}
		fakeClient := newFakeClientBuilder(t).WithObjects(veleroBackup, backup).Build()
		reconciler := NewVeleroBackupReconciler(fakeClient)
		controller := NewVeleroBackupController(fakeClient, reconciler)

		result, err := controller.Reconcile(context.Background(), newReconcilerRequest("ns", "backup"))

		require.NoError(t, err)
		assert.Zero(t, result)
		deleteRequest := &velerov1.DeleteBackupRequest{}
		require.NoError(t, fakeClient.Get(context.Background(), client.ObjectKeyFromObject(backup), deleteRequest))
		assert.Equal(t, backup.Name, deleteRequest.Spec.BackupName)
	})

	t.Run("does nothing if neither resource exists", func(t *testing.T) {
		fakeClient := newFakeClientBuilder(t).Build()
		reconciler := NewVeleroBackupReconciler(fakeClient)
		controller := NewVeleroBackupController(fakeClient, reconciler)

		result, err := controller.Reconcile(context.Background(), newReconcilerRequest("ns", "backup"))

		require.NoError(t, err)
		assert.Zero(t, result)
	})
}
