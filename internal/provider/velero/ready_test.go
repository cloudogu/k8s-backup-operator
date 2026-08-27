package velero

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	k8sv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
)

const testBackupStorage = "test-backup-storage"

func backupStorageLocation(phase velerov1.BackupStorageLocationPhase) *velerov1.BackupStorageLocation {
	return &velerov1.BackupStorageLocation{
		ObjectMeta: metav1.ObjectMeta{Name: testBackupStorage, Namespace: testNamespace},
		Status:     velerov1.BackupStorageLocationStatus{Phase: phase},
	}
}

func TestCheckReadyReportsAnAvailableBackupStorageLocation(t *testing.T) {
	k8sClient := newTestClient(t, &writeCounter{}, backupStorageLocation(velerov1.BackupStorageLocationPhaseAvailable))

	readiness, err := CheckReady(testCtx, k8sClient, testNamespace, testBackupStorage)

	require.NoError(t, err)
	assert.True(t, readiness.Ready)
	assert.Equal(t, ReasonBackupStorageLocationAvailable, readiness.Reason)
	assert.Contains(t, readiness.Message, testBackupStorage)
}

func TestCheckReadyReportsAMissingBackupStorageLocationWithoutAnError(t *testing.T) {
	k8sClient := newTestClient(t, &writeCounter{})

	readiness, err := CheckReady(testCtx, k8sClient, testNamespace, testBackupStorage)

	require.NoError(t, err, "a provider that is not installed yet is not an error")
	assert.False(t, readiness.Ready)
	assert.Equal(t, ReasonBackupStorageLocationNotFound, readiness.Reason)
	assert.Contains(t, readiness.Message, testBackupStorage)
}

func TestCheckReadyReportsEveryPhaseButAvailableAsNotReady(t *testing.T) {
	for _, phase := range []velerov1.BackupStorageLocationPhase{
		velerov1.BackupStorageLocationPhaseUnavailable,
		"",
	} {
		t.Run(string(phase), func(t *testing.T) {
			k8sClient := newTestClient(t, &writeCounter{}, backupStorageLocation(phase))

			readiness, err := CheckReady(testCtx, k8sClient, testNamespace, testBackupStorage)

			require.NoError(t, err)
			assert.False(t, readiness.Ready)
			assert.Equal(t, ReasonBackupStorageLocationNotAvailable, readiness.Reason)
			assert.Contains(t, readiness.Message, string(phase))
		})
	}
}

func TestCheckReadyReportsAFailedReadAsAnError(t *testing.T) {
	testScheme := runtime.NewScheme()
	require.NoError(t, k8sv1.AddToScheme(testScheme))
	require.NoError(t, velerov1.AddToScheme(testScheme))
	k8sClient := fake.NewClientBuilder().
		WithScheme(testScheme).
		WithInterceptorFuncs(interceptor.Funcs{
			Get: func(_ context.Context, _ client.WithWatch, _ client.ObjectKey, _ client.Object, _ ...client.GetOption) error {
				return assert.AnError
			},
		}).
		Build()

	readiness, err := CheckReady(testCtx, k8sClient, testNamespace, testBackupStorage)

	require.ErrorIs(t, err, assert.AnError)
	assert.ErrorContains(t, err, "get velero backup storage location 'name=test-backup-storage'")
	assert.False(t, readiness.Ready)
}
