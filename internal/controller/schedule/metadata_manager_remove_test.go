package schedule

import (
	"context"
	"testing"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestMetadataManagerRemove(t *testing.T) {
	scheme := newMetadataManagerTestScheme(t)

	t.Run("removes finalizer and preserves custom metadata", func(t *testing.T) {
		schedule := newMetadataManagerTestSchedule()
		schedule.Finalizers = []string{
			"custom-finalizer",
			backupv1.BackupScheduleFinalizer,
		}
		schedule.Labels = map[string]string{"custom": "preserved"}
		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(schedule).
			Build()
		manager := defaultMetadataManager{client: fakeClient}

		require.NoError(t, manager.remove(t.Context(), schedule))

		stored := &backupv1.BackupSchedule{}
		require.NoError(t, fakeClient.Get(t.Context(), client.ObjectKeyFromObject(schedule), stored))
		assert.Equal(t, []string{"custom-finalizer"}, stored.Finalizers)
		assert.Equal(t, map[string]string{"custom": "preserved"}, stored.Labels)
	})

	t.Run("does not patch when its finalizer is absent", func(t *testing.T) {
		schedule := newMetadataManagerTestSchedule()
		schedule.Finalizers = []string{"custom-finalizer"}
		patchCalled := false
		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(schedule).
			WithInterceptorFuncs(interceptor.Funcs{
				Patch: func(_ context.Context, _ client.WithWatch, _ client.Object, _ client.Patch, _ ...client.PatchOption) error {
					patchCalled = true
					return assert.AnError
				},
			}).
			Build()
		manager := defaultMetadataManager{client: fakeClient}

		require.NoError(t, manager.remove(t.Context(), schedule))
		assert.False(t, patchCalled)
	})

	t.Run("returns errors from failed patching", func(t *testing.T) {
		schedule := newMetadataManagerTestSchedule()
		schedule.Finalizers = []string{backupv1.BackupScheduleFinalizer}
		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(schedule).
			WithInterceptorFuncs(interceptor.Funcs{
				Patch: func(_ context.Context, _ client.WithWatch, _ client.Object, _ client.Patch, _ ...client.PatchOption) error {
					return assert.AnError
				},
			}).
			Build()
		manager := defaultMetadataManager{client: fakeClient}

		err := manager.remove(t.Context(), schedule)

		require.ErrorIs(t, err, assert.AnError)
	})
}
