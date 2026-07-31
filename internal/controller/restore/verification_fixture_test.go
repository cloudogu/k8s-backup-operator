package restore

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

// This file holds the multi-reconcile fixture for testing the idempotency of the reconciler: one
// fake API server state that survives many Reconcile calls and a controller restart, plus a record
// of every write the reconciler performed.
type recordedClientAction struct {
	Verb string
	Type reflect.Type
	Key  types.NamespacedName
}

type clientActionRecorder struct {
	mutex   sync.Mutex
	actions []recordedClientAction
	paused  bool
}

func (r *clientActionRecorder) record(verb string, object client.Object) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.paused {
		return
	}

	r.actions = append(r.actions, recordedClientAction{
		Verb: verb,
		Type: reflect.TypeOf(object),
		Key:  client.ObjectKeyFromObject(object),
	})
}

func (r *clientActionRecorder) snapshot() []recordedClientAction {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	return append([]recordedClientAction(nil), r.actions...)
}

type reconcileFunction func(context.Context, ctrl.Request) (ctrl.Result, error)

type reconcileFactory func(client.WithWatch) reconcileFunction

// multiReconcileFixture keeps one fake API server state across reconciliations.
// Replacing the reconcile function simulates a controller process restart while
// retaining all objects and resource versions.
type multiReconcileFixture struct {
	client        client.WithWatch
	clientActions *clientActionRecorder
	reconcile     reconcileFunction
}

// newMultiReconcileFixture stores the given objects in a fake API server and builds the reconcile
// function under test. The additional interceptors run after the action recording, so a test can
// inject failures without losing the record of the attempts.
func newMultiReconcileFixture(
	t *testing.T,
	funcs interceptor.Funcs,
	factory reconcileFactory,
	objects ...client.Object,
) *multiReconcileFixture {
	t.Helper()

	clientActions := &clientActionRecorder{}
	fakeClient := newTestClient(t, recordClientActions(clientActions, funcs), objects...)

	return &multiReconcileFixture{
		client:        fakeClient,
		clientActions: clientActions,
		reconcile:     factory(fakeClient),
	}
}

// simulateExternalWrite performs a write on behalf of somebody else — the provider moving its restore
// to the next phase, for instance — without recording it, so that a snapshot only ever contains what
// the reconciler itself did.
func (f *multiReconcileFixture) simulateExternalWrite(t *testing.T, write func(client.WithWatch) error) {
	t.Helper()

	f.clientActions.mutex.Lock()
	f.clientActions.paused = true
	f.clientActions.mutex.Unlock()

	err := write(f.client)

	f.clientActions.mutex.Lock()
	f.clientActions.paused = false
	f.clientActions.mutex.Unlock()

	require.NoError(t, err)
}

func (f *multiReconcileFixture) restart(factory reconcileFactory) {
	f.reconcile = factory(f.client)
}

func (f *multiReconcileFixture) reconcileTimes(
	ctx context.Context,
	request ctrl.Request,
	count int,
) ([]ctrl.Result, []error) {
	results := make([]ctrl.Result, 0, count)
	errors := make([]error, 0, count)
	for range count {
		result, err := f.reconcile(ctx, request)
		results = append(results, result)
		errors = append(errors, err)
	}

	return results, errors
}

func newRestoreRequest(name string) ctrl.Request {
	return ctrl.Request{NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: name}}
}

// recordClientActions records every mutating call and then hands it to next, or to the fake client
// when next does not intercept that verb. Chaining exists so that the recording is never lost when a
// test also has to inject a failure
func recordClientActions(recorder *clientActionRecorder, next interceptor.Funcs) interceptor.Funcs {
	return interceptor.Funcs{
		// Reads are not recorded, but they are still handed on, so that a test can fail one.
		Get: func(ctx context.Context, wrapped client.WithWatch, key client.ObjectKey, object client.Object, opts ...client.GetOption) error {
			if next.Get != nil {
				return next.Get(ctx, wrapped, key, object, opts...)
			}

			return wrapped.Get(ctx, key, object, opts...)
		},
		List: func(ctx context.Context, wrapped client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
			if next.List != nil {
				return next.List(ctx, wrapped, list, opts...)
			}

			return wrapped.List(ctx, list, opts...)
		},
		Create: func(ctx context.Context, wrapped client.WithWatch, object client.Object, opts ...client.CreateOption) error {
			recorder.record("create", object)
			if next.Create != nil {
				return next.Create(ctx, wrapped, object, opts...)
			}

			return wrapped.Create(ctx, object, opts...)
		},
		Update: func(ctx context.Context, wrapped client.WithWatch, object client.Object, opts ...client.UpdateOption) error {
			recorder.record("update", object)
			if next.Update != nil {
				return next.Update(ctx, wrapped, object, opts...)
			}

			return wrapped.Update(ctx, object, opts...)
		},
		Patch: func(ctx context.Context, wrapped client.WithWatch, object client.Object, patch client.Patch, opts ...client.PatchOption) error {
			recorder.record("patch", object)
			if next.Patch != nil {
				return next.Patch(ctx, wrapped, object, patch, opts...)
			}

			return wrapped.Patch(ctx, object, patch, opts...)
		},
		Delete: func(ctx context.Context, wrapped client.WithWatch, object client.Object, opts ...client.DeleteOption) error {
			recorder.record("delete", object)
			if next.Delete != nil {
				return next.Delete(ctx, wrapped, object, opts...)
			}

			return wrapped.Delete(ctx, object, opts...)
		},
		SubResourceUpdate: func(ctx context.Context, wrapped client.Client, subResource string, object client.Object, opts ...client.SubResourceUpdateOption) error {
			recorder.record(fmt.Sprintf("%s-update", subResource), object)
			if next.SubResourceUpdate != nil {
				return next.SubResourceUpdate(ctx, wrapped, subResource, object, opts...)
			}

			return wrapped.SubResource(subResource).Update(ctx, object, opts...)
		},
		SubResourcePatch: func(ctx context.Context, wrapped client.Client, subResource string, object client.Object, patch client.Patch, opts ...client.SubResourcePatchOption) error {
			recorder.record(fmt.Sprintf("%s-patch", subResource), object)
			if next.SubResourcePatch != nil {
				return next.SubResourcePatch(ctx, wrapped, subResource, object, patch, opts...)
			}

			return wrapped.SubResource(subResource).Patch(ctx, object, patch, opts...)
		},
	}
}

// createOf is the action the creation of the given object records.
func createOf(object client.Object) recordedClientAction {
	return recordedClientAction{
		Verb: "create",
		Type: reflect.TypeOf(object),
		Key:  client.ObjectKeyFromObject(object),
	}
}

// updateOf is the action a write of the given object records.
func updateOf(object client.Object) recordedClientAction {
	return recordedClientAction{
		Verb: "update",
		Type: reflect.TypeOf(object),
		Key:  client.ObjectKeyFromObject(object),
	}
}

// statusUpdateOf is the action a status write of the given object records.
func statusUpdateOf(object client.Object) recordedClientAction {
	return recordedClientAction{
		Verb: "status-update",
		Type: reflect.TypeOf(object),
		Key:  client.ObjectKeyFromObject(object),
	}
}
