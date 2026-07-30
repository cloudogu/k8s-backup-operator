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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var testCtx = context.TODO()

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
	ensureErr    error
	deleteErr    error
}

func (m *fakeCronJobManager) Ensure(ctx context.Context, s *backupv1.BackupSchedule) error {
	m.ensureCalled = true
	return m.ensureErr
}

func (m *fakeCronJobManager) Delete(ctx context.Context, s *backupv1.BackupSchedule) error {
	m.deleteCalled = true
	return m.deleteErr
}

type fakeMetaData struct {
	ensureCalled bool
	removeCalled bool
	ensureErr    error
	removeErr    error
}

func (m *fakeMetaData) Ensure(ctx context.Context, s *backupv1.BackupSchedule) error {
	m.ensureCalled = true
	return m.ensureErr
}

func (m *fakeMetaData) Remove(ctx context.Context, s *backupv1.BackupSchedule) error {
	m.removeCalled = true
	return m.removeErr
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testValidator := &fakeValidator{err: tt.validatorErr}
			testCronJobs := &fakeCronJobManager{ensureErr: tt.cronJobErr}
			testMetaData := &fakeMetaData{ensureErr: tt.metadataErr}
			schedule := &backupv1.BackupSchedule{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test",
					Namespace:  "default",
					Finalizers: []string{backupv1.BackupScheduleFinalizer},
				},
			}
			reconciler := newTestReconciler(nil, testValidator, testCronJobs, testMetaData)

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
			testCronJobs := &fakeCronJobManager{deleteErr: tt.cronJobErr}
			testMetadata := &fakeMetaData{removeErr: tt.metadataErr}
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

func Test_mainRecocileLoop(t *testing.T) {
	deletionTimestamp := metav1.Now()
	tests := []struct {
		name string

		deletionTimestamp *metav1.Time
		metadataErr       error
		cronJobErr        error

		expectError          bool
		expectMetadataEnsure bool
		expectMetadataRemove bool
		expectValidation     bool
		expectCronJobEnsure  bool
		expectCronJobDelete  bool
	}{
		{
			name:                 "reconciles normally",
			expectMetadataEnsure: true,
			expectValidation:     true,
			expectCronJobEnsure:  true,
		},
		{
			name:                 "reconciles deletion",
			deletionTimestamp:    &deletionTimestamp,
			expectMetadataRemove: true,
			expectCronJobDelete:  true,
		},
		{
			name:                 "returns normal reconciliation error",
			metadataErr:          errors.New("metadata ensure failed"),
			expectError:          true,
			expectMetadataEnsure: true,
		},
		{
			name:                "returns deletion reconciliation error",
			deletionTimestamp:   &deletionTimestamp,
			cronJobErr:          errors.New("cronjob delete failed"),
			expectError:         true,
			expectCronJobDelete: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schedule := newReconcileTestSchedule(tt.deletionTimestamp)
			fakeClient := newFakeScheduleClient(t, schedule)
			validator := &fakeValidator{}
			cronJobs := &fakeCronJobManager{deleteErr: tt.cronJobErr}
			metadata := &fakeMetaData{ensureErr: tt.metadataErr}
			reconciler := newTestReconciler(fakeClient, validator, cronJobs, metadata)

			_, err := reconciler.Reconcile(testCtx, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(schedule)})

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.expectMetadataEnsure, metadata.ensureCalled)
			assert.Equal(t, tt.expectMetadataRemove, metadata.removeCalled)
			assert.Equal(t, tt.expectValidation, validator.validatorCalled)
			assert.Equal(t, tt.expectCronJobEnsure, cronJobs.ensureCalled)
			assert.Equal(t, tt.expectCronJobDelete, cronJobs.deleteCalled)
		})
	}
}

func newReconcileTestSchedule(deletionTimestamp *metav1.Time) *backupv1.BackupSchedule {
	return &backupv1.BackupSchedule{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test",
			Namespace:         "default",
			Finalizers:        []string{backupv1.BackupScheduleFinalizer},
			DeletionTimestamp: deletionTimestamp,
		},
		Spec: backupv1.BackupScheduleSpec{Schedule: "0 2 * * *"},
	}
}

func newFakeScheduleClient(t *testing.T, schedule *backupv1.BackupSchedule) client.Client {
	t.Helper()

	scheme := runtime.NewScheme()
	require.NoError(t, backupv1.AddToScheme(scheme))

	return fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&backupv1.BackupSchedule{}).
		WithObjects(schedule).
		Build()
}
