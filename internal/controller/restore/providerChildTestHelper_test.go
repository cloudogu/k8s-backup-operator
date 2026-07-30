package restore

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	k8sv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
)

const (
	testRestoreUID = types.UID("11111111-1111-1111-1111-111111111111")
	testBackup     = "test-backup"
)

// childWriteCounter counts every mutating call the workflow makes through the fake client, so
// "performs no write" is asserted against the client rather than against an expectation on one verb.
type childWriteCounter struct {
	creates int
	updates int
	patches int
	deletes int
}

func (c *childWriteCounter) total() int {
	return c.creates + c.updates + c.patches + c.deletes
}

func (c *childWriteCounter) interceptor() interceptor.Funcs {
	return interceptor.Funcs{
		Create: func(ctx context.Context, wrapped client.WithWatch, object client.Object, opts ...client.CreateOption) error {
			c.creates++

			return wrapped.Create(ctx, object, opts...)
		},
		Update: func(ctx context.Context, wrapped client.WithWatch, object client.Object, opts ...client.UpdateOption) error {
			c.updates++

			return wrapped.Update(ctx, object, opts...)
		},
		Patch: func(ctx context.Context, wrapped client.WithWatch, object client.Object, patch client.Patch, opts ...client.PatchOption) error {
			c.patches++

			return wrapped.Patch(ctx, object, patch, opts...)
		},
		Delete: func(ctx context.Context, wrapped client.WithWatch, object client.Object, opts ...client.DeleteOption) error {
			c.deletes++

			return wrapped.Delete(ctx, object, opts...)
		},
	}
}

func newChildTestClient(t *testing.T, writes *childWriteCounter, objects ...client.Object) client.WithWatch {
	t.Helper()

	testScheme := runtime.NewScheme()
	require.NoError(t, k8sv1.AddToScheme(testScheme))
	require.NoError(t, velerov1.AddToScheme(testScheme))

	return fake.NewClientBuilder().
		WithScheme(testScheme).
		WithObjects(objects...).
		WithInterceptorFuncs(writes.interceptor()).
		Build()
}

func newParentRestore() *k8sv1.Restore {
	return &k8sv1.Restore{
		ObjectMeta: metav1.ObjectMeta{Name: testRestore, Namespace: testNamespace, UID: testRestoreUID},
		Spec:       k8sv1.RestoreSpec{BackupName: testBackup, Provider: k8sv1.ProviderVelero},
	}
}
