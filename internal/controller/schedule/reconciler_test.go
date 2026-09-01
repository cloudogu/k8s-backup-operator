package schedule

import (
	"context"
	"errors"
	"testing"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
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

func (v *fakeValidator) validate(*backupv1.BackupSchedule) error {
	v.validatorCalled = true
	return v.err
}

type fakeCronJobManager struct {
	ensureCalled bool
	deleteCalled bool
	ensureErr    error
	deleteErr    error
}

func (m *fakeCronJobManager) ensure(ctx context.Context, s *backupv1.BackupSchedule) error {
	m.ensureCalled = true
	return m.ensureErr
}

func (m *fakeCronJobManager) delete(ctx context.Context, s *backupv1.BackupSchedule) error {
	m.deleteCalled = true
	return m.deleteErr
}

type fakeMetaData struct {
	ensureCalled bool
	removeCalled bool
	ensureErr    error
	removeErr    error
	onRemove     func(*backupv1.BackupSchedule)
}

func (m *fakeMetaData) ensure(ctx context.Context, s *backupv1.BackupSchedule) error {
	m.ensureCalled = true
	return m.ensureErr
}

func (m *fakeMetaData) remove(ctx context.Context, s *backupv1.BackupSchedule) error {
	m.removeCalled = true
	if m.onRemove != nil {
		m.onRemove(s)
	}
	return m.removeErr
}

func newTestReconciler(client client.Client, validator validator, cronJobs cronJobManager, metaData metadataManager) *defaultReconciler {
	return &defaultReconciler{
		client:     client,
		recorder:   newFakeEventRecorder(),
		validator:  validator,
		cronJobs:   cronJobs,
		conditions: defaultConditionManager{},
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

		expectedAccepted        metav1.ConditionStatus
		expectAcceptedCondition bool
		expectedReady           metav1.ConditionStatus
		expectedReadyReason     string
		expectedAcceptedReason  string
	}{
		{
			name: "success",

			expectValidatorCalled:      true,
			expectCronJobsEnsureCalled: true,

			expectedAccepted:        metav1.ConditionTrue,
			expectAcceptedCondition: true,
			expectedReady:           metav1.ConditionTrue,
			expectedReadyReason:     backupv1.ReasonReady,
		},
		{
			name: "validator error",

			validatorErr: errors.New("invalid"),

			expectValidatorCalled: true,

			expectedAccepted:        metav1.ConditionFalse,
			expectAcceptedCondition: true,
			expectedReady:           metav1.ConditionFalse,
			expectedReadyReason:     backupv1.ReasonInvalidSpec,
		},
		{
			name: "metadata error",

			metadataErr: errors.New("metadata"),

			expectError: true,

			expectedAccepted:        metav1.ConditionUnknown,
			expectAcceptedCondition: true,
			expectedReady:           metav1.ConditionFalse,
			expectedReadyReason:     backupv1.ReasonNotEvaluated,
			expectedAcceptedReason:  backupv1.ReasonNotEvaluated,
		},
		{
			name: "cronjob error",

			cronJobErr: errors.New("cronjob"),

			expectError: true,

			expectValidatorCalled:      true,
			expectCronJobsEnsureCalled: true,

			expectedAccepted:        metav1.ConditionTrue,
			expectAcceptedCondition: true,
			expectedReady:           metav1.ConditionFalse,
			expectedReadyReason:     backupv1.ReasonSyncFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testValidator := &fakeValidator{err: tt.validatorErr}
			testCronJobs := &fakeCronJobManager{ensureErr: tt.cronJobErr}
			testMetaData := &fakeMetaData{ensureErr: tt.metadataErr}
			recorder := newFakeEventRecorder()
			schedule := &backupv1.BackupSchedule{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test",
					Namespace:  "default",
					Finalizers: []string{backupv1.BackupScheduleFinalizer},
				},
			}
			reconciler := newTestReconciler(nil, testValidator, testCronJobs, testMetaData)
			reconciler.recorder = recorder

			err := reconciler.reconcileNormal(testCtx, schedule)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.expectValidatorCalled, testValidator.validatorCalled)
			assert.Equal(t, tt.expectCronJobsEnsureCalled, testCronJobs.ensureCalled)
			assert.Equal(t, tt.expectDeleteCalled, testCronJobs.deleteCalled)

			accepted := meta.FindStatusCondition(schedule.Status.Conditions, backupv1.ConditionAccepted)
			ready := meta.FindStatusCondition(schedule.Status.Conditions, backupv1.ConditionReady)

			if tt.expectAcceptedCondition {
				require.NotNil(t, accepted)
				assert.Equal(t, tt.expectedAccepted, accepted.Status)
				if tt.expectedAcceptedReason != "" {
					assert.Equal(t, tt.expectedAcceptedReason, accepted.Reason)
				}
			} else {
				assert.Nil(t, accepted)
			}
			assert.Equal(t, tt.expectedReady, ready.Status)
			assert.Equal(t, tt.expectedReadyReason, ready.Reason)
			assert.Len(t, schedule.Status.Conditions, 2)

			if tt.validatorErr != nil {
				requireRecordedEvent(t, recorder, schedule, corev1.EventTypeWarning, backupv1.InvalidScheduleEventReason, actionValidateBackupSchedule, "BackupSchedule has an invalid schedule: invalid")
			} else {
				requireNoRecordedEvent(t, recorder)
			}
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
			recorder := newFakeEventRecorder()
			schedule := &backupv1.BackupSchedule{}
			reconciler := newTestReconciler(nil, nil, testCronJobs, testMetadata)
			reconciler.recorder = recorder

			err := reconciler.reconcileDelete(testCtx, schedule)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tt.expectDeleteCalled, testCronJobs.deleteCalled)
			assert.Equal(t, tt.expectMetadataRemove, testMetadata.removeCalled)
			if tt.metadataErr != nil {
				requireRecordedEventContains(t, recorder, schedule, corev1.EventTypeWarning, backupv1.FinalizerRemovalFailedEventReason, actionDeleteBackupSchedule, "Failed to remove finalizer", backupv1.BackupScheduleFinalizer, tt.metadataErr.Error())
			} else {
				requireNoRecordedEvent(t, recorder)
			}

		})
	}
}

func Test_patchStatus_doesNotPatchUnchangedStatus(t *testing.T) {
	schedule := &backupv1.BackupSchedule{}
	reconciler := &defaultReconciler{}

	err := reconciler.patchStatus(testCtx, schedule.DeepCopy(), schedule.DeepCopy())

	require.NoError(t, err)
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

			if tt.deletionTimestamp != nil {
				testDeletion(t, metadata, fakeClient)
			}

			_, err := reconciler.reconcile(testCtx, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(schedule)})

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

func testDeletion(t *testing.T, metadata *fakeMetaData, fakeClient client.Client) {
	metadata.onRemove = func(schedule *backupv1.BackupSchedule) {
		stored := &backupv1.BackupSchedule{}
		require.NoError(t, fakeClient.Get(testCtx, client.ObjectKeyFromObject(schedule), stored))

		ready := meta.FindStatusCondition(stored.Status.Conditions, backupv1.ConditionReady)
		require.NotNil(t, ready)
		assert.Equal(t, metav1.ConditionFalse, ready.Status)
		assert.Equal(t, backupv1.ReasonDeleting, ready.Reason)
		assert.Len(t, stored.Status.Conditions, 1)
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
