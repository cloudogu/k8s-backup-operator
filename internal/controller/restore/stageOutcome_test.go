package restore

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ctrl "sigs.k8s.io/controller-runtime"

	k8sv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
)

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
			name:       "controlled retry requeues after the given delay",
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
			result, err := test.outcome.result()

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
		"zero delay":     retryAfter(0),
		"negative delay": retryAfter(-time.Minute),
		"nil error":      retryOnError(nil),
	}

	for name, outcome := range tests {
		t.Run(name, func(t *testing.T) {
			result, err := outcome.result()

			require.NoError(t, err)
			assert.Equal(t, defaultRequeueDelay, result.RequeueAfter)
		})
	}
}

func TestRunStagesRunsOrderedStagesUntilOneDoesNotContinue(t *testing.T) {
	var executed []string
	stage := func(name string, outcome stageOutcome) reconcileStage {
		return func(_ context.Context, _ *k8sv1.Restore) stageOutcome {
			executed = append(executed, name)

			return outcome
		}
	}

	tests := []struct {
		name         string
		stages       []reconcileStage
		wantExecuted []string
		wantResult   ctrl.Result
		wantErr      error
	}{
		{
			name:         "all stages continue",
			stages:       []reconcileStage{stage("first", next()), stage("second", next())},
			wantExecuted: []string{"first", "second"},
			wantResult:   ctrl.Result{},
		},
		{
			name:         "a retrying stage stops the following stages",
			stages:       []reconcileStage{stage("first", retryAfter(time.Minute)), stage("second", next())},
			wantExecuted: []string{"first"},
			wantResult:   ctrl.Result{RequeueAfter: time.Minute},
		},
		{
			name:         "an aborting stage stops the following stages",
			stages:       []reconcileStage{stage("first", next()), stage("second", abort()), stage("third", next())},
			wantExecuted: []string{"first", "second"},
			wantResult:   ctrl.Result{},
		},
		{
			name:         "a failing stage stops the following stages and reports the error",
			stages:       []reconcileStage{stage("first", retryOnError(assert.AnError)), stage("second", next())},
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

			result, err := runStages(testCtx, restoreWith(""), test.stages...)

			assert.Equal(t, test.wantExecuted, executed)
			assert.Equal(t, test.wantResult, result)
			assert.Equal(t, test.wantErr, err)
		})
	}
}
