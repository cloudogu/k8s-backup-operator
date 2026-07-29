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
)

func TestEnsureFinalizer(t *testing.T) {
	tests := []struct {
		name                   string
		initialFinalizer       []string
		testEnsure             bool
		changeExpected         bool
		expectedFinalizerAfter []string
	}{
		{
			name:             "ensure adds finalizer",
			initialFinalizer: nil,
			testEnsure:       true,
			changeExpected:   true,
			expectedFinalizerAfter: []string{
				backupv1.BackupScheduleFinalizer,
			},
		},
		{
			name: "ensure doesn't add existing finalizer",
			initialFinalizer: []string{
				backupv1.BackupScheduleFinalizer,
			},
			testEnsure:     true,
			changeExpected: false,
			expectedFinalizerAfter: []string{
				backupv1.BackupScheduleFinalizer,
			},
		},
		{
			name: "remove removes finalizer",
			initialFinalizer: []string{
				backupv1.BackupScheduleFinalizer,
			},
			testEnsure:             false,
			changeExpected:         true,
			expectedFinalizerAfter: nil,
		},
		{
			name:                   "remove doesn't try to remove non-existing finalizer",
			initialFinalizer:       nil,
			testEnsure:             false,
			changeExpected:         false,
			expectedFinalizerAfter: nil,
		},
	}
	scheme := runtime.NewScheme()
	require.NoError(t, backupv1.AddToScheme(scheme))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			schedule := &backupv1.BackupSchedule{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test",
					Namespace:  "default",
					Finalizers: tt.initialFinalizer,
				},
			}

			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(schedule).
				Build()

			reconciler := &defaultReconciler{
				client: fakeClient,
			}

			var (
				changed bool
				err     error
			)

			if tt.testEnsure {
				changed, err = reconciler.ensureFinalizerSet(context.Background(), schedule)
			} else {
				changed, err = reconciler.removeFinalizer(context.Background(), schedule)
			}

			require.NoError(t, err)
			assert.Equal(t, tt.changeExpected, changed)

			stored := &backupv1.BackupSchedule{}
			err = fakeClient.Get(
				context.Background(),
				client.ObjectKeyFromObject(schedule),
				stored,
			)
			require.NoError(t, err)

			assert.ElementsMatch(t, tt.expectedFinalizerAfter, stored.Finalizers)
		})
	}

}
