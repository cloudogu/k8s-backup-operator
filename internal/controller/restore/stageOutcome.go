package restore

import (
	"context"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"

	k8sv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
)

// providerObservationRecoveryDelay is the delay a stage uses while it waits for the provider to
// decide. The owned child's events are the observation path, so this timer only ever matters when a
// watch event was lost.
const providerObservationRecoveryDelay = 5 * time.Minute

type stageAction int

const (
	// actionNext continues with the following stage.
	actionNext stageAction = iota
	// actionRetry ends this reconciliation and asks for another one.
	actionRetry
	// actionAbort ends this reconciliation without asking for another one, either because the
	// restore is terminal or because further progress depends on an event.
	actionAbort
)

type stageOutcome struct {
	action       stageAction
	requeueAfter time.Duration
	err          error
}

func next() stageOutcome {
	return stageOutcome{action: actionNext}
}

// retry ends the reconciliation and asks for another one at the controller's configured requeue time.
func retry() stageOutcome {
	return stageOutcome{action: actionRetry}
}

func retryAfter(delay time.Duration) stageOutcome {
	return stageOutcome{action: actionRetry, requeueAfter: delay}
}

// retryOnError ends the reconciliation with an error and leaves the delay to the controller-runtime
// backoff. Use it for transient failures. A nil error would silently stop the workflow, so it is
// treated as an immediate controlled retry instead.
func retryOnError(err error) stageOutcome {
	if err == nil {
		return retry()
	}

	return stageOutcome{action: actionRetry, err: err}
}

// abort ends the reconciliation without requeueing.
func abort() stageOutcome {
	return stageOutcome{action: actionAbort}
}

// result translates the outcome into the reconciler return values. This is the single place that
// decides how outcomes reach controller-runtime: an error is never combined with an explicit
// requeue, because controller-runtime would ignore the requeue and log the combination.
func (o stageOutcome) result(defaultRequeueDelay time.Duration) (ctrl.Result, error) {
	if o.err != nil {
		return ctrl.Result{}, o.err
	}

	if o.action == actionRetry {
		delay := o.requeueAfter
		if delay <= 0 {
			delay = defaultRequeueDelay
		}
		return ctrl.Result{RequeueAfter: delay}, nil
	}

	return ctrl.Result{}, nil
}

// reconcileStage is one ordered step of the Restore workflow. A stage that persisted the Restore
// returns the persisted object so the following stages see it; a stage that did not may return the
// Restore it received, or nil for the same effect.
type reconcileStage func(ctx context.Context, restore *k8sv1.Restore) (*k8sv1.Restore, stageOutcome)

// runStages executes the stages in order until one of them does not report next, and returns that
// stage's result. Each stage sees the Restore as the preceding stages left it. Running out of stages
// ends the reconciliation without a requeue.
func runStages(ctx context.Context, restore *k8sv1.Restore, defaultRequeueDelay time.Duration, stages ...reconcileStage) (ctrl.Result, error) {
	for _, stage := range stages {
		updated, outcome := stage(ctx, restore)
		if updated != nil {
			restore = updated
		}

		if outcome.action != actionNext {
			return outcome.result(defaultRequeueDelay)
		}
	}

	return abort().result(defaultRequeueDelay)
}
