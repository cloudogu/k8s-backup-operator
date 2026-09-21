package velero

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	k8sv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
)

func sourceBackup(phase velerov1.BackupPhase) *velerov1.Backup {
	return &velerov1.Backup{
		ObjectMeta: metav1.ObjectMeta{Name: testBackup, Namespace: testNamespace},
		Status:     velerov1.BackupStatus{Phase: phase},
	}
}

func TestCheckSourceBackupAcceptsEveryFinishedBackup(t *testing.T) {
	for _, phase := range []velerov1.BackupPhase{
		velerov1.BackupPhaseCompleted,
		velerov1.BackupPhasePartiallyFailed,
	} {
		t.Run(string(phase), func(t *testing.T) {
			k8sClient := newTestClient(t, &writeCounter{}, sourceBackup(phase))

			check, err := CheckSourceBackup(testCtx, k8sClient, testNamespace, testBackup)

			require.NoError(t, err)
			assert.True(t, check.Usable, "a partially failed backup still carries a recovery point")
			assert.Equal(t, ReasonVeleroSourceBackupUsable, check.Reason)
			assert.Contains(t, check.Message, testBackup)
		})
	}
}

func TestCheckSourceBackupRefusesEveryUnfinishedBackupAndNamesThePhase(t *testing.T) {
	for _, phase := range []velerov1.BackupPhase{
		"",
		velerov1.BackupPhaseNew,
		velerov1.BackupPhaseInProgress,
		velerov1.BackupPhaseWaitingForPluginOperations,
		velerov1.BackupPhaseWaitingForPluginOperationsPartiallyFailed,
		velerov1.BackupPhaseFinalizing,
		velerov1.BackupPhaseFinalizingPartiallyFailed,
		velerov1.BackupPhaseFailed,
		velerov1.BackupPhaseFailedValidation,
		velerov1.BackupPhaseDeleting,
		"SomePhaseALaterVeleroAdded",
	} {
		t.Run(string(phase), func(t *testing.T) {
			k8sClient := newTestClient(t, &writeCounter{}, sourceBackup(phase))

			check, err := CheckSourceBackup(testCtx, k8sClient, testNamespace, testBackup)

			require.NoError(t, err)
			assert.False(t, check.Usable)
			assert.Equal(t, ReasonVeleroSourceBackupNotUsable, check.Reason)
			assert.Contains(t, check.Message, string(phase))
		})
	}
}

func TestCheckSourceBackupReportsAMissingBackupWithoutAnError(t *testing.T) {
	k8sClient := newTestClient(t, &writeCounter{})

	check, err := CheckSourceBackup(testCtx, k8sClient, testNamespace, testBackup)

	require.NoError(t, err)
	assert.False(t, check.Usable)
	assert.Equal(t, ReasonVeleroSourceBackupNotUsable, check.Reason)
	assert.Contains(t, check.Message, testBackup)
}

func TestCheckSourceBackupReportsAFailedReadAsAnError(t *testing.T) {
	testScheme := runtime.NewScheme()
	require.NoError(t, k8sv1.AddToScheme(testScheme))
	require.NoError(t, velerov1.AddToScheme(testScheme))
	require.NoError(t, appsv1.AddToScheme(testScheme))
	k8sClient := fake.NewClientBuilder().
		WithScheme(testScheme).
		WithInterceptorFuncs(interceptor.Funcs{
			Get: func(_ context.Context, _ client.WithWatch, _ client.ObjectKey, _ client.Object, _ ...client.GetOption) error {
				return assert.AnError
			},
		}).
		Build()

	check, err := CheckSourceBackup(testCtx, k8sClient, testNamespace, testBackup)

	require.ErrorIs(t, err, assert.AnError)
	assert.ErrorContains(t, err, "get velero backup 'name=test-backup'")
	assert.False(t, check.Usable)
}
