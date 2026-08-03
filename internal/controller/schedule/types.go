package schedule

const (
	AcceptedCondition      = "Accepted"
	CronJobSyncedCondition = "CronJobSynced"
	ReadyCondition         = "Ready"
	DeletingCondition      = "Deleting"
)

const (
	ReasonValidSpec    = "ValidSpec"
	ReasonInvalidSpec  = "InvalidSpec"
	ReasonSynced       = "Synced"
	ReasonSyncFailed   = "SyncFailed"
	ReasonDeleting     = "Deleting"
	ReasonNotEvaluated = "NotEvaluated"
	ReasonReady        = "Ready"
)
