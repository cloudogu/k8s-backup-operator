package logging

import (
	"context"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func FromContext(ctx context.Context) logr.Logger {
	return log.FromContext(ctx)
}

func IntoContext(ctx context.Context, logger logr.Logger) context.Context {
	return log.IntoContext(ctx, logger)
}

func Info(ctx context.Context, message string, keysAndValues ...any) {
	FromContext(ctx).Info(message, keysAndValues...)
}

func Debug(ctx context.Context, message string, keysAndValues ...any) {
	FromContext(ctx).V(1).Info(message, keysAndValues...)
}

func Error(ctx context.Context, err error, message string, keysAndValues ...any) {
	FromContext(ctx).Error(err, message, keysAndValues...)
}
