package schedule

import (
	"context"
	"testing"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestMetadataManagerEnsure(t *testing.T) {
	tests := []struct {
		name               string
		initialFinalizers  []string
		initialLabels      map[string]string
		expectedFinalizers []string
		expectedLabels     map[string]string
	}{
		{
			name: "adds required metadata",
			expectedFinalizers: []string{
				backupv1.BackupScheduleFinalizer,
			},
			expectedLabels: map[string]string{
				LabelApp:    LabelValueApp,
				LabelPartOf: LabelValuePartOf,
			},
		},
		{
			name: "corrects outdated labels and preserves custom metadata",
			initialFinalizers: []string{
				"custom-finalizer",
				backupv1.BackupScheduleFinalizer,
			},
			initialLabels: map[string]string{
				LabelApp:    "outdated",
				LabelPartOf: "outdated",
				"custom":    "preserved",
			},
			expectedFinalizers: []string{
				"custom-finalizer",
				backupv1.BackupScheduleFinalizer,
			},
			expectedLabels: map[string]string{
				LabelApp:    LabelValueApp,
				LabelPartOf: LabelValuePartOf,
				"custom":    "preserved",
			},
		},
	}

	scheme := newMetadataManagerTestScheme(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schedule := newMetadataManagerTestSchedule()
			schedule.Finalizers = tt.initialFinalizers
			schedule.Labels = tt.initialLabels
			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(schedule).
				Build()
			manager := defaultMetadataManager{client: fakeClient}

			require.NoError(t, manager.ensure(t.Context(), schedule))

			stored := &backupv1.BackupSchedule{}
			require.NoError(t, fakeClient.Get(t.Context(), client.ObjectKeyFromObject(schedule), stored))
			assert.ElementsMatch(t, tt.expectedFinalizers, stored.Finalizers)
			assert.Equal(t, tt.expectedLabels, stored.Labels)
		})
	}

	t.Run("does not patch already synced metadata", func(t *testing.T) {
		schedule := newMetadataManagerTestSchedule()
		schedule.Finalizers = []string{backupv1.BackupScheduleFinalizer}
		schedule.Labels = map[string]string{
			LabelApp:    LabelValueApp,
			LabelPartOf: LabelValuePartOf,
		}
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

		require.NoError(t, manager.ensure(t.Context(), schedule))
		assert.False(t, patchCalled)
	})

	t.Run("returns patch errors", func(t *testing.T) {
		schedule := newMetadataManagerTestSchedule()
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

		err := manager.ensure(t.Context(), schedule)

		require.ErrorIs(t, err, assert.AnError)
	})
}

func newMetadataManagerTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()

	scheme := runtime.NewScheme()
	require.NoError(t, backupv1.AddToScheme(scheme))
	return scheme
}

func newMetadataManagerTestSchedule() *backupv1.BackupSchedule {
	return &backupv1.BackupSchedule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
	}
}
