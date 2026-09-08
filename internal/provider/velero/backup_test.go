package velero

import (
	"context"
	"errors"
	"testing"

	k8sv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestCreateVeleroBackupResource(t *testing.T) {
	backup := &k8sv1.Backup{ObjectMeta: metav1.ObjectMeta{
		Name: testBackup, Namespace: testNamespace,
		Annotations: map[string]string{"backup.cloudogu.com/blueprintId": "blueprint-id"},
	}}
	labels := map[string]string{"app": "ces", "k8s.cloudogu.com/part-of": "backup"}

	actual := CreateVeleroBackupResource(backup, "backup-storage", labels)

	assert.Equal(t, backup.Name, actual.Name)
	assert.Equal(t, backup.Namespace, actual.Namespace)
	assert.Equal(t, labels, actual.Labels)
	assert.Equal(t, backup.Annotations, actual.Annotations)
	assert.Equal(t, []string{backup.Namespace}, actual.Spec.IncludedNamespaces)
	assert.Equal(t, []string{"configmaps", "secrets", "persistentvolumeclaims", "persistentvolumes", "dogus.k8s.cloudogu.com"}, actual.Spec.IncludedResources)
	require.Len(t, actual.Spec.OrLabelSelectors, 3)
	assert.Equal(t, map[string]string{"k8s.cloudogu.com/type": "global-config"}, actual.Spec.OrLabelSelectors[0].MatchLabels)
	assert.Equal(t, "dogu.name", actual.Spec.OrLabelSelectors[1].MatchExpressions[0].Key)
	assert.Equal(t, "k8s.cloudogu.com/backup-scope", actual.Spec.OrLabelSelectors[2].MatchExpressions[0].Key)
	assert.Equal(t, metav1.LabelSelectorOpExists, actual.Spec.OrLabelSelectors[1].MatchExpressions[0].Operator)
	assert.Equal(t, metav1.LabelSelectorOpExists, actual.Spec.OrLabelSelectors[2].MatchExpressions[0].Operator)
	assert.Equal(t, metav1.Duration{Duration: defaultBackupTTL}, actual.Spec.TTL)
	assert.Equal(t, "backup-storage", actual.Spec.StorageLocation)
	require.NotNil(t, actual.Spec.DefaultVolumesToFsBackup)
	assert.False(t, *actual.Spec.DefaultVolumesToFsBackup)
}

func TestCreateVeleroDeleteBackupRequestIfNotExists(t *testing.T) {
	backup := &k8sv1.Backup{ObjectMeta: metav1.ObjectMeta{Name: testBackup, Namespace: testNamespace}}

	t.Run("creates a missing request", func(t *testing.T) {
		writes := &writeCounter{}
		k8sClient := newTestClient(t, writes, backup)
		actual, err := CreateVeleroDeleteBackupRequestIfNotExists(context.Background(), k8sClient, backup)
		require.NoError(t, err)
		assert.Equal(t, backup.Name, actual.Name)
		assert.Equal(t, backup.Namespace, actual.Namespace)
		assert.Equal(t, backup.Name, actual.Spec.BackupName)
		assert.Equal(t, 1, writes.creates)
		assert.Equal(t, 1, writes.total())
	})

	t.Run("returns an existing request without writing", func(t *testing.T) {
		existing := &velerov1.DeleteBackupRequest{ObjectMeta: metav1.ObjectMeta{Name: backup.Name, Namespace: backup.Namespace}, Spec: velerov1.DeleteBackupRequestSpec{BackupName: backup.Name}}
		writes := &writeCounter{}
		k8sClient := newTestClient(t, writes, backup, existing)
		actual, err := CreateVeleroDeleteBackupRequestIfNotExists(context.Background(), k8sClient, backup)
		require.NoError(t, err)
		assert.Equal(t, existing.Spec, actual.Spec)
		assert.Zero(t, writes.total())
	})

	t.Run("wraps get errors", func(t *testing.T) {
		writes := &writeCounter{}
		wrapped := newTestClient(t, writes, backup)
		k8sClient := interceptor.NewClient(wrapped, interceptor.Funcs{Get: func(context.Context, client.WithWatch, client.ObjectKey, client.Object, ...client.GetOption) error {
			return errors.New("get failed")
		}})
		actual, err := CreateVeleroDeleteBackupRequestIfNotExists(context.Background(), k8sClient, backup)
		assert.Nil(t, actual)
		assert.EqualError(t, err, "get velero delete backup request: get failed")
		assert.Zero(t, writes.total())
	})

	t.Run("wraps create errors", func(t *testing.T) {
		writes := &writeCounter{}
		wrapped := newTestClient(t, writes, backup)
		k8sClient := interceptor.NewClient(wrapped, interceptor.Funcs{Create: func(context.Context, client.WithWatch, client.Object, ...client.CreateOption) error {
			return errors.New("create failed")
		}})
		actual, err := CreateVeleroDeleteBackupRequestIfNotExists(context.Background(), k8sClient, backup)
		assert.Nil(t, actual)
		assert.EqualError(t, err, "create velero delete backup request: create failed")
	})
}
