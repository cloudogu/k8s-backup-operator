package backup

import (
	"context"
	"time"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type stageAction int

const (
	// actionNext continues with the following stage.
	actionNext stageAction = iota
	// actionRetry ends this reconciliation and asks for another one.
	actionRetry
	// actionAbort ends this reconciliation without asking for another one, either because the
	// backup is terminal or because further progress depends on an event.
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

// retry ends the reconciliation and asks for another one at the controller's requeue cadence
func retry() stageOutcome {
	return stageOutcome{action: actionRetry}
}

// retryAfter ends the reconciliation and asks for another one after the given delay.
// A delay that is not usable falls back to the controller's requeue delay in result.
func retryAfter(delay time.Duration) stageOutcome {
	return stageOutcome{action: actionRetry, requeueAfter: delay}
}

// retryOnError ends the reconciliation with an error and leaves the delay to the controller-runtime
// backoff. Use it for transient failures. A nil error would silently stop the workflow, so it is
// treated as a controlled retry instead.
func retryOnError(err error) stageOutcome {
	if err == nil {
		return retryAfter(0)
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

// stage is one ordered step of the Backup workflow. A stage that persisted the Backup returns the
// persisted object so the following stages see it; a stage that did not may return the Backup it
// received, or nil for the same effect.
type stage func(ctx context.Context, backup *backupv1.Backup) (*backupv1.Backup, stageOutcome)

// runStages executes the stages in order until one of them does not report next, and returns that
// stage's result. Each stage sees the Backup as the preceding stages left it. Running out of stages
// ends the reconciliation without a requeue.
func runStages(ctx context.Context, backup *backupv1.Backup, defaultRequeueDelay time.Duration, stages ...stage) (ctrl.Result, error) {
	for _, s := range stages {
		updated, outcome := s(ctx, backup)
		if updated != nil {
			backup = updated
		}

		if outcome.action != actionNext {
			return outcome.result(defaultRequeueDelay)
		}
	}

	return abort().result(defaultRequeueDelay)
}
