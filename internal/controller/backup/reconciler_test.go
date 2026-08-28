package backup

import (
	"context"
	"reflect"
	"testing"
	"time"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/k8s-backup-operator/internal/leases"
	operatortime "github.com/cloudogu/k8s-backup-operator/internal/time"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	coordinationv1 "k8s.io/api/coordination/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func newTestEventRecorder() eventRecorder {
	return record.NewFakeRecorder(100)
}

func newVeleroBackupForReconcilerTest(namespace string, name string, phase velerov1.BackupPhase) *velerov1.Backup {
	return &velerov1.Backup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Status: velerov1.BackupStatus{
			Phase: phase,
		},
	}
}

func newVeleroBackupStorageLocationForReconcilerTest(phase velerov1.BackupStorageLocationPhase) *velerov1.BackupStorageLocation {
	return &velerov1.BackupStorageLocation{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "default",
		},
		Status: velerov1.BackupStorageLocationStatus{
			Phase: phase,
		},
	}
}

type callCounter struct {
	configMapGetCount         int
	veleroBackupGetCount      int
	veleroBackupGetCallError  error
	veleroBackupCreateCount   int
	subResourcePatchCount     int
	subResourcePatchCallError error
	getCallError              error
	createCallError           error
}

func (c *callCounter) getCall(ctx context.Context, client client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	if c.getCallError != nil {
		return c.getCallError
	}
	if reflect.TypeOf(obj) == reflect.TypeFor[*corev1.ConfigMap]() {
		c.configMapGetCount++
	}
	if reflect.TypeOf(obj) == reflect.TypeFor[*velerov1.Backup]() {
		if c.veleroBackupGetCallError != nil {
			return c.veleroBackupGetCallError
		}
		c.veleroBackupGetCount++
	}
	return client.Get(ctx, key, obj, opts...)
}

func (c *callCounter) subResourcePatchCall(ctx context.Context, client client.Client, subResourceName string, obj client.Object, patch client.Patch, opts ...client.SubResourcePatchOption) error {
	if c.subResourcePatchCallError != nil {
		return c.subResourcePatchCallError
	}
	c.subResourcePatchCount++
	return client.SubResource(subResourceName).Patch(ctx, obj, patch, opts...)
}

func (c *callCounter) createCall(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
	if c.createCallError != nil {
		return c.createCallError
	}
	if reflect.TypeOf(obj) == reflect.TypeFor[*velerov1.Backup]() {
		c.veleroBackupCreateCount++
	}
	return client.Create(ctx, obj, opts...)
}

func TestBackupRunOutcome(t *testing.T) {
	succeeded := metav1.Condition{Type: backupv1.ConditionSucceeded, Status: metav1.ConditionTrue}
	failed := metav1.Condition{Type: backupv1.ConditionSucceeded, Status: metav1.ConditionFalse}
	running := metav1.Condition{Type: backupv1.ConditionSucceeded, Status: metav1.ConditionUnknown}
	canceled := metav1.Condition{Type: backupv1.ConditionCanceled, Status: metav1.ConditionTrue}

	for expected, conditions := range map[string][]metav1.Condition{
		"succeeded": {succeeded},
		"failed":    {failed},
		"canceled":  {running, canceled},
		"unknown":   {running},
	} {
		backup := newBackupForTest("ns", "backup")
		backup.Status.Conditions = conditions
		assert.Equal(t, expected, backupRunOutcome(backup))
	}

	t.Run("prefers a cancellation over the provider outcome", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		backup.Status.Conditions = []metav1.Condition{failed, canceled}
		assert.Equal(t, "canceled", backupRunOutcome(backup))
	})

	t.Run("reports an unknown outcome without conditions", func(t *testing.T) {
		assert.Equal(t, "unknown", backupRunOutcome(newBackupForTest("ns", "backup")))
	})
}

func TestBackupRunDuration(t *testing.T) {
	start := time.Date(2026, 8, 21, 5, 3, 0, 0, time.UTC)

	t.Run("reports the time between start and completion", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		backup.Status.StartTimestamp = metav1.NewTime(start)
		backup.Status.CompletionTimestamp = metav1.NewTime(start.Add(3 * time.Minute))

		assert.Equal(t, "3m0s", backupRunDuration(backup))
	})

	t.Run("reports an unknown duration while a timestamp is missing", func(t *testing.T) {
		backup := newBackupForTest("ns", "backup")
		assert.Equal(t, "unknown", backupRunDuration(backup))

		backup.Status.StartTimestamp = metav1.NewTime(start)
		assert.Equal(t, "unknown", backupRunDuration(backup))
	})
}

func newRealClock() Clock {
	return &operatortime.Clock{}
}

func newBackupWithSucceededStatusForReconcilerTest(namespace string, name string, conditionStatus metav1.ConditionStatus) *backupv1.Backup {
	return &backupv1.Backup{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Spec: backupv1.BackupSpec{
			Provider: "velero",
		},
		Status: backupv1.BackupStatus{
			Conditions: []metav1.Condition{
				{
					Type:   backupv1.ConditionSucceeded,
					Status: conditionStatus,
					Reason: "aReason",
				},
			},
		},
	}
}

func newBackupWithNoConditionsForReconcilerTest(namespace string, name string) *backupv1.Backup {
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

func newBackupWithProviderSucceededStatusForReconcilerTest(namespace string, name string, conditionStatus metav1.ConditionStatus) *backupv1.Backup {
	backup := newBackupWithNoConditionsForReconcilerTest(namespace, name)
	backup.Status.Conditions = []metav1.Condition{
		{
			Type:   backupv1.ConditionProviderSucceeded,
			Status: conditionStatus,
			Reason: "aReason",
		},
	}
	return backup
}

// newHeldBackupLeaseForTest builds the shared lease claimed by the given backup, so that the backup
// is its owner as far as leases.Manager.Holds is concerned.
func newHeldBackupLeaseForTest(backup *backupv1.Backup) *coordinationv1.Lease {
	backup.UID = types.UID(backup.Name + "-uid")
	return leases.NewLease(backup.Namespace, leases.DefaultName, backup, backupLeaseHolderKind)
}

func assertBackupLeaseStillHeldBy(t *testing.T, k8sClient client.Client, backup *backupv1.Backup) {
	t.Helper()
	lease := &coordinationv1.Lease{}
	require.NoError(t, k8sClient.Get(context.Background(), client.ObjectKey{Namespace: backup.Namespace, Name: leases.DefaultName}, lease))
	assert.True(t, leases.IsHolder(lease, backup, backupLeaseHolderKind), "expected backup %s to still hold the lease", backup.Name)
}

func assertBackupLeaseReleased(t *testing.T, k8sClient client.Client, backup *backupv1.Backup) {
	t.Helper()
	lease := &coordinationv1.Lease{}
	err := k8sClient.Get(context.Background(), client.ObjectKey{Namespace: backup.Namespace, Name: leases.DefaultName}, lease)
	assert.True(t, apierrors.IsNotFound(err), "expected the lease to be released, got %v", err)
}
