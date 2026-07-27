package restore

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
)

func restoreWith(legacyStatus string, conditions ...metav1.Condition) *backupv1.Restore {
	return &backupv1.Restore{
		ObjectMeta: metav1.ObjectMeta{Name: testRestore, Namespace: testNamespace},
		Status:     backupv1.RestoreStatus{Status: legacyStatus, Conditions: conditions},
	}
}

func successful(status metav1.ConditionStatus, reason string) metav1.Condition {
	return metav1.Condition{Type: backupv1.ConditionSuccessful, Status: status, Reason: reason}
}

func TestLegacyStatusForNeverRegressesAnObservedRestoreToNew(t *testing.T) {
	deleting := restoreWith("", successful(metav1.ConditionUnknown, ReasonPreparing))
	deleting.DeletionTimestamp = &metav1.Time{Time: metav1.Now().Time}

	tests := []struct {
		name    string
		restore *backupv1.Restore
		want    string
	}{
		{
			name:    "untouched restore stays new",
			restore: restoreWith(""),
			want:    backupv1.RestoreStatusNew,
		},
		{
			name:    "successful restore is completed",
			restore: restoreWith("", successful(metav1.ConditionTrue, ReasonRestoreCompleted)),
			want:    backupv1.RestoreStatusCompleted,
		},
		{
			name:    "terminally failed restore is failed",
			restore: restoreWith("", successful(metav1.ConditionFalse, ReasonVeleroRestoreFailed)),
			want:    backupv1.RestoreStatusFailed,
		},
		{
			name:    "unknown outcome is in progress",
			restore: restoreWith("", successful(metav1.ConditionUnknown, ReasonPreparing)),
			want:    backupv1.RestoreStatusInProgress,
		},
		{
			name:    "waiting for another restore is in progress, not new",
			restore: restoreWith("", successful(metav1.ConditionUnknown, ReasonWaitingForActiveRestore)),
			want:    backupv1.RestoreStatusInProgress,
		},
		{
			name: "milestone condition without an outcome is in progress, not new",
			restore: restoreWith("", metav1.Condition{
				Type:   backupv1.ConditionPrepared,
				Status: metav1.ConditionTrue,
				Reason: ReasonPreparing,
			}),
			want: backupv1.RestoreStatusInProgress,
		},
		{
			name:    "deletion wins over the outcome condition",
			restore: deleting,
			want:    backupv1.RestoreStatusDeleting,
		},
		{
			name: "a milestone write does not regress a terminally completed legacy restore",
			restore: restoreWith(backupv1.RestoreStatusCompleted, metav1.Condition{
				Type:   backupv1.ConditionPrepared,
				Status: metav1.ConditionTrue,
				Reason: ReasonPreparing,
			}),
			want: backupv1.RestoreStatusCompleted,
		},
		{
			name: "a milestone write does not regress a terminally failed legacy restore",
			restore: restoreWith(backupv1.RestoreStatusFailed, metav1.Condition{
				Type:   backupv1.ConditionWorkloadsRecovered,
				Status: metav1.ConditionFalse,
				Reason: ReasonRecoveryNotAttemptedAfterProviderFailure,
			}),
			want: backupv1.RestoreStatusFailed,
		},
		{
			name: "a milestone write on a legacy in progress restore keeps it in progress",
			restore: restoreWith(backupv1.RestoreStatusInProgress, metav1.Condition{
				Type:   backupv1.ConditionPrepared,
				Status: metav1.ConditionTrue,
				Reason: ReasonPreparing,
			}),
			want: backupv1.RestoreStatusInProgress,
		},
		{
			name: "an outcome condition still overrides a terminal legacy status",
			restore: restoreWith(backupv1.RestoreStatusFailed,
				successful(metav1.ConditionTrue, ReasonRestoreCompleted)),
			want: backupv1.RestoreStatusCompleted,
		},
		{
			name:    "legacy in progress restore without conditions keeps its status",
			restore: restoreWith(backupv1.RestoreStatusInProgress),
			want:    backupv1.RestoreStatusInProgress,
		},
		{
			name:    "legacy completed restore without conditions keeps its status",
			restore: restoreWith(backupv1.RestoreStatusCompleted),
			want:    backupv1.RestoreStatusCompleted,
		},
		{
			name:    "conditions override a stale legacy status",
			restore: restoreWith(backupv1.RestoreStatusInProgress, successful(metav1.ConditionTrue, ReasonRestoreCompleted)),
			want:    backupv1.RestoreStatusCompleted,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, legacyStatusFor(test.restore))
		})
	}
}

func TestLegacySuccessfulConditionInterpretsRestoresCreatedBeforeConditions(t *testing.T) {
	tests := []struct {
		name       string
		restore    *backupv1.Restore
		wantStatus metav1.ConditionStatus
	}{
		{
			name:       "legacy completed becomes a successful outcome",
			restore:    restoreWith(backupv1.RestoreStatusCompleted),
			wantStatus: metav1.ConditionTrue,
		},
		{
			name:       "legacy failed becomes a terminally failed outcome",
			restore:    restoreWith(backupv1.RestoreStatusFailed),
			wantStatus: metav1.ConditionFalse,
		},
		{
			name:       "legacy in progress becomes an unknown outcome",
			restore:    restoreWith(backupv1.RestoreStatusInProgress),
			wantStatus: metav1.ConditionUnknown,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			condition := determineLegacySuccessfulCondition(test.restore)

			require.NotNil(t, condition)
			assert.Equal(t, backupv1.ConditionSuccessful, condition.Type)
			assert.Equal(t, test.wantStatus, condition.Status)
			assert.Equal(t, ReasonMigratedFromLegacyStatus, condition.Reason)
			assert.NotEmpty(t, condition.Message)
		})
	}
}

func TestLegacySuccessfulConditionCarriesNoOutcomeForNewAndDeletingRestores(t *testing.T) {
	assert.Nil(t, determineLegacySuccessfulCondition(restoreWith(backupv1.RestoreStatusNew)))
	assert.Nil(t, determineLegacySuccessfulCondition(restoreWith(backupv1.RestoreStatusDeleting)))
	assert.Nil(t, determineLegacySuccessfulCondition(restoreWith("something an older operator never wrote")))
}

func TestEffectiveSuccessfulConditionPrefersTheWrittenCondition(t *testing.T) {
	restore := restoreWith(backupv1.RestoreStatusFailed, successful(metav1.ConditionTrue, ReasonRestoreCompleted))

	condition := effectiveSuccessfulCondition(restore)

	require.NotNil(t, condition)
	assert.Equal(t, metav1.ConditionTrue, condition.Status)
	assert.Equal(t, ReasonRestoreCompleted, condition.Reason)
}

func TestIsTerminalKeepsExistingTerminalLegacyRestoresOutOfNewWork(t *testing.T) {
	tests := []struct {
		name    string
		restore *backupv1.Restore
		want    bool
	}{
		{
			name:    "legacy completed restore is terminal",
			restore: restoreWith(backupv1.RestoreStatusCompleted),
			want:    true,
		},
		{
			name:    "legacy failed restore is terminal",
			restore: restoreWith(backupv1.RestoreStatusFailed),
			want:    true,
		},
		{
			name:    "legacy in progress restore is not terminal",
			restore: restoreWith(backupv1.RestoreStatusInProgress),
			want:    false,
		},
		{
			name:    "new restore is not terminal",
			restore: restoreWith(backupv1.RestoreStatusNew),
			want:    false,
		},
		{
			name:    "successful restore is terminal",
			restore: restoreWith("", successful(metav1.ConditionTrue, ReasonRestoreCompleted)),
			want:    true,
		},
		{
			name:    "failed restore is terminal",
			restore: restoreWith("", successful(metav1.ConditionFalse, ReasonPreparationFailed)),
			want:    true,
		},
		{
			name:    "running restore is not terminal",
			restore: restoreWith("", successful(metav1.ConditionUnknown, ReasonVeleroRestoreRunning)),
			want:    false,
		},
		{
			name: "milestone conditions alone are not terminal",
			restore: restoreWith("", metav1.Condition{
				Type:   backupv1.ConditionWorkloadsRecovered,
				Status: metav1.ConditionTrue,
				Reason: ReasonRecoveringWorkloads,
			}),
			want: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, isTerminal(test.restore))
		})
	}
}
