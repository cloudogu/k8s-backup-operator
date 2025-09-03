package backupschedule

import (
	k8sv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/k8s-backup-operator/pkg/additionalimages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestNewUpdateManager(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given

		// when
		manager := newUpdateManager(nil, nil, "test", additionalimages.ImageConfig{})

		// then
		require.NotNil(t, manager)
	})
}

func Test_defaultUpdateManager_update(t *testing.T) {
	originalMaxTries := maxTries
	defer func() { maxTries = originalMaxTries }()
	maxTries = 1

	t.Run("success", func(t *testing.T) {
		// given
		backupScheduleName := "backupSchedule"
		testNamespace := "ecosystem"
		backupSchedule := &k8sv1.BackupSchedule{
			ObjectMeta: metav1.ObjectMeta{Name: backupScheduleName, Namespace: testNamespace},
			Spec: k8sv1.BackupScheduleSpec{
				Schedule: "0 0 * * *",
				Provider: "velero",
			},
		}

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(backupSchedule, corev1.EventTypeNormal, k8sv1.UpdateEventReason, "Updating backup schedule")

		backupScheduleClientMock := newMockEcosystemBackupScheduleInterface(t)
		v1Alpha1Mock := newMockEcosystemV1Alpha1InterfaceInterface(t)
		v1Alpha1Mock.EXPECT().BackupSchedules(testNamespace).Return(backupScheduleClientMock)
		clientMock := newMockEcosystemInterface(t)
		clientMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Mock)
		backupScheduleClientMock.EXPECT().UpdateStatusUpdating(testCtx, backupSchedule).Return(backupSchedule, nil)
		backupScheduleClientMock.EXPECT().UpdateStatusCreated(testCtx, backupSchedule).Return(backupSchedule, nil)

		batchV1Mock := newMockBatchV1Interface(t)
		cronJobMock := newMockCronJobInterface(t)
		batchV1Mock.EXPECT().CronJobs(testNamespace).Return(cronJobMock)
		clientMock.EXPECT().BatchV1().Return(batchV1Mock)
		cronJob := &batchv1.CronJob{}
		cronJobMock.EXPECT().Get(testCtx, backupSchedule.CronJobName(), metav1.GetOptions{}).Return(cronJob, nil)
		cronJobMock.EXPECT().Update(testCtx, cronJob, metav1.UpdateOptions{}).Return(&batchv1.CronJob{}, nil)

		image := additionalimages.ImageConfig{OperatorImage: "MyImage"}
		sut := &defaultUpdateManager{recorder: recorderMock, clientSet: clientMock, namespace: testNamespace, imageConfig: image}

		// when
		err := sut.update(testCtx, backupSchedule)

		// then
		require.NoError(t, err)
		assert.Equal(t, "0 0 * * *", cronJob.Spec.Schedule)
		args := cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Args
		assert.True(t, len(args) == 3)
		expectedProviderArg := "--provider=velero"
		assert.Contains(t, args, expectedProviderArg)
		assert.Equal(t, "MyImage", cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image)
	})

	t.Run("should return error on update status updating error", func(t *testing.T) {
		// given
		backupScheduleName := "backupSchedule"
		testNamespace := "ecosystem"
		backupSchedule := &k8sv1.BackupSchedule{
			ObjectMeta: metav1.ObjectMeta{Name: backupScheduleName, Namespace: testNamespace},
			Spec: k8sv1.BackupScheduleSpec{
				Schedule: "0 0 * * *",
				Provider: "velero",
			},
		}

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(backupSchedule, corev1.EventTypeNormal, k8sv1.UpdateEventReason, "Updating backup schedule")

		backupScheduleClientMock := newMockEcosystemBackupScheduleInterface(t)
		v1Alpha1Mock := newMockEcosystemV1Alpha1InterfaceInterface(t)
		v1Alpha1Mock.EXPECT().BackupSchedules(testNamespace).Return(backupScheduleClientMock)
		clientMock := newMockEcosystemInterface(t)
		clientMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Mock)
		backupScheduleClientMock.EXPECT().UpdateStatusUpdating(testCtx, backupSchedule).Return(nil, assert.AnError)

		sut := &defaultUpdateManager{recorder: recorderMock, clientSet: clientMock, namespace: testNamespace}

		// when
		err := sut.update(testCtx, backupSchedule)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to set status [updating] in backup schedule resource")
	})

	t.Run("should retry 5 times on failed GET of the cron job and set status to failed", func(t *testing.T) {
		// given
		backupScheduleName := "backupSchedule"
		testNamespace := "ecosystem"
		backupSchedule := &k8sv1.BackupSchedule{
			ObjectMeta: metav1.ObjectMeta{Name: backupScheduleName, Namespace: testNamespace},
			Spec: k8sv1.BackupScheduleSpec{
				Schedule: "0 0 * * *",
				Provider: "velero",
			},
		}

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(backupSchedule, corev1.EventTypeNormal, k8sv1.UpdateEventReason, "Updating backup schedule")

		backupScheduleClientMock := newMockEcosystemBackupScheduleInterface(t)
		v1Alpha1Mock := newMockEcosystemV1Alpha1InterfaceInterface(t)
		v1Alpha1Mock.EXPECT().BackupSchedules(testNamespace).Return(backupScheduleClientMock)
		clientMock := newMockEcosystemInterface(t)
		clientMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Mock)
		backupScheduleClientMock.EXPECT().UpdateStatusUpdating(testCtx, backupSchedule).Return(backupSchedule, nil)
		backupScheduleClientMock.EXPECT().UpdateStatusFailed(testCtx, backupSchedule).Return(backupSchedule, nil)

		batchV1Mock := newMockBatchV1Interface(t)
		cronJobMock := newMockCronJobInterface(t)
		batchV1Mock.EXPECT().CronJobs(testNamespace).Return(cronJobMock)
		clientMock.EXPECT().BatchV1().Return(batchV1Mock)
		cronJobMock.EXPECT().Get(testCtx, backupSchedule.CronJobName(), metav1.GetOptions{}).Return(nil, assert.AnError)

		sut := &defaultUpdateManager{recorder: recorderMock, clientSet: clientMock, namespace: testNamespace}

		// when
		err := sut.update(testCtx, backupSchedule)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to update cron job for backup schedule")
	})

	t.Run("should retry 5 times on failed update of the cron job and set status to failed", func(t *testing.T) {
		// given
		backupScheduleName := "backupSchedule"
		testNamespace := "ecosystem"
		backupSchedule := &k8sv1.BackupSchedule{
			ObjectMeta: metav1.ObjectMeta{Name: backupScheduleName, Namespace: testNamespace},
			Spec: k8sv1.BackupScheduleSpec{
				Schedule: "0 0 * * *",
				Provider: "velero",
			},
		}

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(backupSchedule, corev1.EventTypeNormal, k8sv1.UpdateEventReason, "Updating backup schedule")

		backupScheduleClientMock := newMockEcosystemBackupScheduleInterface(t)
		v1Alpha1Mock := newMockEcosystemV1Alpha1InterfaceInterface(t)
		v1Alpha1Mock.EXPECT().BackupSchedules(testNamespace).Return(backupScheduleClientMock)
		clientMock := newMockEcosystemInterface(t)
		clientMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Mock)
		backupScheduleClientMock.EXPECT().UpdateStatusUpdating(testCtx, backupSchedule).Return(backupSchedule, nil)
		backupScheduleClientMock.EXPECT().UpdateStatusFailed(testCtx, backupSchedule).Return(backupSchedule, nil)

		batchV1Mock := newMockBatchV1Interface(t)
		cronJobMock := newMockCronJobInterface(t)
		batchV1Mock.EXPECT().CronJobs(testNamespace).Return(cronJobMock)
		clientMock.EXPECT().BatchV1().Return(batchV1Mock)
		cronJobMock.EXPECT().Get(testCtx, backupSchedule.CronJobName(), metav1.GetOptions{}).Return(&batchv1.CronJob{}, nil)
		cronJobMock.EXPECT().Update(testCtx, mock.Anything, metav1.UpdateOptions{}).Return(nil, assert.AnError)

		sut := &defaultUpdateManager{recorder: recorderMock, clientSet: clientMock, namespace: testNamespace}

		// when
		err := sut.update(testCtx, backupSchedule)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to update cron job for backup schedule")
	})

	t.Run("should retry 5 times on failed update of the cron job and set status to failed", func(t *testing.T) {
		// given
		backupScheduleName := "backupSchedule"
		testNamespace := "ecosystem"
		backupSchedule := &k8sv1.BackupSchedule{
			ObjectMeta: metav1.ObjectMeta{Name: backupScheduleName, Namespace: testNamespace},
			Spec: k8sv1.BackupScheduleSpec{
				Schedule: "0 0 * * *",
				Provider: "velero",
			},
		}

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(backupSchedule, corev1.EventTypeNormal, k8sv1.UpdateEventReason, "Updating backup schedule")

		backupScheduleClientMock := newMockEcosystemBackupScheduleInterface(t)
		v1Alpha1Mock := newMockEcosystemV1Alpha1InterfaceInterface(t)
		v1Alpha1Mock.EXPECT().BackupSchedules(testNamespace).Return(backupScheduleClientMock)
		clientMock := newMockEcosystemInterface(t)
		clientMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Mock)
		backupScheduleClientMock.EXPECT().UpdateStatusUpdating(testCtx, backupSchedule).Return(backupSchedule, nil)
		backupScheduleClientMock.EXPECT().UpdateStatusFailed(testCtx, backupSchedule).Return(nil, assert.AnError)

		batchV1Mock := newMockBatchV1Interface(t)
		cronJobMock := newMockCronJobInterface(t)
		batchV1Mock.EXPECT().CronJobs(testNamespace).Return(cronJobMock)
		clientMock.EXPECT().BatchV1().Return(batchV1Mock)
		cronJobMock.EXPECT().Get(testCtx, backupSchedule.CronJobName(), metav1.GetOptions{}).Return(&batchv1.CronJob{}, nil)
		cronJobMock.EXPECT().Update(testCtx, mock.Anything, metav1.UpdateOptions{}).Return(nil, assert.AnError)

		sut := &defaultUpdateManager{recorder: recorderMock, clientSet: clientMock, namespace: testNamespace}

		// when
		err := sut.update(testCtx, backupSchedule)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to update cron job for backup schedule")
		assert.ErrorContains(t, err, "failed to update backup schedule status to 'Failed'")
	})

	t.Run("should return error on update status updating error", func(t *testing.T) {
		// given
		backupScheduleName := "backupSchedule"
		testNamespace := "ecosystem"
		backupSchedule := &k8sv1.BackupSchedule{
			ObjectMeta: metav1.ObjectMeta{Name: backupScheduleName, Namespace: testNamespace},
			Spec: k8sv1.BackupScheduleSpec{
				Schedule: "0 0 * * *",
				Provider: "velero",
			},
		}

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(backupSchedule, corev1.EventTypeNormal, k8sv1.UpdateEventReason, "Updating backup schedule")

		backupScheduleClientMock := newMockEcosystemBackupScheduleInterface(t)
		v1Alpha1Mock := newMockEcosystemV1Alpha1InterfaceInterface(t)
		v1Alpha1Mock.EXPECT().BackupSchedules(testNamespace).Return(backupScheduleClientMock)
		clientMock := newMockEcosystemInterface(t)
		clientMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Mock)
		backupScheduleClientMock.EXPECT().UpdateStatusUpdating(testCtx, backupSchedule).Return(backupSchedule, nil)
		backupScheduleClientMock.EXPECT().UpdateStatusCreated(testCtx, backupSchedule).Return(nil, assert.AnError)

		batchV1Mock := newMockBatchV1Interface(t)
		cronJobMock := newMockCronJobInterface(t)
		batchV1Mock.EXPECT().CronJobs(testNamespace).Return(cronJobMock)
		clientMock.EXPECT().BatchV1().Return(batchV1Mock)
		cronJobMock.EXPECT().Get(testCtx, backupSchedule.CronJobName(), metav1.GetOptions{}).Return(&batchv1.CronJob{}, nil)
		cronJobMock.EXPECT().Update(testCtx, mock.Anything, metav1.UpdateOptions{}).Return(&batchv1.CronJob{}, nil)

		sut := &defaultUpdateManager{recorder: recorderMock, clientSet: clientMock, namespace: testNamespace}

		// when
		err := sut.update(testCtx, backupSchedule)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to set status [created] in backup schedule resource")
	})
}
