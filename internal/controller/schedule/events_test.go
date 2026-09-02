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
	related   runtime.Object
	eventType string
	reason    string
	action    string
	message   string
}

type fakeEventRecorder struct {
	events []recordedEvent
}

func newFakeEventRecorder() *fakeEventRecorder {
	return &fakeEventRecorder{}
}

func (r *fakeEventRecorder) Eventf(regarding runtime.Object, related runtime.Object, eventType, reason, action, note string, args ...interface{}) {
	r.events = append(r.events, recordedEvent{
		object: regarding, related: related, eventType: eventType, reason: reason, action: action, message: fmt.Sprintf(note, args...),
	})
}

func requireRecordedEvent(t *testing.T, recorder *fakeEventRecorder, expectedObject, expectedRelated runtime.Object, expectedType, expectedReason, expectedAction, expectedMessage string) {
	if len(recorder.events) == 0 {
		require.Fail(t, "expected a Kubernetes Event, but none was recorded", "expected reason %q and message %q", expectedReason, expectedMessage)
		return
	}

	event := recorder.events[0]
	recorder.events = recorder.events[1:]
	require.Same(t, expectedObject, event.object)
	require.Equal(t, expectedRelated, event.related)
	require.Equal(t, expectedType, event.eventType)
	require.Equal(t, expectedReason, event.reason)
	require.Equal(t, expectedAction, event.action)
	require.Equal(t, expectedMessage, event.message)
}

func requireRecordedEventContains(t *testing.T, recorder *fakeEventRecorder, expectedObject, expectedRelated runtime.Object, expectedType, expectedReason, expectedAction string, expectedMessageParts ...string) {
	if len(recorder.events) == 0 {
		require.Fail(t, "expected a Kubernetes Event, but none was recorded")
		return
	}

	event := recorder.events[0]
	recorder.events = recorder.events[1:]
	require.Same(t, expectedObject, event.object)
	require.Equal(t, expectedRelated, event.related)
	require.Equal(t, expectedType, event.eventType)
	require.Equal(t, expectedReason, event.reason)
	require.Equal(t, expectedAction, event.action)
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
