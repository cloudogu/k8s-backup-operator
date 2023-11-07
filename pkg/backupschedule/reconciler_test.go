package backupschedule

import (
	k8sv1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/config"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"testing"
	"time"
)

const testNamespace = "test-ns"
const testBackupSchedule = "test-backup-schedule"

func TestNewReconciler(t *testing.T) {
	// given

	// when
	actual := NewReconciler(nil, nil, testNamespace, nil)

	// then
	assert.NotEmpty(t, actual)
}

func Test_backupScheduleReconciler_SetupWithManager(t *testing.T) {
	t.Run("should fail", func(t *testing.T) {
		// given
		sut := &backupScheduleReconciler{}

		// when
		err := sut.SetupWithManager(nil)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "must provide a non-nil Manager")
	})
	t.Run("should succeed", func(t *testing.T) {
		// given
		ctrlManMock := newMockControllerManager(t)
		ctrlManMock.EXPECT().GetControllerOptions().Return(config.Controller{})
		ctrlManMock.EXPECT().GetScheme().Return(createScheme(t))
		logger := log.FromContext(testCtx)
		ctrlManMock.EXPECT().GetLogger().Return(logger)
		ctrlManMock.EXPECT().Add(mock.Anything).Return(nil)
		ctrlManMock.EXPECT().GetCache().Return(nil)

		sut := &backupScheduleReconciler{}

		// when
		err := sut.SetupWithManager(ctrlManMock)

		// then
		require.NoError(t, err)
	})
}

func createScheme(t *testing.T) *runtime.Scheme {
	t.Helper()

	scheme := runtime.NewScheme()
	gv, err := schema.ParseGroupVersion("k8s.cloudogu.com/v1")
	assert.NoError(t, err)

	scheme.AddKnownTypes(gv, &k8sv1.BackupSchedule{})
	return scheme
}

func Test_backupScheduleReconciler_Reconcile(t *testing.T) {
	originalMaxTries := maxTries
	defer func() { maxTries = originalMaxTries }()
	maxTries = 1

	now := metav1.NewTime(time.Now())
	t.Run("should ignore not found when getting backup schedule", func(t *testing.T) {
		// given
		backupScheduleClient := newMockEcosystemBackupScheduleInterface(t)
		notFound := errors.NewNotFound(schema.GroupResource{}, testBackupSchedule)
		backupScheduleClient.EXPECT().Get(testCtx, testBackupSchedule, metav1.GetOptions{}).Return(nil, notFound)
		ecosystemClient := newMockEcosystemV1Alpha1Interface(t)
		ecosystemClient.EXPECT().BackupSchedules(testNamespace).Return(backupScheduleClient)
		clientSet := newMockEcosystemInterface(t)
		clientSet.EXPECT().EcosystemV1Alpha1().Return(ecosystemClient)

		sut := &backupScheduleReconciler{
			clientSet: clientSet,
			namespace: testNamespace,
		}
		request := ctrl.Request{NamespacedName: types.NamespacedName{
			Namespace: testNamespace,
			Name:      testBackupSchedule,
		}}

		// when
		actual, err := sut.Reconcile(testCtx, request)

		// then
		require.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, actual)
	})
	t.Run("should fail to get backup schedule", func(t *testing.T) {
		// given
		backupScheduleClient := newMockEcosystemBackupScheduleInterface(t)
		backupScheduleClient.EXPECT().Get(testCtx, testBackupSchedule, metav1.GetOptions{}).Return(nil, assert.AnError)
		ecosystemClient := newMockEcosystemV1Alpha1Interface(t)
		ecosystemClient.EXPECT().BackupSchedules(testNamespace).Return(backupScheduleClient)
		clientSet := newMockEcosystemInterface(t)
		clientSet.EXPECT().EcosystemV1Alpha1().Return(ecosystemClient)

		sut := &backupScheduleReconciler{
			clientSet: clientSet,
			namespace: testNamespace,
		}
		request := ctrl.Request{NamespacedName: types.NamespacedName{
			Namespace: testNamespace,
			Name:      testBackupSchedule,
		}}

		// when
		actual, err := sut.Reconcile(testCtx, request)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.Equal(t, ctrl.Result{}, actual)
	})
	t.Run("should perform delete when deletion timestamp is set", func(t *testing.T) {
		// given
		backupSchedule := &k8sv1.BackupSchedule{
			ObjectMeta: metav1.ObjectMeta{
				Name:              testBackupSchedule,
				Namespace:         testNamespace,
				DeletionTimestamp: &now,
			},
			Spec: k8sv1.BackupScheduleSpec{
				Schedule: "0 0 * * *",
				Provider: "velero",
			},
		}

		backupScheduleClient := newMockEcosystemBackupScheduleInterface(t)
		backupScheduleClient.EXPECT().Get(testCtx, testBackupSchedule, metav1.GetOptions{}).Return(backupSchedule, nil)
		ecosystemClient := newMockEcosystemV1Alpha1Interface(t)
		ecosystemClient.EXPECT().BackupSchedules(testNamespace).Return(backupScheduleClient)
		clientSet := newMockEcosystemInterface(t)
		clientSet.EXPECT().EcosystemV1Alpha1().Return(ecosystemClient)

		manager := NewMockManager(t)
		manager.EXPECT().delete(testCtx, backupSchedule).Return(nil)
		recorder := newMockEventRecorder(t)
		recorder.EXPECT().Event(backupSchedule, "Normal", "Delete", "Delete successful")
		requeueHandler := newMockRequeueHandler(t)
		requeueHandler.EXPECT().Handle(testCtx, "Delete of backup schedule test-backup-schedule failed", backupSchedule, nil, "").Return(ctrl.Result{}, nil)

		sut := &backupScheduleReconciler{
			clientSet:      clientSet,
			namespace:      testNamespace,
			manager:        manager,
			recorder:       recorder,
			requeueHandler: requeueHandler,
		}
		request := ctrl.Request{NamespacedName: types.NamespacedName{
			Namespace: testNamespace,
			Name:      testBackupSchedule,
		}}

		// when
		actual, err := sut.Reconcile(testCtx, request)

		// then
		require.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, actual)
	})
	t.Run("should ignore status failed", func(t *testing.T) {
		// given
		backupSchedule := &k8sv1.BackupSchedule{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testBackupSchedule,
				Namespace: testNamespace,
			},
			Spec: k8sv1.BackupScheduleSpec{
				Schedule: "0 0 * * *",
				Provider: "velero",
			},
			Status: k8sv1.BackupScheduleStatus{Status: k8sv1.BackupScheduleStatusFailed},
		}

		backupScheduleClient := newMockEcosystemBackupScheduleInterface(t)
		backupScheduleClient.EXPECT().Get(testCtx, testBackupSchedule, metav1.GetOptions{}).Return(backupSchedule, nil)
		ecosystemClient := newMockEcosystemV1Alpha1Interface(t)
		ecosystemClient.EXPECT().BackupSchedules(testNamespace).Return(backupScheduleClient)
		clientSet := newMockEcosystemInterface(t)
		clientSet.EXPECT().EcosystemV1Alpha1().Return(ecosystemClient)

		manager := NewMockManager(t)
		recorder := newMockEventRecorder(t)
		requeueHandler := newMockRequeueHandler(t)

		sut := &backupScheduleReconciler{
			clientSet:      clientSet,
			namespace:      testNamespace,
			manager:        manager,
			recorder:       recorder,
			requeueHandler: requeueHandler,
		}
		request := ctrl.Request{NamespacedName: types.NamespacedName{
			Namespace: testNamespace,
			Name:      testBackupSchedule,
		}}

		// when
		actual, err := sut.Reconcile(testCtx, request)

		// then
		require.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, actual)
	})
	t.Run("should ignore status creating", func(t *testing.T) {
		// given
		backupSchedule := &k8sv1.BackupSchedule{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testBackupSchedule,
				Namespace: testNamespace,
			},
			Spec: k8sv1.BackupScheduleSpec{
				Schedule: "0 0 * * *",
				Provider: "velero",
			},
			Status: k8sv1.BackupScheduleStatus{Status: k8sv1.BackupScheduleStatusCreating},
		}

		backupScheduleClient := newMockEcosystemBackupScheduleInterface(t)
		backupScheduleClient.EXPECT().Get(testCtx, testBackupSchedule, metav1.GetOptions{}).Return(backupSchedule, nil)
		ecosystemClient := newMockEcosystemV1Alpha1Interface(t)
		ecosystemClient.EXPECT().BackupSchedules(testNamespace).Return(backupScheduleClient)
		clientSet := newMockEcosystemInterface(t)
		clientSet.EXPECT().EcosystemV1Alpha1().Return(ecosystemClient)

		manager := NewMockManager(t)
		recorder := newMockEventRecorder(t)
		requeueHandler := newMockRequeueHandler(t)

		sut := &backupScheduleReconciler{
			clientSet:      clientSet,
			namespace:      testNamespace,
			manager:        manager,
			recorder:       recorder,
			requeueHandler: requeueHandler,
		}
		request := ctrl.Request{NamespacedName: types.NamespacedName{
			Namespace: testNamespace,
			Name:      testBackupSchedule,
		}}

		// when
		actual, err := sut.Reconcile(testCtx, request)

		// then
		require.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, actual)
	})
	t.Run("should create if status new", func(t *testing.T) {
		// given
		backupSchedule := &k8sv1.BackupSchedule{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testBackupSchedule,
				Namespace: testNamespace,
			},
			Spec: k8sv1.BackupScheduleSpec{
				Schedule: "0 0 * * *",
				Provider: "velero",
			},
			Status: k8sv1.BackupScheduleStatus{Status: k8sv1.BackupScheduleStatusNew},
		}

		backupScheduleClient := newMockEcosystemBackupScheduleInterface(t)
		backupScheduleClient.EXPECT().Get(testCtx, testBackupSchedule, metav1.GetOptions{}).Return(backupSchedule, nil)
		ecosystemClient := newMockEcosystemV1Alpha1Interface(t)
		ecosystemClient.EXPECT().BackupSchedules(testNamespace).Return(backupScheduleClient)
		clientSet := newMockEcosystemInterface(t)
		clientSet.EXPECT().EcosystemV1Alpha1().Return(ecosystemClient)

		manager := NewMockManager(t)
		manager.EXPECT().create(testCtx, backupSchedule).Return(nil)
		recorder := newMockEventRecorder(t)
		recorder.EXPECT().Event(backupSchedule, "Normal", "Creation", "Creation successful")
		requeueHandler := newMockRequeueHandler(t)
		requeueHandler.EXPECT().Handle(testCtx, "Creation of backup schedule test-backup-schedule failed", backupSchedule, nil, "").Return(ctrl.Result{}, nil)

		sut := &backupScheduleReconciler{
			clientSet:      clientSet,
			namespace:      testNamespace,
			manager:        manager,
			recorder:       recorder,
			requeueHandler: requeueHandler,
		}
		request := ctrl.Request{NamespacedName: types.NamespacedName{
			Namespace: testNamespace,
			Name:      testBackupSchedule,
		}}

		// when
		actual, err := sut.Reconcile(testCtx, request)

		// then
		require.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, actual)
	})
	t.Run("should fail on create and fail to handle requeue", func(t *testing.T) {
		// given
		backupSchedule := &k8sv1.BackupSchedule{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testBackupSchedule,
				Namespace: testNamespace,
			},
			Spec: k8sv1.BackupScheduleSpec{
				Schedule: "0 0 * * *",
				Provider: "velero",
			},
			Status: k8sv1.BackupScheduleStatus{Status: k8sv1.BackupScheduleStatusNew},
		}

		backupScheduleClient := newMockEcosystemBackupScheduleInterface(t)
		backupScheduleClient.EXPECT().Get(testCtx, testBackupSchedule, metav1.GetOptions{}).Return(backupSchedule, nil)
		ecosystemClient := newMockEcosystemV1Alpha1Interface(t)
		ecosystemClient.EXPECT().BackupSchedules(testNamespace).Return(backupScheduleClient)
		clientSet := newMockEcosystemInterface(t)
		clientSet.EXPECT().EcosystemV1Alpha1().Return(ecosystemClient)

		manager := NewMockManager(t)
		manager.EXPECT().create(testCtx, backupSchedule).Return(assert.AnError)
		recorder := newMockEventRecorder(t)
		recorder.EXPECT().Event(backupSchedule, "Warning", "Creation", "Creation failed. Reason: assert.AnError general error for testing")
		recorder.EXPECT().Eventf(backupSchedule, "Warning", "Requeue", "Failed to requeue the %s.", "creation")
		requeueHandler := newMockRequeueHandler(t)
		requeueHandler.EXPECT().Handle(testCtx, "Creation of backup schedule test-backup-schedule failed", backupSchedule, assert.AnError, "").Return(ctrl.Result{}, assert.AnError)

		sut := &backupScheduleReconciler{
			clientSet:      clientSet,
			namespace:      testNamespace,
			manager:        manager,
			recorder:       recorder,
			requeueHandler: requeueHandler,
		}
		request := ctrl.Request{NamespacedName: types.NamespacedName{
			Namespace: testNamespace,
			Name:      testBackupSchedule,
		}}

		// when
		actual, err := sut.Reconcile(testCtx, request)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to handle requeue")
		assert.Equal(t, ctrl.Result{}, actual)
	})
	t.Run("should create if status created but cronjob not found", func(t *testing.T) {
		// given
		backupSchedule := &k8sv1.BackupSchedule{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testBackupSchedule,
				Namespace: testNamespace,
			},
			Spec: k8sv1.BackupScheduleSpec{
				Schedule: "0 0 * * *",
				Provider: "velero",
			},
			Status: k8sv1.BackupScheduleStatus{Status: k8sv1.BackupScheduleStatusCreated},
		}

		cronJobClient := newMockCronJobInterface(t)
		notFound := errors.NewNotFound(schema.GroupResource{}, testBackupSchedule)
		cronJobClient.EXPECT().Get(testCtx, backupSchedule.CronJobName(), metav1.GetOptions{}).Return(nil, notFound)
		batchClient := newMockBatchV1Interface(t)
		batchClient.EXPECT().CronJobs(testNamespace).Return(cronJobClient)

		backupScheduleClient := newMockEcosystemBackupScheduleInterface(t)
		backupScheduleClient.EXPECT().Get(testCtx, testBackupSchedule, metav1.GetOptions{}).Return(backupSchedule, nil)
		ecosystemClient := newMockEcosystemV1Alpha1Interface(t)
		ecosystemClient.EXPECT().BackupSchedules(testNamespace).Return(backupScheduleClient)

		clientSet := newMockEcosystemInterface(t)
		clientSet.EXPECT().EcosystemV1Alpha1().Return(ecosystemClient)
		clientSet.EXPECT().BatchV1().Return(batchClient)

		manager := NewMockManager(t)
		manager.EXPECT().create(testCtx, backupSchedule).Return(nil)
		recorder := newMockEventRecorder(t)
		recorder.EXPECT().Event(backupSchedule, "Normal", "Creation", "Creation successful")
		requeueHandler := newMockRequeueHandler(t)
		requeueHandler.EXPECT().Handle(testCtx, "Creation of backup schedule test-backup-schedule failed", backupSchedule, nil, "").Return(ctrl.Result{}, nil)

		sut := &backupScheduleReconciler{
			clientSet:      clientSet,
			namespace:      testNamespace,
			manager:        manager,
			recorder:       recorder,
			requeueHandler: requeueHandler,
		}
		request := ctrl.Request{NamespacedName: types.NamespacedName{
			Namespace: testNamespace,
			Name:      testBackupSchedule,
		}}

		// when
		actual, err := sut.Reconcile(testCtx, request)

		// then
		require.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, actual)
	})
	t.Run("should fail if status created and getting cronjob failed", func(t *testing.T) {
		// given
		backupSchedule := &k8sv1.BackupSchedule{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testBackupSchedule,
				Namespace: testNamespace,
			},
			Spec: k8sv1.BackupScheduleSpec{
				Schedule: "0 0 * * *",
				Provider: "velero",
			},
			Status: k8sv1.BackupScheduleStatus{Status: k8sv1.BackupScheduleStatusCreated},
		}

		cronJobClient := newMockCronJobInterface(t)
		cronJobClient.EXPECT().Get(testCtx, backupSchedule.CronJobName(), metav1.GetOptions{}).Return(nil, assert.AnError)
		batchClient := newMockBatchV1Interface(t)
		batchClient.EXPECT().CronJobs(testNamespace).Return(cronJobClient)

		backupScheduleClient := newMockEcosystemBackupScheduleInterface(t)
		backupScheduleClient.EXPECT().Get(testCtx, testBackupSchedule, metav1.GetOptions{}).Return(backupSchedule, nil)
		ecosystemClient := newMockEcosystemV1Alpha1Interface(t)
		ecosystemClient.EXPECT().BackupSchedules(testNamespace).Return(backupScheduleClient)

		clientSet := newMockEcosystemInterface(t)
		clientSet.EXPECT().EcosystemV1Alpha1().Return(ecosystemClient)
		clientSet.EXPECT().BatchV1().Return(batchClient)

		manager := NewMockManager(t)
		recorder := newMockEventRecorder(t)
		requeueHandler := newMockRequeueHandler(t)

		sut := &backupScheduleReconciler{
			clientSet:      clientSet,
			namespace:      testNamespace,
			manager:        manager,
			recorder:       recorder,
			requeueHandler: requeueHandler,
		}
		request := ctrl.Request{NamespacedName: types.NamespacedName{
			Namespace: testNamespace,
			Name:      testBackupSchedule,
		}}

		// when
		actual, err := sut.Reconcile(testCtx, request)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to find cron job for backup schedule test-backup-schedule")
		assert.ErrorContains(t, err, "failed to evaluate required operation for backup schedule test-backup-schedule")
		assert.Equal(t, ctrl.Result{}, actual)
	})

	t.Run("should update if status created and backup schedule does not equal cronjob schedule", func(t *testing.T) {
		// given
		backupSchedule := &k8sv1.BackupSchedule{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testBackupSchedule,
				Namespace: testNamespace,
			},
			Spec: k8sv1.BackupScheduleSpec{
				Schedule: "0 0 * * *",
				Provider: "velero",
			},
			Status: k8sv1.BackupScheduleStatus{Status: k8sv1.BackupScheduleStatusCreated},
		}

		cronJob := &batchv1.CronJob{
			ObjectMeta: metav1.ObjectMeta{
				Name:      backupSchedule.CronJobName(),
				Namespace: testNamespace,
			},
			Spec: batchv1.CronJobSpec{
				Schedule: "* * * * *",
				JobTemplate: batchv1.JobTemplateSpec{
					Spec: batchv1.JobSpec{
						Template: corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{{
									Name: testBackupSchedule,
									Env: []corev1.EnvVar{
										{Name: k8sv1.ProviderEnvVar, Value: "velero"}},
								}},
							},
						},
					},
				},
			},
		}

		cronJobClient := newMockCronJobInterface(t)
		cronJobClient.EXPECT().Get(testCtx, backupSchedule.CronJobName(), metav1.GetOptions{}).Return(cronJob, nil)
		batchClient := newMockBatchV1Interface(t)
		batchClient.EXPECT().CronJobs(testNamespace).Return(cronJobClient)

		backupScheduleClient := newMockEcosystemBackupScheduleInterface(t)
		backupScheduleClient.EXPECT().Get(testCtx, testBackupSchedule, metav1.GetOptions{}).Return(backupSchedule, nil)
		ecosystemClient := newMockEcosystemV1Alpha1Interface(t)
		ecosystemClient.EXPECT().BackupSchedules(testNamespace).Return(backupScheduleClient)

		clientSet := newMockEcosystemInterface(t)
		clientSet.EXPECT().EcosystemV1Alpha1().Return(ecosystemClient)
		clientSet.EXPECT().BatchV1().Return(batchClient)

		manager := NewMockManager(t)
		manager.EXPECT().update(testCtx, backupSchedule).Return(nil)
		recorder := newMockEventRecorder(t)
		recorder.EXPECT().Event(backupSchedule, "Normal", "Update", "Update successful")
		requeueHandler := newMockRequeueHandler(t)
		requeueHandler.EXPECT().Handle(testCtx, "Update of backup schedule test-backup-schedule failed", backupSchedule, nil, "created").Return(ctrl.Result{}, nil)

		sut := &backupScheduleReconciler{
			clientSet:      clientSet,
			namespace:      testNamespace,
			manager:        manager,
			recorder:       recorder,
			requeueHandler: requeueHandler,
		}
		request := ctrl.Request{NamespacedName: types.NamespacedName{
			Namespace: testNamespace,
			Name:      testBackupSchedule,
		}}

		// when
		actual, err := sut.Reconcile(testCtx, request)

		// then
		require.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, actual)
	})

	t.Run("should update if status created and backup provider does not equal cronjob provider", func(t *testing.T) {
		// given
		backupSchedule := &k8sv1.BackupSchedule{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testBackupSchedule,
				Namespace: testNamespace,
			},
			Spec: k8sv1.BackupScheduleSpec{
				Schedule: "0 0 * * *",
				Provider: "veleroV2",
			},
			Status: k8sv1.BackupScheduleStatus{Status: k8sv1.BackupScheduleStatusCreated},
		}

		cronJob := &batchv1.CronJob{
			ObjectMeta: metav1.ObjectMeta{
				Name:      backupSchedule.CronJobName(),
				Namespace: testNamespace,
			},
			Spec: batchv1.CronJobSpec{
				Schedule: "0 0 * * *",
				JobTemplate: batchv1.JobTemplateSpec{
					Spec: batchv1.JobSpec{
						Template: corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{{
									Name: testBackupSchedule,
									Env: []corev1.EnvVar{
										{Name: k8sv1.ProviderEnvVar, Value: "velero"}},
								}},
							},
						},
					},
				},
			},
		}

		cronJobClient := newMockCronJobInterface(t)
		cronJobClient.EXPECT().Get(testCtx, backupSchedule.CronJobName(), metav1.GetOptions{}).Return(cronJob, nil)
		batchClient := newMockBatchV1Interface(t)
		batchClient.EXPECT().CronJobs(testNamespace).Return(cronJobClient)

		backupScheduleClient := newMockEcosystemBackupScheduleInterface(t)
		backupScheduleClient.EXPECT().Get(testCtx, testBackupSchedule, metav1.GetOptions{}).Return(backupSchedule, nil)
		ecosystemClient := newMockEcosystemV1Alpha1Interface(t)
		ecosystemClient.EXPECT().BackupSchedules(testNamespace).Return(backupScheduleClient)

		clientSet := newMockEcosystemInterface(t)
		clientSet.EXPECT().EcosystemV1Alpha1().Return(ecosystemClient)
		clientSet.EXPECT().BatchV1().Return(batchClient)

		manager := NewMockManager(t)
		manager.EXPECT().update(testCtx, backupSchedule).Return(nil)
		recorder := newMockEventRecorder(t)
		recorder.EXPECT().Event(backupSchedule, "Normal", "Update", "Update successful")
		requeueHandler := newMockRequeueHandler(t)
		requeueHandler.EXPECT().Handle(testCtx, "Update of backup schedule test-backup-schedule failed", backupSchedule, nil, "created").Return(ctrl.Result{}, nil)

		sut := &backupScheduleReconciler{
			clientSet:      clientSet,
			namespace:      testNamespace,
			manager:        manager,
			recorder:       recorder,
			requeueHandler: requeueHandler,
		}
		request := ctrl.Request{NamespacedName: types.NamespacedName{
			Namespace: testNamespace,
			Name:      testBackupSchedule,
		}}

		// when
		actual, err := sut.Reconcile(testCtx, request)

		// then
		require.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, actual)
	})
}
