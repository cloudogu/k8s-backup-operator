package backupschedule

import (
	"context"
	"github.com/cloudogu/k8s-backup-operator/pkg/api/ecosystem"
	backupv1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	typedbatchv1 "k8s.io/client-go/kubernetes/typed/batch/v1"
	"testing"
)

var testCtx = context.TODO()

//goland:noinspection GoUnusedType
type ecosystemBackupScheduleInterface interface {
	ecosystem.BackupScheduleInterface
}

//goland:noinspection GoUnusedType
type ecosystemV1Alpha1Interface interface {
	ecosystem.V1Alpha1Interface
}

//goland:noinspection GoUnusedType
type batchV1Interface interface {
	typedbatchv1.BatchV1Interface
}

//goland:noinspection GoUnusedType
type cronJobInterface interface {
	typedbatchv1.CronJobInterface
}

func TestNewCreateManager(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given

		// when
		manager := newCreateManager(nil, nil, "test", "")

		// then
		require.NotNil(t, manager)
	})
}

func Test_defaultCreateManager_create(t *testing.T) {
	originalMaxTries := maxTries
	defer func() { maxTries = originalMaxTries }()
	maxTries = 1

	t.Run("success", func(t *testing.T) {
		// given
		backupScheduleName := "backupSchedule"
		testNamespace := "ecosystem"
		backupSchedule := &backupv1.BackupSchedule{
			ObjectMeta: metav1.ObjectMeta{Name: backupScheduleName, Namespace: testNamespace},
			Spec: backupv1.BackupScheduleSpec{
				Schedule: "0 0 * * *",
				Provider: "velero",
			},
		}

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(backupSchedule, corev1.EventTypeNormal, backupv1.CreateEventReason, "Creating backup schedule")

		backupScheduleClientMock := newMockEcosystemBackupScheduleInterface(t)
		v1Alpha1Mock := newMockEcosystemV1Alpha1InterfaceInterface(t)
		v1Alpha1Mock.EXPECT().BackupSchedules(testNamespace).Return(backupScheduleClientMock)
		clientMock := newMockEcosystemInterface(t)
		clientMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Mock)
		backupScheduleClientMock.EXPECT().UpdateStatusCreating(testCtx, backupSchedule).Return(backupSchedule, nil)
		backupScheduleClientMock.EXPECT().AddFinalizer(testCtx, backupSchedule, backupv1.BackupScheduleFinalizer).Return(backupSchedule, nil)
		backupScheduleClientMock.EXPECT().AddLabels(testCtx, backupSchedule).Return(backupSchedule, nil)
		backupScheduleClientMock.EXPECT().UpdateStatus(testCtx, backupSchedule, metav1.UpdateOptions{}).Return(backupSchedule, nil)
		backupScheduleClientMock.EXPECT().UpdateStatusCreated(testCtx, backupSchedule).Return(backupSchedule, nil)

		batchV1Mock := newMockBatchV1Interface(t)
		cronJobMock := newMockCronJobInterface(t)
		batchV1Mock.EXPECT().CronJobs(testNamespace).Return(cronJobMock)
		clientMock.EXPECT().BatchV1().Return(batchV1Mock)

		mode := int32(0550)
		expectedCreatedCronJob := &batchv1.CronJob{
			ObjectMeta: metav1.ObjectMeta{
				Name: "backup-schedule-backupSchedule", Namespace: testNamespace,
				Labels: map[string]string{
					"app":                          "ces",
					"k8s.cloudogu.com/part-of":     "backup",
					"app.kubernetes.io/created-by": "k8s-backup-operator",
					"app.kubernetes.io/part-of":    "k8s-backup-operator",
				},
			},
			Spec: batchv1.CronJobSpec{
				Schedule: "0 0 * * *",
				JobTemplate: batchv1.JobTemplateSpec{
					Spec: batchv1.JobSpec{
						Template: corev1.PodTemplateSpec{
							ObjectMeta: metav1.ObjectMeta{
								Name: "scheduled-backup-creator", Namespace: testNamespace,
								Labels: map[string]string{
									"app":                          "ces",
									"k8s.cloudogu.com/part-of":     "backup",
									"app.kubernetes.io/created-by": "k8s-backup-operator",
									"app.kubernetes.io/part-of":    "k8s-backup-operator",
								},
							},
							Spec: corev1.PodSpec{
								Volumes: []corev1.Volume{{
									Name: "k8s-backup-operator-create-backup-script",
									VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{
										LocalObjectReference: corev1.LocalObjectReference{Name: "k8s-create-backup-script"}, DefaultMode: &mode,
									}},
								}},
								Containers: []corev1.Container{{
									Name:    "backup-schedule-backupSchedule",
									Image:   "bitnami/kubectl:1.27.7",
									Command: []string{"/bin/entrypoint.sh"},
									Env: []corev1.EnvVar{
										{Name: "NAMESPACE", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.namespace"}}},
										{Name: "SCHEDULED_BACKUP_NAME", Value: "backupSchedule"},
										{Name: "PROVIDER", Value: "velero"},
									},
									VolumeMounts:    []corev1.VolumeMount{{Name: "k8s-backup-operator-create-backup-script", ReadOnly: true, MountPath: "/bin/entrypoint.sh", SubPath: "entrypoint.sh"}},
									ImagePullPolicy: corev1.PullIfNotPresent,
								}},
								RestartPolicy:      corev1.RestartPolicyOnFailure,
								ServiceAccountName: "k8s-backup-operator-scheduled-backup-creator-manager",
							},
						},
					},
				},
			},
		}
		cronJobMock.EXPECT().Create(testCtx, expectedCreatedCronJob, metav1.CreateOptions{}).Return(&batchv1.CronJob{}, nil)

		sut := &defaultCreateManager{
			recorder:     recorderMock,
			clientSet:    clientMock,
			namespace:    testNamespace,
			kubectlImage: "bitnami/kubectl:1.27.7",
		}

		// when
		err := sut.create(testCtx, backupSchedule)

		// then
		require.NoError(t, err)
		assert.Equal(t, "bitnami/kubectl:1.27.7", backupSchedule.Status.CurrentKubectlImage)
	})

	t.Run("should return error on update status creating error", func(t *testing.T) {
		// given
		backupScheduleName := "backupSchedule"
		testNamespace := "ecosystem"
		backupSchedule := &backupv1.BackupSchedule{ObjectMeta: metav1.ObjectMeta{Name: backupScheduleName, Namespace: testNamespace}, Spec: backupv1.BackupScheduleSpec{
			Schedule: "0 0 * * *",
			Provider: "velero",
		}}

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(backupSchedule, corev1.EventTypeNormal, backupv1.CreateEventReason, "Creating backup schedule")

		backupScheduleClientMock := newMockEcosystemBackupScheduleInterface(t)
		v1Alpha1Mock := newMockEcosystemV1Alpha1InterfaceInterface(t)
		v1Alpha1Mock.EXPECT().BackupSchedules(testNamespace).Return(backupScheduleClientMock)
		clientMock := newMockEcosystemInterface(t)
		clientMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Mock)
		backupScheduleClientMock.EXPECT().UpdateStatusCreating(testCtx, backupSchedule).Return(nil, assert.AnError)

		sut := &defaultCreateManager{recorder: recorderMock, clientSet: clientMock, namespace: testNamespace}

		// when
		err := sut.create(testCtx, backupSchedule)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to set status [creating] in backup schedule resource")
	})

	t.Run("should return error on finalizer update", func(t *testing.T) {
		// given
		backupScheduleName := "backupSchedule"
		testNamespace := "ecosystem"
		backupSchedule := &backupv1.BackupSchedule{
			ObjectMeta: metav1.ObjectMeta{Name: backupScheduleName, Namespace: testNamespace},
			Spec: backupv1.BackupScheduleSpec{
				Schedule: "0 0 * * *",
				Provider: "velero",
			},
		}

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(backupSchedule, corev1.EventTypeNormal, backupv1.CreateEventReason, "Creating backup schedule")

		backupScheduleClientMock := newMockEcosystemBackupScheduleInterface(t)
		v1Alpha1Mock := newMockEcosystemV1Alpha1InterfaceInterface(t)
		v1Alpha1Mock.EXPECT().BackupSchedules(testNamespace).Return(backupScheduleClientMock)
		clientMock := newMockEcosystemInterface(t)
		clientMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Mock)
		backupScheduleClientMock.EXPECT().UpdateStatusCreating(testCtx, backupSchedule).Return(backupSchedule, nil)
		backupScheduleClientMock.EXPECT().AddFinalizer(testCtx, backupSchedule, backupv1.BackupScheduleFinalizer).Return(nil, assert.AnError)

		sut := &defaultCreateManager{recorder: recorderMock, clientSet: clientMock, namespace: testNamespace}

		// when
		err := sut.create(testCtx, backupSchedule)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to add finalizer [cloudogu-backup-schedule-finalizer] in backup schedule resource")
	})

	t.Run("should return error on adding app=ces label", func(t *testing.T) {
		// given
		backupScheduleName := "backupSchedule"
		testNamespace := "ecosystem"
		backupSchedule := &backupv1.BackupSchedule{
			ObjectMeta: metav1.ObjectMeta{Name: backupScheduleName, Namespace: testNamespace},
			Spec: backupv1.BackupScheduleSpec{
				Schedule: "0 0 * * *",
				Provider: "velero",
			},
		}

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(backupSchedule, corev1.EventTypeNormal, backupv1.CreateEventReason, "Creating backup schedule")

		backupScheduleClientMock := newMockEcosystemBackupScheduleInterface(t)
		v1Alpha1Mock := newMockEcosystemV1Alpha1InterfaceInterface(t)
		v1Alpha1Mock.EXPECT().BackupSchedules(testNamespace).Return(backupScheduleClientMock)
		clientMock := newMockEcosystemInterface(t)
		clientMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Mock)
		backupScheduleClientMock.EXPECT().UpdateStatusCreating(testCtx, backupSchedule).Return(backupSchedule, nil)
		backupScheduleClientMock.EXPECT().AddFinalizer(testCtx, backupSchedule, backupv1.BackupScheduleFinalizer).Return(backupSchedule, nil)
		backupScheduleClientMock.EXPECT().AddLabels(testCtx, backupSchedule).Return(nil, assert.AnError)

		sut := &defaultCreateManager{recorder: recorderMock, clientSet: clientMock, namespace: testNamespace}

		// when
		err := sut.create(testCtx, backupSchedule)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to add labels to backup schedule resource")
	})

	t.Run("should return error on update status error", func(t *testing.T) {
		// given
		backupScheduleName := "backupSchedule"
		testNamespace := "ecosystem"
		backupSchedule := &backupv1.BackupSchedule{
			ObjectMeta: metav1.ObjectMeta{Name: backupScheduleName, Namespace: testNamespace},
			Spec: backupv1.BackupScheduleSpec{
				Schedule: "0 0 * * *",
				Provider: "velero",
			},
		}

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(backupSchedule, corev1.EventTypeNormal, backupv1.CreateEventReason, "Creating backup schedule")

		backupScheduleClientMock := newMockEcosystemBackupScheduleInterface(t)
		v1Alpha1Mock := newMockEcosystemV1Alpha1InterfaceInterface(t)
		v1Alpha1Mock.EXPECT().BackupSchedules(testNamespace).Return(backupScheduleClientMock)
		clientMock := newMockEcosystemInterface(t)
		clientMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Mock)
		backupScheduleClientMock.EXPECT().UpdateStatusCreating(testCtx, backupSchedule).Return(backupSchedule, nil)
		backupScheduleClientMock.EXPECT().AddFinalizer(testCtx, backupSchedule, backupv1.BackupScheduleFinalizer).Return(backupSchedule, nil)
		backupScheduleClientMock.EXPECT().AddLabels(testCtx, backupSchedule).Return(backupSchedule, nil)
		backupScheduleClientMock.EXPECT().UpdateStatus(testCtx, backupSchedule, metav1.UpdateOptions{}).Return(nil, assert.AnError)

		batchV1Mock := newMockBatchV1Interface(t)
		cronJobMock := newMockCronJobInterface(t)
		batchV1Mock.EXPECT().CronJobs(testNamespace).Return(cronJobMock)
		clientMock.EXPECT().BatchV1().Return(batchV1Mock)
		cronJobMock.EXPECT().Create(testCtx, mock.Anything, metav1.CreateOptions{}).Return(&batchv1.CronJob{}, nil)

		sut := &defaultCreateManager{recorder: recorderMock, clientSet: clientMock, namespace: testNamespace}

		// when
		err := sut.create(testCtx, backupSchedule)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to set currently used kubectl image in status of backup schedule resource")
	})

	t.Run("should return error on set status created error", func(t *testing.T) {
		// given
		backupScheduleName := "backupSchedule"
		testNamespace := "ecosystem"
		backupSchedule := &backupv1.BackupSchedule{
			ObjectMeta: metav1.ObjectMeta{Name: backupScheduleName, Namespace: testNamespace},
			Spec: backupv1.BackupScheduleSpec{
				Schedule: "0 0 * * *",
				Provider: "velero",
			},
		}

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(backupSchedule, corev1.EventTypeNormal, backupv1.CreateEventReason, "Creating backup schedule")

		backupScheduleClientMock := newMockEcosystemBackupScheduleInterface(t)
		v1Alpha1Mock := newMockEcosystemV1Alpha1InterfaceInterface(t)
		v1Alpha1Mock.EXPECT().BackupSchedules(testNamespace).Return(backupScheduleClientMock)
		clientMock := newMockEcosystemInterface(t)
		clientMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Mock)
		backupScheduleClientMock.EXPECT().UpdateStatusCreating(testCtx, backupSchedule).Return(backupSchedule, nil)
		backupScheduleClientMock.EXPECT().AddFinalizer(testCtx, backupSchedule, backupv1.BackupScheduleFinalizer).Return(backupSchedule, nil)
		backupScheduleClientMock.EXPECT().AddLabels(testCtx, backupSchedule).Return(backupSchedule, nil)
		backupScheduleClientMock.EXPECT().UpdateStatus(testCtx, backupSchedule, metav1.UpdateOptions{}).Return(backupSchedule, nil)
		backupScheduleClientMock.EXPECT().UpdateStatusCreated(testCtx, backupSchedule).Return(nil, assert.AnError)

		batchV1Mock := newMockBatchV1Interface(t)
		cronJobMock := newMockCronJobInterface(t)
		batchV1Mock.EXPECT().CronJobs(testNamespace).Return(cronJobMock)
		clientMock.EXPECT().BatchV1().Return(batchV1Mock)
		cronJobMock.EXPECT().Create(testCtx, mock.Anything, metav1.CreateOptions{}).Return(&batchv1.CronJob{}, nil)

		sut := &defaultCreateManager{recorder: recorderMock, clientSet: clientMock, namespace: testNamespace}

		// when
		err := sut.create(testCtx, backupSchedule)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to set status [created] in backup schedule resource")
	})

	t.Run("should retry 5 times on failed creation of the cron job and set status to failed", func(t *testing.T) {
		// given
		backupScheduleName := "backupSchedule"
		testNamespace := "ecosystem"
		backupSchedule := &backupv1.BackupSchedule{
			ObjectMeta: metav1.ObjectMeta{Name: backupScheduleName, Namespace: testNamespace},
			Spec: backupv1.BackupScheduleSpec{
				Schedule: "0 0 * * *",
				Provider: "velero",
			},
		}

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(backupSchedule, corev1.EventTypeNormal, backupv1.CreateEventReason, "Creating backup schedule")

		backupScheduleClientMock := newMockEcosystemBackupScheduleInterface(t)
		v1Alpha1Mock := newMockEcosystemV1Alpha1InterfaceInterface(t)
		v1Alpha1Mock.EXPECT().BackupSchedules(testNamespace).Return(backupScheduleClientMock)
		clientMock := newMockEcosystemInterface(t)
		clientMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Mock)
		backupScheduleClientMock.EXPECT().UpdateStatusCreating(testCtx, backupSchedule).Return(backupSchedule, nil)
		backupScheduleClientMock.EXPECT().AddFinalizer(testCtx, backupSchedule, backupv1.BackupScheduleFinalizer).Return(backupSchedule, nil)
		backupScheduleClientMock.EXPECT().AddLabels(testCtx, backupSchedule).Return(backupSchedule, nil)
		backupScheduleClientMock.EXPECT().UpdateStatusFailed(testCtx, backupSchedule).Return(backupSchedule, nil)

		batchV1Mock := newMockBatchV1Interface(t)
		cronJobMock := newMockCronJobInterface(t)
		batchV1Mock.EXPECT().CronJobs(testNamespace).Return(cronJobMock)
		clientMock.EXPECT().BatchV1().Return(batchV1Mock)
		cronJobMock.EXPECT().Create(testCtx, mock.Anything, metav1.CreateOptions{}).Return(nil, assert.AnError)

		sut := &defaultCreateManager{recorder: recorderMock, clientSet: clientMock, namespace: testNamespace}

		// when
		err := sut.create(testCtx, backupSchedule)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to create cron job for backup schedule")
	})

	t.Run("should return error on set status failed error", func(t *testing.T) {
		// given
		backupScheduleName := "backupSchedule"
		testNamespace := "ecosystem"
		backupSchedule := &backupv1.BackupSchedule{
			ObjectMeta: metav1.ObjectMeta{Name: backupScheduleName, Namespace: testNamespace},
			Spec: backupv1.BackupScheduleSpec{
				Schedule: "0 0 * * *",
				Provider: "velero",
			},
		}

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(backupSchedule, corev1.EventTypeNormal, backupv1.CreateEventReason, "Creating backup schedule")

		backupScheduleClientMock := newMockEcosystemBackupScheduleInterface(t)
		v1Alpha1Mock := newMockEcosystemV1Alpha1InterfaceInterface(t)
		v1Alpha1Mock.EXPECT().BackupSchedules(testNamespace).Return(backupScheduleClientMock)
		clientMock := newMockEcosystemInterface(t)
		clientMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Mock)
		backupScheduleClientMock.EXPECT().UpdateStatusCreating(testCtx, backupSchedule).Return(backupSchedule, nil)
		backupScheduleClientMock.EXPECT().AddFinalizer(testCtx, backupSchedule, backupv1.BackupScheduleFinalizer).Return(backupSchedule, nil)
		backupScheduleClientMock.EXPECT().AddLabels(testCtx, backupSchedule).Return(backupSchedule, nil)
		backupScheduleClientMock.EXPECT().UpdateStatusFailed(testCtx, backupSchedule).Return(nil, assert.AnError)

		batchV1Mock := newMockBatchV1Interface(t)
		cronJobMock := newMockCronJobInterface(t)
		batchV1Mock.EXPECT().CronJobs(testNamespace).Return(cronJobMock)
		clientMock.EXPECT().BatchV1().Return(batchV1Mock)
		cronJobMock.EXPECT().Create(testCtx, mock.Anything, metav1.CreateOptions{}).Return(nil, assert.AnError)

		sut := &defaultCreateManager{recorder: recorderMock, clientSet: clientMock, namespace: testNamespace}

		// when
		err := sut.create(testCtx, backupSchedule)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to create cron job for backup schedule")
		assert.ErrorContains(t, err, "failed to update backup schedule status to 'Failed'")
	})
}
