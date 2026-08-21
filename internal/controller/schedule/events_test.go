package schedule

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
)

// helper class for testing that events were logged in other tests
// construct custom event struct so object, type, reason, and message can all be checked

type recordedEvent struct {
	object    runtime.Object
	eventType string
	reason    string
	message   string
}

type fakeEventRecorder struct {
	events []recordedEvent
}

func newFakeEventRecorder() *fakeEventRecorder {
	return &fakeEventRecorder{}
}

// implement the record.EventRecorder interface
func (r *fakeEventRecorder) Event(object runtime.Object, eventType, reason, message string) {
	r.events = append(r.events, recordedEvent{object: object, eventType: eventType, reason: reason, message: message})
}

func (r *fakeEventRecorder) Eventf(object runtime.Object, eventType, reason, messageFmt string, args ...interface{}) {
	r.Event(object, eventType, reason, fmt.Sprintf(messageFmt, args...))
}

func (r *fakeEventRecorder) AnnotatedEventf(object runtime.Object, _ map[string]string, eventType, reason, messageFmt string, args ...interface{}) {
	r.Eventf(object, eventType, reason, messageFmt, args...)
}

func requireRecordedEvent(t *testing.T, recorder *fakeEventRecorder, expectedObject runtime.Object, expectedType, expectedReason, expectedMessage string) {
	if len(recorder.events) == 0 {
		require.Fail(t, "expected a Kubernetes Event, but none was recorded", "expected reason %q and message %q", expectedReason, expectedMessage)
		return
	}

	event := recorder.events[0]
	recorder.events = recorder.events[1:]
	require.Same(t, expectedObject, event.object)
	require.Equal(t, expectedType, event.eventType)
	require.Equal(t, expectedReason, event.reason)
	require.Equal(t, expectedMessage, event.message)
}

func requireRecordedEventContains(t *testing.T, recorder *fakeEventRecorder, expectedObject runtime.Object, expectedType, expectedReason string, expectedMessageParts ...string) {
	if len(recorder.events) == 0 {
		require.Fail(t, "expected a Kubernetes Event, but none was recorded")
		return
	}

	event := recorder.events[0]
	recorder.events = recorder.events[1:]
	require.Same(t, expectedObject, event.object)
	require.Equal(t, expectedType, event.eventType)
	require.Equal(t, expectedReason, event.reason)
	for _, expectedPart := range expectedMessageParts {
		if !strings.Contains(event.message, expectedPart) {
			require.Failf(t, "recorded event message does not contain expected text", "message %q does not contain %q", event.message, expectedPart)
		}
	}
}

func requireNoRecordedEvent(t *testing.T, recorder *fakeEventRecorder) {
	if len(recorder.events) > 0 {
		event := recorder.events[0]
		require.Fail(t, "unexpected Kubernetes Event", "reason: %q, message: %q", event.reason, event.message)
	}
}
