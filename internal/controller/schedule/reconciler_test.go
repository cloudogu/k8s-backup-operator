package schedule

import (
	"context"
	"errors"
	"testing"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var testCtx = context.TODO()

func TestEnsureFinalizer(t *testing.T) {
	tests := []struct {
		name                   string
		initialFinalizer       []string
		testEnsure             bool
		expectedFinalizerAfter []string
	}{
		{
			name:             "ensure adds finalizer",
			initialFinalizer: nil,
			testEnsure:       true,
			expectedFinalizerAfter: []string{
				backupv1.BackupScheduleFinalizer,
			},
		},
		{
			name: "ensure doesn't add existing finalizer",
			initialFinalizer: []string{
				backupv1.BackupScheduleFinalizer,
			},
			testEnsure: true,
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
			expectedFinalizerAfter: nil,
		},
		{
			name:                   "remove doesn't try to remove non-existing finalizer",
			initialFinalizer:       nil,
			testEnsure:             false,
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

			manager := metadataManager{client: fakeClient}

			var (
				err error
			)

			if tt.testEnsure {
				err = manager.Ensure(context.Background(), schedule)
			} else {
				err = manager.Remove(context.Background(), schedule)
			}

			require.NoError(t, err)

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

// ----------------------

type fakeValidator struct {
	validatorCalled bool
	err             error
}

func (v *fakeValidator) Validate(*backupv1.BackupSchedule) error {
	v.validatorCalled = true
	return v.err
}

type fakeCronJobManager struct {
	ensureCalled bool
	deleteCalled bool
	err          error
}

func (m *fakeCronJobManager) Ensure(ctx context.Context, s *backupv1.BackupSchedule) error {
	m.ensureCalled = true
	return m.err
}

func (m *fakeCronJobManager) Delete(ctx context.Context, s *backupv1.BackupSchedule) error {
	m.deleteCalled = true
	return m.err
}

type fakeMetaData struct {
	ensureCalled bool
	removeCalled bool
	err          error
}

func (m *fakeMetaData) Ensure(ctx context.Context, s *backupv1.BackupSchedule) error {
	m.ensureCalled = true
	return m.err
}

func (m *fakeMetaData) Remove(ctx context.Context, s *backupv1.BackupSchedule) error {
	m.removeCalled = true
	return m.err
}

func newTestReconciler(client client.Client, validator Validator, cronJobs CronJobManager, metaData MetadataManager) *defaultReconciler {
	return &defaultReconciler{
		client:     client,
		validator:  validator,
		cronJobs:   cronJobs,
		conditions: conditionManager{},
		metadata:   metaData,
	}
}

func Test_reconcileNormal(t *testing.T) {
	tests := []struct {
		name string

		validatorErr error
		metadataErr  error
		cronJobErr   error

		expectError bool

		expectValidatorCalled      bool
		expectCronJobsEnsureCalled bool
		expectDeleteCalled         bool

		expectedAccepted metav1.ConditionStatus
		expectedSynced   metav1.ConditionStatus
		expectedReady    metav1.ConditionStatus
	}{
		{
			name: "success",

			expectValidatorCalled:      true,
			expectCronJobsEnsureCalled: true,

			expectedAccepted: metav1.ConditionTrue,
			expectedSynced:   metav1.ConditionTrue,
			expectedReady:    metav1.ConditionTrue,
		},
		{
			name: "validator error",

			validatorErr: errors.New("invalid"),

			expectValidatorCalled: true,

			expectedAccepted: metav1.ConditionFalse,
			expectedSynced:   metav1.ConditionFalse,
			expectedReady:    metav1.ConditionFalse,
		},
		{
			name: "metadata error",

			metadataErr: errors.New("metadata"),

			expectError: true,

			expectedAccepted: metav1.ConditionFalse,
			expectedSynced:   metav1.ConditionFalse,
			expectedReady:    metav1.ConditionFalse,
		},
		{
			name: "cronjob error",

			cronJobErr: errors.New("cronjob"),

			expectError: true,

			expectValidatorCalled:      true,
			expectCronJobsEnsureCalled: true,

			expectedAccepted: metav1.ConditionTrue,
			expectedSynced:   metav1.ConditionFalse,
			expectedReady:    metav1.ConditionFalse,
		},
	}

	scheme := runtime.NewScheme()
	require.NoError(t, backupv1.AddToScheme(scheme))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			testValidator := &fakeValidator{
				err: tt.validatorErr,
			}

			testCronJobs := &fakeCronJobManager{
				err: tt.cronJobErr,
			}

			testMetaData := &fakeMetaData{
				err: tt.metadataErr,
			}

			schedule := &backupv1.BackupSchedule{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
					Finalizers: []string{
						backupv1.BackupScheduleFinalizer,
					},
				},
			}

			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(schedule).
				Build()

			reconciler := newTestReconciler(
				fakeClient,
				testValidator,
				testCronJobs,
				testMetaData,
			)

			err := reconciler.reconcileNormal(testCtx, schedule)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.expectValidatorCalled, testValidator.validatorCalled)
			assert.Equal(t, tt.expectCronJobsEnsureCalled, testCronJobs.ensureCalled)
			assert.Equal(t, tt.expectDeleteCalled, testCronJobs.deleteCalled)

			accepted := meta.FindStatusCondition(schedule.Status.Conditions, AcceptedCondition)
			synced := meta.FindStatusCondition(schedule.Status.Conditions, CronJobSyncedCondition)
			ready := meta.FindStatusCondition(schedule.Status.Conditions, ReadyCondition)

			assert.Equal(t, tt.expectedAccepted, accepted.Status)
			assert.Equal(t, tt.expectedSynced, synced.Status)
			assert.Equal(t, tt.expectedReady, ready.Status)
		})
	}

}

func Test_reconcileDelete(t *testing.T) {
	tests := []struct {
		name string

		cronJobErr  error
		metadataErr error

		expectError          bool
		expectDeleteCalled   bool
		expectMetadataRemove bool
	}{
		{
			name:                 "successful deletion",
			expectDeleteCalled:   true,
			expectMetadataRemove: true,
		},
		{
			name:               "error deleting cronjob",
			cronJobErr:         errors.New("deleting cronjob failed"),
			expectError:        true,
			expectDeleteCalled: true,
		},
		{
			name:                 "error removing finalizer",
			metadataErr:          errors.New("deleting metadata failed"),
			expectError:          true,
			expectDeleteCalled:   true,
			expectMetadataRemove: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testCronJobs := &fakeCronJobManager{err: tt.cronJobErr}
			testMetadata := &fakeMetaData{err: tt.metadataErr}
			schedule := &backupv1.BackupSchedule{}
			reconciler := newTestReconciler(nil, nil, testCronJobs, testMetadata)

			err := reconciler.reconcileDelete(testCtx, schedule)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tt.expectDeleteCalled, testCronJobs.deleteCalled)
			assert.Equal(t, tt.expectMetadataRemove, testMetadata.removeCalled)

			deleting := meta.FindStatusCondition(schedule.Status.Conditions, DeletingCondition)
			require.NotNil(t, deleting)
			assert.Equal(t, metav1.ConditionTrue, deleting.Status)
			assert.Equal(t, ReasonDeleting, deleting.Reason)
		})
	}
}
