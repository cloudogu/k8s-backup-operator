package scheduledbackup

import (
	"context"
	"testing"
	"time"

	v1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

const testNamespace = "my-namespace"

var testCtx = context.Background()

func TestNewManager(t *testing.T) {
	// given
	// when
	manager := NewManager(nil, Options{})

	// then
	assert.NotEmpty(t, manager)
}

func TestDefaultManager_ScheduleBackup(t *testing.T) {
	t.Run("should fail to apply backup", func(t *testing.T) {
		// given
		expectedBackupName := "banana-2023-11-30t10.30.00"
		expectedBackup := &v1.Backup{
			Name:      expectedBackupName,
			Namespace: testNamespace,
			Labels: map[string]string{
				"app":                          "ces",
				"k8s.cloudogu.com/part-of":     "backup",
				"app.kubernetes.io/name":       "backup",
				"app.kubernetes.io/part-of":    "k8s-backup-operator",
				"app.kubernetes.io/created-by": "k8s-backup-operator",
			},
			Spec: v1.BackupSpec{
				Provider: v1.Provider("velero"),
			},
		}

		clockMock := newMockTimeProvider(t)
		clockMock.EXPECT().Now().Return(time.Date(2023, time.November, 30, 10, 30, 00, 00, time.Local))

		k8sClient := fake.NewClientBuilder().WithInterceptorFuncs(interceptor.Funcs{Create: func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
			assert.Equal(t, expectedBackup, obj)
			return assert.AnError
		}}).Build()

		options := Options{Name: "banana", Namespace: testNamespace, Provider: "velero"}

		sut := &DefaultManager{client: k8sClient, options: options, clock: clockMock}

		// when
		err := sut.ScheduleBackup(testCtx)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to apply backup \"banana-2023-11-30t10.30.00\"")
	})
	t.Run("should succeed to apply backup", func(t *testing.T) {
		// given
		expectedBackupName := "banana-2023-11-30t10.30.00"
		expectedBackup := &v1.Backup{
			Name:      expectedBackupName,
			Namespace: testNamespace,
			Labels: map[string]string{
				"app":                          "ces",
				"k8s.cloudogu.com/part-of":     "backup",
				"app.kubernetes.io/name":       "backup",
				"app.kubernetes.io/part-of":    "k8s-backup-operator",
				"app.kubernetes.io/created-by": "k8s-backup-operator",
			},
			Spec: v1.BackupSpec{
				Provider: v1.Provider("velero"),
			},
		}

		clockMock := newMockTimeProvider(t)
		clockMock.EXPECT().Now().Return(time.Date(2023, time.November, 30, 10, 30, 00, 00, time.Local))

		k8sClient := fake.NewClientBuilder().
			WithScheme(testScheme(t)).
			WithInterceptorFuncs(interceptor.Funcs{Create: func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
				assert.Equal(t, expectedBackup, obj)
				return client.Create(ctx, obj, opts...)
			}}).Build()

		options := Options{Name: "banana", Namespace: testNamespace, Provider: "velero"}

		sut := &DefaultManager{client: k8sClient, options: options, clock: clockMock}

		// when
		err := sut.ScheduleBackup(testCtx)

		// then
		require.NoError(t, err)
	})
}

func testScheme(t *testing.T) *runtime.Scheme {
	scheme := runtime.NewScheme()
	builder := runtime.NewSchemeBuilder(v1.AddToScheme)
	err := builder.AddToScheme(scheme)
	assert.NoError(t, err)
	return scheme
}
