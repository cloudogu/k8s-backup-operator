package restore

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	k8sv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
)

// objectWrites counts the mutating calls that targeted one object.
type objectWrites struct {
	creates       int
	updates       int
	patches       int
	deletes       int
	statusUpdates int
}

func (w objectWrites) total() int {
	return w.creates + w.updates + w.patches + w.deletes + w.statusUpdates
}

// clientWrites records every mutating call the fake client saw, split by the object it targeted. The
// split matters because the workflow writes the parent and the child through the same client.
type clientWrites struct {
	// parent counts writes to the Restore this controller owns.
	parent objectWrites
	// child counts writes to the provider child.
	child objectWrites
	// other counts writes to anything else, so a stray write cannot hide in a passing assertion.
	other objectWrites
}

func (w *clientWrites) total() int {
	return w.parent.total() + w.child.total() + w.other.total()
}

// forObject picks the counter the given object belongs to.
func (w *clientWrites) forObject(object client.Object) *objectWrites {
	switch object.(type) {
	case *k8sv1.Restore:
		return &w.parent
	case *velerov1.Restore:
		return &w.child
	default:
		return &w.other
	}
}

func (w *clientWrites) interceptor() interceptor.Funcs {
	return interceptor.Funcs{
		Create: func(ctx context.Context, wrapped client.WithWatch, object client.Object, opts ...client.CreateOption) error {
			w.forObject(object).creates++

			return wrapped.Create(ctx, object, opts...)
		},
		Update: func(ctx context.Context, wrapped client.WithWatch, object client.Object, opts ...client.UpdateOption) error {
			w.forObject(object).updates++

			return wrapped.Update(ctx, object, opts...)
		},
		Patch: func(ctx context.Context, wrapped client.WithWatch, object client.Object, patch client.Patch, opts ...client.PatchOption) error {
			w.forObject(object).patches++

			return wrapped.Patch(ctx, object, patch, opts...)
		},
		Delete: func(ctx context.Context, wrapped client.WithWatch, object client.Object, opts ...client.DeleteOption) error {
			w.forObject(object).deletes++

			return wrapped.Delete(ctx, object, opts...)
		},
		SubResourceUpdate: func(ctx context.Context, wrapped client.Client, subResource string, object client.Object, opts ...client.SubResourceUpdateOption) error {
			w.forObject(object).statusUpdates++

			return wrapped.SubResource(subResource).Update(ctx, object, opts...)
		},
	}
}

// newTestClient builds a fake client that knows the Restore and Velero types, with the given
// interceptors. Status subresources are enabled for Restore so that Status().Update behaves like the
// real client instead of writing the whole object. Pass interceptor.Funcs{} when the test does not
// care about the writes, and writes.interceptor() when it does.
func newTestClient(t *testing.T, funcs interceptor.Funcs, objects ...client.Object) client.WithWatch {
	t.Helper()

	testScheme := runtime.NewScheme()
	require.NoError(t, k8sv1.AddToScheme(testScheme))
	require.NoError(t, velerov1.AddToScheme(testScheme))

	return fake.NewClientBuilder().
		WithScheme(testScheme).
		WithObjects(objects...).
		WithStatusSubresource(&k8sv1.Restore{}).
		WithInterceptorFuncs(funcs).
		Build()
}

// newTestClientWithParent stores the parent Restore and syncs the resource version the client assigned
// back onto the caller's object. Without that sync the first status update would be rejected as stale,
// because the in-memory fixture never went through the API, and the retry would inflate every write
// count by one.
func newTestClientWithParent(t *testing.T, funcs interceptor.Funcs, parent *k8sv1.Restore, extra ...client.Object) client.WithWatch {
	t.Helper()

	testClient := newTestClient(t, funcs, append([]client.Object{parent}, extra...)...)

	stored := &k8sv1.Restore{}
	require.NoError(t, testClient.Get(context.Background(), client.ObjectKeyFromObject(parent), stored))
	parent.ResourceVersion = stored.ResourceVersion

	return testClient
}

// failingDelete makes every delete fail with the given error, to cover the non-NotFound error path of
// the child deletion.
func failingDelete(err error) interceptor.Funcs {
	return interceptor.Funcs{
		Delete: func(_ context.Context, _ client.WithWatch, _ client.Object, _ ...client.DeleteOption) error {
			return err
		},
	}
}

// failingStatusUpdate makes every status write fail with the given error, for the paths that have to
// report a status update they could not persist.
func failingStatusUpdate(err error) interceptor.Funcs {
	return failingStatusUpdateFrom(1, err)
}

// failingStatusUpdateFrom makes the nth status write and every later one fail with the given error.
// The offset is needed because the create flow writes the status more than once and the tests have to
// pick which of those writes fails.
func failingStatusUpdateFrom(nth int, err error) interceptor.Funcs {
	seen := 0

	return interceptor.Funcs{
		SubResourceUpdate: func(ctx context.Context, wrapped client.Client, subResource string, object client.Object, opts ...client.SubResourceUpdateOption) error {
			seen++
			if seen >= nth {
				return err
			}

			return wrapped.SubResource(subResource).Update(ctx, object, opts...)
		},
	}
}

// failingUpdate makes every update of the whole object fail with the given error, for the metadata
// paths. Status writes are left working, so a test can fail a finalizer or label write in isolation.
func failingUpdate(err error) interceptor.Funcs {
	return interceptor.Funcs{
		Update: func(_ context.Context, _ client.WithWatch, _ client.Object, _ ...client.UpdateOption) error {
			return err
		},
	}
}
