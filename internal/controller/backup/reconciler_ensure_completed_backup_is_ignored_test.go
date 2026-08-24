package backup

import (
	"context"
	"testing"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestReconcilerEnsureCompletedBackupIsIgnored(t *testing.T) {
	tests := []struct {
		name            string
		succeededStatus metav1.ConditionStatus
		action          action
	}{
		{"Ignores failed backups", metav1.ConditionFalse, Abort},
		{"Ignores succeeded backups", metav1.ConditionTrue, Abort},
		{"Continues with backup that are not completed yet.", metav1.ConditionUnknown, Next},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			backup := newBackupWithSucceededStatusForReconcilerTest("ns", "backup", test.succeededStatus)
			fakeClient := newFakeClientBuilder(t).Build()
			maintenanceGatewayMock := newMockMaintenanceGateway(t)

			reconciler := NewReconciler(fakeClient, maintenanceGatewayMock, newRealClock(), "default")

			nextAction, err := reconciler.ensureCompletedBackupIsIgnored(context.Background(), backup)

			assert.NoError(t, err)
			assert.Equal(t, test.action, nextAction)
		})

	}

	t.Run("If the backup has no succeeded condition then continue the backup", func(t *testing.T) {
		backup := newBackupWithNoConditionsForReconcilerTest("ns", "backup")
		fakeClient := newFakeClientBuilder(t).Build()
		maintenanceGatewayMock := newMockMaintenanceGateway(t)

		reconciler := NewReconciler(fakeClient, maintenanceGatewayMock, newRealClock(), "default")

		nextAction, err := reconciler.ensureCompletedBackupIsIgnored(context.Background(), backup)

		assert.NoError(t, err)
		assert.Equal(t, Next, nextAction)
	})
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
