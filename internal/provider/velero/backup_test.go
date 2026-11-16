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

func TestCreateVeleroUnitBackupResource(t *testing.T) {
	labels := map[string]string{"app": "ces", "k8s.cloudogu.com/backup": testBackup}
	annotations := map[string]string{"k8s.cloudogu.com/backup-units": `[{"type":"misc"}]`}
	selector := &metav1.LabelSelector{MatchLabels: map[string]string{"k8s.cloudogu.com/dogu.name": "ldap"}}
	orSelectors := []*metav1.LabelSelector{{MatchLabels: map[string]string{"k8s.cloudogu.com/dogu.name": "ldap"}}, {MatchLabels: map[string]string{"dogu.name": "ldap"}}}
	includedResources := []string{"configmaps", "secrets", "persistentvolumeclaims", "dogus.k8s.cloudogu.com"}

	successTests := []struct {
		name        string
		selector    *metav1.LabelSelector
		orSelectors []*metav1.LabelSelector
	}{
		{name: "success with selector", selector: selector},
		{name: "success with orSelectors", orSelectors: orSelectors},
	}
	for _, tt := range successTests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := CreateVeleroUnitBackupResource(testNamespace, "unit-name", "backup-storage", labels, annotations, tt.selector, tt.orSelectors, includedResources)

			require.NoError(t, err)
			assert.Equal(t, "unit-name", actual.Name)
			assert.Equal(t, testNamespace, actual.Namespace)
			assert.Equal(t, labels, actual.Labels)
			assert.Equal(t, annotations, actual.Annotations)
			assert.Equal(t, []string{testNamespace}, actual.Spec.IncludedNamespaces)
			assert.Equal(t, includedResources, actual.Spec.IncludedResources)
			assert.Equal(t, tt.selector, actual.Spec.LabelSelector)
			assert.Equal(t, tt.orSelectors, actual.Spec.OrLabelSelectors)
			assert.Equal(t, "backup-storage", actual.Spec.StorageLocation)
			assert.Equal(t, defaultBackupTTL, actual.Spec.TTL.Duration)
			require.NotNil(t, actual.Spec.DefaultVolumesToFsBackup)
			assert.False(t, *actual.Spec.DefaultVolumesToFsBackup)
		})
	}

	t.Run("should return error if selector and orSelectors are both set", func(t *testing.T) {
		_, err := CreateVeleroUnitBackupResource(testNamespace, "unit-name", "backup-storage", labels, annotations, selector, orSelectors, includedResources)

		require.Error(t, err)
		assert.ErrorContains(t, err, "selector and orSelectors must not both be set")
	})
}

func TestCreateVeleroDeleteBackupRequestIfNotExists(t *testing.T) {
	backup := &k8sv1.Backup{Name: testBackup, Namespace: testNamespace}

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
		existing := &velerov1.DeleteBackupRequest{Name: backup.Name, Namespace: backup.Namespace, Spec: velerov1.DeleteBackupRequestSpec{BackupName: backup.Name}}
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
