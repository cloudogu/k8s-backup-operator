package backupschedule

import (
	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestNewDeleteManager(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given

		// when
		manager := newDeleteManager(nil, nil, "test")

		// then
		require.NotNil(t, manager)
	})
}

func Test_defaultDeleteManager_delete(t *testing.T) {
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
		recorderMock.EXPECT().Event(backupSchedule, corev1.EventTypeNormal, backupv1.DeleteEventReason, "Deleting backup schedule")

		backupScheduleClientMock := newMockEcosystemBackupScheduleInterface(t)
		v1Alpha1Mock := newMockEcosystemV1Alpha1InterfaceInterface(t)
		v1Alpha1Mock.EXPECT().BackupSchedules(testNamespace).Return(backupScheduleClientMock)
		clientMock := newMockEcosystemInterface(t)
		clientMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Mock)
		backupScheduleClientMock.EXPECT().UpdateStatusDeleting(testCtx, backupSchedule).Return(backupSchedule, nil)
		backupScheduleClientMock.EXPECT().RemoveFinalizer(testCtx, backupSchedule, backupv1.BackupScheduleFinalizer).Return(backupSchedule, nil)

		batchV1Mock := newMockBatchV1Interface(t)
		cronJobMock := newMockCronJobInterface(t)
		batchV1Mock.EXPECT().CronJobs(testNamespace).Return(cronJobMock)
		clientMock.EXPECT().BatchV1().Return(batchV1Mock)
		cronJobMock.EXPECT().Delete(testCtx, backupSchedule.CronJobName(), metav1.DeleteOptions{}).Return(nil)

		sut := &defaultDeleteManager{recorder: recorderMock, clientSet: clientMock, namespace: testNamespace}

		// when
		err := sut.delete(testCtx, backupSchedule)

		// then
		require.NoError(t, err)
	})

	t.Run("should return error on update status deleting error", func(t *testing.T) {
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
		recorderMock.EXPECT().Event(backupSchedule, corev1.EventTypeNormal, backupv1.DeleteEventReason, "Deleting backup schedule")

		backupScheduleClientMock := newMockEcosystemBackupScheduleInterface(t)
		v1Alpha1Mock := newMockEcosystemV1Alpha1InterfaceInterface(t)
		v1Alpha1Mock.EXPECT().BackupSchedules(testNamespace).Return(backupScheduleClientMock)
		clientMock := newMockEcosystemInterface(t)
		clientMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Mock)
		backupScheduleClientMock.EXPECT().UpdateStatusDeleting(testCtx, backupSchedule).Return(nil, assert.AnError)

		sut := &defaultDeleteManager{recorder: recorderMock, clientSet: clientMock, namespace: testNamespace}

		// when
		err := sut.delete(testCtx, backupSchedule)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to set status [deleting] in backup schedule resource")
	})

	t.Run("should retry 5 times on failed deletion of the cron job and set status to failed", func(t *testing.T) {
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
		recorderMock.EXPECT().Event(backupSchedule, corev1.EventTypeNormal, backupv1.DeleteEventReason, "Deleting backup schedule")

		backupScheduleClientMock := newMockEcosystemBackupScheduleInterface(t)
		v1Alpha1Mock := newMockEcosystemV1Alpha1InterfaceInterface(t)
		v1Alpha1Mock.EXPECT().BackupSchedules(testNamespace).Return(backupScheduleClientMock)
		clientMock := newMockEcosystemInterface(t)
		clientMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Mock)
		backupScheduleClientMock.EXPECT().UpdateStatusDeleting(testCtx, backupSchedule).Return(backupSchedule, nil)
		backupScheduleClientMock.EXPECT().UpdateStatusFailed(testCtx, backupSchedule).Return(backupSchedule, nil)

		batchV1Mock := newMockBatchV1Interface(t)
		cronJobMock := newMockCronJobInterface(t)
		batchV1Mock.EXPECT().CronJobs(testNamespace).Return(cronJobMock)
		clientMock.EXPECT().BatchV1().Return(batchV1Mock)
		cronJobMock.EXPECT().Delete(testCtx, backupSchedule.CronJobName(), metav1.DeleteOptions{}).Return(assert.AnError)

		sut := &defaultDeleteManager{recorder: recorderMock, clientSet: clientMock, namespace: testNamespace}

		// when
		err := sut.delete(testCtx, backupSchedule)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to delete cron job for backup schedule")
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
		recorderMock.EXPECT().Event(backupSchedule, corev1.EventTypeNormal, backupv1.DeleteEventReason, "Deleting backup schedule")

		backupScheduleClientMock := newMockEcosystemBackupScheduleInterface(t)
		v1Alpha1Mock := newMockEcosystemV1Alpha1InterfaceInterface(t)
		v1Alpha1Mock.EXPECT().BackupSchedules(testNamespace).Return(backupScheduleClientMock)
		clientMock := newMockEcosystemInterface(t)
		clientMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Mock)
		backupScheduleClientMock.EXPECT().UpdateStatusDeleting(testCtx, backupSchedule).Return(backupSchedule, nil)
		backupScheduleClientMock.EXPECT().UpdateStatusFailed(testCtx, backupSchedule).Return(nil, assert.AnError)

		batchV1Mock := newMockBatchV1Interface(t)
		cronJobMock := newMockCronJobInterface(t)
		batchV1Mock.EXPECT().CronJobs(testNamespace).Return(cronJobMock)
		clientMock.EXPECT().BatchV1().Return(batchV1Mock)
		cronJobMock.EXPECT().Delete(testCtx, backupSchedule.CronJobName(), metav1.DeleteOptions{}).Return(assert.AnError)

		sut := &defaultDeleteManager{recorder: recorderMock, clientSet: clientMock, namespace: testNamespace}

		// when
		err := sut.delete(testCtx, backupSchedule)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to delete cron job for backup schedule")
		assert.ErrorContains(t, err, "failed to update backup schedule status to 'Failed'")
	})

	t.Run("should return error on removing finalizer", func(t *testing.T) {
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
		recorderMock.EXPECT().Event(backupSchedule, corev1.EventTypeNormal, backupv1.DeleteEventReason, "Deleting backup schedule")

		backupScheduleClientMock := newMockEcosystemBackupScheduleInterface(t)
		v1Alpha1Mock := newMockEcosystemV1Alpha1InterfaceInterface(t)
		v1Alpha1Mock.EXPECT().BackupSchedules(testNamespace).Return(backupScheduleClientMock)
		clientMock := newMockEcosystemInterface(t)
		clientMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Mock)
		backupScheduleClientMock.EXPECT().UpdateStatusDeleting(testCtx, backupSchedule).Return(backupSchedule, nil)
		backupScheduleClientMock.EXPECT().RemoveFinalizer(testCtx, backupSchedule, backupv1.BackupScheduleFinalizer).Return(nil, assert.AnError)

		batchV1Mock := newMockBatchV1Interface(t)
		cronJobMock := newMockCronJobInterface(t)
		batchV1Mock.EXPECT().CronJobs(testNamespace).Return(cronJobMock)
		clientMock.EXPECT().BatchV1().Return(batchV1Mock)
		cronJobMock.EXPECT().Delete(testCtx, backupSchedule.CronJobName(), metav1.DeleteOptions{}).Return(nil)

		sut := &defaultDeleteManager{recorder: recorderMock, clientSet: clientMock, namespace: testNamespace}

		// when
		err := sut.delete(testCtx, backupSchedule)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to remove finalizer [cloudogu-backup-schedule-finalizer] in backup schedule resource")
	})

}
