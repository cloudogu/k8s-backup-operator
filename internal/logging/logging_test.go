package logging

import (
	"context"
	"errors"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
)

type recordingSink struct {
	infos  []string
	errors []error
}

func (*recordingSink) Init(logr.RuntimeInfo) {}

func (*recordingSink) Enabled(int) bool { return true }

func (sink *recordingSink) Info(_ int, message string, _ ...any) {
	sink.infos = append(sink.infos, message)
}

func (sink *recordingSink) Error(err error, _ string, _ ...any) {
	sink.errors = append(sink.errors, err)
}

func (sink *recordingSink) WithValues(...any) logr.LogSink { return sink }

func (sink *recordingSink) WithName(string) logr.LogSink { return sink }

func TestLoggingUsesLoggerFromContext(t *testing.T) {
	sink := &recordingSink{}
	ctx := IntoContext(context.Background(), logr.New(sink))
	testErr := errors.New("test error")

	Info(ctx, "info")
	Debug(ctx, "debug")
	Error(ctx, testErr, "error")

	require.Equal(t, []string{"info", "debug"}, sink.infos)
	require.Equal(t, []error{testErr}, sink.errors)
	require.Same(t, sink, FromContext(ctx).GetSink())
}
