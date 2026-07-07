package config

import "context"

type Gateway interface {
	RetryLimit(context.Context) (int, error)
}
