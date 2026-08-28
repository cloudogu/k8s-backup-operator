package restore

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	k8sv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
)

const (
	testRestoreUID    = types.UID("11111111-1111-1111-1111-111111111111")
	testBackup        = "test-backup"
	testBackupStorage = "test-backup-storage"
)

// recoverableRestore is a Restore whose provider restore succeeded, so workload recovery can start.
func recoverableRestore() *k8sv1.Restore {
	return withProviderRestoreSuccess(startableRestore())
}

func newParentRestore() *k8sv1.Restore {
	return &k8sv1.Restore{
		ObjectMeta: metav1.ObjectMeta{Name: testRestore, Namespace: testNamespace, UID: testRestoreUID},
		Spec:       k8sv1.RestoreSpec{BackupName: testBackup, Provider: k8sv1.ProviderVelero},
	}
}

// withMetadata adds the finalizer and the labels the metadata stage writes, so that a test
// starting behind that stage does not have to reconcile it first.
func withMetadata(restore *k8sv1.Restore) *k8sv1.Restore {
	restore.Finalizers = []string{k8sv1.RestoreFinalizer}
	restore.Labels = restoreLabels()

	return restore
}

// withInitializedConditions adds the Unknown workflow conditions the condition stage writes, so that
// a test starting behind that stage does not have to reconcile it first.
func withInitializedConditions(restore *k8sv1.Restore) *k8sv1.Restore {
	applyConditions(restore, missingWorkflowConditions(restore))

	return restore
}

// withPreparation adds the Prepared milestone the preparation stage writes, so that a test starting
// behind that stage does not have to reconcile the destructive preparation first.
func withPreparation(restore *k8sv1.Restore) *k8sv1.Restore {
	applyConditions(restore, []metav1.Condition{{
		Type:    k8sv1.ConditionPrepared,
		Status:  metav1.ConditionTrue,
		Reason:  ReasonPreparationCompleted,
		Message: "The ecosystem was prepared for the restore.",
	}})

	return restore
}

// withProviderRestoreSuccess adds the ProviderRestoreSuccessful milestone the completion stage
// writes, so that a test starting behind that stage does not have to run a provider restore first.
func withProviderRestoreSuccess(restore *k8sv1.Restore) *k8sv1.Restore {
	applyConditions(restore, []metav1.Condition{{
		Type:    k8sv1.ConditionProviderSucceeded,
		Status:  metav1.ConditionTrue,
		Reason:  ReasonProviderRestoreCompleted,
		Message: "The provider restored the backup.",
	}})

	return restore
}

func readyStorageLocation() *velerov1.BackupStorageLocation {
	return backupStorageLocation(velerov1.BackupStorageLocationPhaseAvailable)
}

func backupStorageLocation(phase velerov1.BackupStorageLocationPhase) *velerov1.BackupStorageLocation {
	return &velerov1.BackupStorageLocation{
		ObjectMeta: metav1.ObjectMeta{Name: testBackupStorage, Namespace: testNamespace},
		Status:     velerov1.BackupStorageLocationStatus{Phase: phase},
	}
}

func deletedRestore() *k8sv1.Restore {
	return &k8sv1.Restore{ObjectMeta: metav1.ObjectMeta{
		Name:              testRestore,
		Namespace:         testNamespace,
		DeletionTimestamp: &metav1.Time{Time: time.Now()},
		Finalizers:        []string{k8sv1.RestoreFinalizer},
	}}
}

// assertPersistedMetadata asserts the finalizer and the labels the create flow has to apply.
func assertPersistedMetadata(t *testing.T, testClient client.Client, name string) {
	t.Helper()

	stored := &k8sv1.Restore{}
	require.NoError(t, testClient.Get(context.Background(), client.ObjectKey{Namespace: testNamespace, Name: name}, stored))

	assert.Contains(t, stored.Finalizers, k8sv1.RestoreFinalizer)
	for key, value := range restoreLabels() {
		assert.Equal(t, value, stored.Labels[key], "label %s", key)
	}
}

// assertPersistedCondition asserts the status and the reason of one condition of the Restore under
// test, as it was persisted through the given client.
func assertPersistedCondition(t *testing.T, testClient client.Client, conditionType string, status metav1.ConditionStatus, reason string) {
	t.Helper()

	stored := &k8sv1.Restore{}
	require.NoError(t, testClient.Get(context.Background(), client.ObjectKey{Namespace: testNamespace, Name: testRestore}, stored))

	condition := meta.FindStatusCondition(stored.Status.Conditions, conditionType)
	require.NotNil(t, condition, "no %s condition was persisted", conditionType)
	assert.Equal(t, status, condition.Status, "status of the %s condition", conditionType)
	assert.Equal(t, reason, condition.Reason, "reason of the %s condition", conditionType)
}

// assertSuccessfulCondition asserts the Successful condition persisted through the given client.
func assertSuccessfulCondition(t *testing.T, testClient client.Client, name string, status metav1.ConditionStatus, reason string) {
	t.Helper()

	stored := &k8sv1.Restore{}
	require.NoError(t, testClient.Get(context.Background(), client.ObjectKey{Namespace: testNamespace, Name: name}, stored))

	condition := findSuccessfulCondition(stored)
	require.NotNil(t, condition, "no Successful condition was persisted")
	require.Equal(t, status, condition.Status)
	require.Equal(t, reason, condition.Reason)
}

// matchesRestoreNamed matches a Restore by name, for mocks that see the object as the client returned
// it rather than the exact pointer the test built.
func matchesRestoreNamed(name string) any {
	return mock.MatchedBy(func(restore *k8sv1.Restore) bool {
		return restore != nil && restore.Name == name
	})
}
