package backup

import (
	"context"
	"testing"
	"time"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ctrl "sigs.k8s.io/controller-runtime"
)

// stageTestRequeueDelay stands in for the delay the controller injects into runStages.
const stageTestRequeueDelay = 5 * time.Second

func backupWithStatus(status string) *backupv1.Backup {
	return &backupv1.Backup{Status: backupv1.BackupStatus{Status: status}}
}

func TestStageOutcomeNeverCombinesAnErrorWithAnExplicitRequeue(t *testing.T) {
	tests := []struct {
		name       string
		outcome    stageOutcome
		wantResult ctrl.Result
		wantErr    error
	}{
		{
			name:       "next does not requeue",
			outcome:    next(),
			wantResult: ctrl.Result{},
		},
		{
			name:       "abort does not requeue",
			outcome:    abort(),
			wantResult: ctrl.Result{},
		},
		{
			name:       "controlled retry requeues at the controller's cadence",
			outcome:    retry(),
			wantResult: ctrl.Result{RequeueAfter: stageTestRequeueDelay},
		},
		{
			name:       "a domain delay requeues after that delay",
			outcome:    retryAfter(30 * time.Second),
			wantResult: ctrl.Result{RequeueAfter: 30 * time.Second},
		},
		{
			name:       "transient failure returns the error and no requeue",
			outcome:    retryOnError(assert.AnError),
			wantResult: ctrl.Result{},
			wantErr:    assert.AnError,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := test.outcome.result(stageTestRequeueDelay)

			assert.Equal(t, test.wantResult, result)
			assert.Equal(t, test.wantErr, err)
			if err != nil {
				assert.Zero(t, result.RequeueAfter, "an error must rely on the controller-runtime backoff")
			}
		})
	}
}

func TestStageOutcomeCannotSilentlyDropARetry(t *testing.T) {
	tests := map[string]stageOutcome{
		"no delay":       retry(),
		"zero delay":     retryAfter(0),
		"negative delay": retryAfter(-time.Minute),
		"nil error":      retryOnError(nil),
	}

	for name, outcome := range tests {
		t.Run(name, func(t *testing.T) {
			result, err := outcome.result(stageTestRequeueDelay)

			require.NoError(t, err)
			assert.Equal(t, stageTestRequeueDelay, result.RequeueAfter)
		})
	}
}

func TestRunStagesRunsOrderedStagesUntilOneDoesNotContinue(t *testing.T) {
	var executed []string
	newStage := func(name string, outcome stageOutcome) stage {
		return func(_ context.Context, backup *backupv1.Backup) (*backupv1.Backup, stageOutcome) {
			executed = append(executed, name)

			return backup, outcome
		}
	}

	tests := []struct {
		name         string
		stages       []stage
		wantExecuted []string
		wantResult   ctrl.Result
		wantErr      error
	}{
		{
			name:         "all stages continue",
			stages:       []stage{newStage("first", next()), newStage("second", next())},
			wantExecuted: []string{"first", "second"},
			wantResult:   ctrl.Result{},
		},
		{
			name:         "a retrying stage stops the following stages",
			stages:       []stage{newStage("first", retryAfter(time.Minute)), newStage("second", next())},
			wantExecuted: []string{"first"},
			wantResult:   ctrl.Result{RequeueAfter: time.Minute},
		},
		{
			name:         "an aborting stage stops the following stages",
			stages:       []stage{newStage("first", next()), newStage("second", abort()), newStage("third", next())},
			wantExecuted: []string{"first", "second"},
			wantResult:   ctrl.Result{},
		},
		{
			name:         "a failing stage stops the following stages and reports the error",
			stages:       []stage{newStage("first", retryOnError(assert.AnError)), newStage("second", next())},
			wantExecuted: []string{"first"},
			wantResult:   ctrl.Result{},
			wantErr:      assert.AnError,
		},
		{
			name:         "no stages at all",
			stages:       nil,
			wantExecuted: nil,
			wantResult:   ctrl.Result{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			executed = nil

			result, err := runStages(context.Background(), backupWithStatus(""), stageTestRequeueDelay, test.stages...)

			assert.Equal(t, test.wantExecuted, executed)
			assert.Equal(t, test.wantResult, result)
			assert.Equal(t, test.wantErr, err)
		})
	}
}

func TestRunStagesThreadsTheBackupThroughTheStages(t *testing.T) {
	replaced := backupWithStatus("replaced")
	var seen []string

	replacing := func(_ context.Context, backup *backupv1.Backup) (*backupv1.Backup, stageOutcome) {
		seen = append(seen, backup.Status.Status)

		return replaced, next()
	}
	// A stage that returns no backup must not undo the replacement done by an earlier one.
	returningNothing := func(_ context.Context, backup *backupv1.Backup) (*backupv1.Backup, stageOutcome) {
		seen = append(seen, backup.Status.Status)

		return nil, next()
	}
	observing := func(_ context.Context, backup *backupv1.Backup) (*backupv1.Backup, stageOutcome) {
		seen = append(seen, backup.Status.Status)

		return backup, abort()
	}

	_, err := runStages(context.Background(), backupWithStatus("initial"), stageTestRequeueDelay, replacing, returningNothing, observing)

	require.NoError(t, err)
	assert.Equal(t, []string{"initial", "replaced", "replaced"}, seen)
}
