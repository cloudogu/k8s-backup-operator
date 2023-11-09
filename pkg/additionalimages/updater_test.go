package additionalimages

import (
	context "context"
	"fmt"
	backupv1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"testing"
)

func Test_updater_Update(t *testing.T) {
	kubectlImage := "bitnami/kubectl:1.1.1"

	t.Run("success", func(t *testing.T) {
		// given
		backupScheduleClientMock := newMockBackupScheduleClient(t)
		v1Alpha1Mock := newMockEcosystemV1Alpha1Interface(t)
		v1Alpha1Mock.EXPECT().BackupSchedules(testNamespace).Return(backupScheduleClientMock)
		clientMock := newMockEcosystemClientSet(t)
		clientMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Mock)

		scheduleUpToDate := backupv1.BackupSchedule{
			Status: backupv1.BackupScheduleStatus{
				CurrentKubectlImage: kubectlImage,
			},
		}
		scheduleOldImage := backupv1.BackupSchedule{
			ObjectMeta: metav1.ObjectMeta{
				Name: "OldImage",
			},
			Status: backupv1.BackupScheduleStatus{
				CurrentKubectlImage: "bitnami/kubectl:1.1.0",
			},
		}
		scheduleNoImage := backupv1.BackupSchedule{
			ObjectMeta: metav1.ObjectMeta{
				Name: "NoImage",
			},
			Status: backupv1.BackupScheduleStatus{},
		}
		schedules := []backupv1.BackupSchedule{
			scheduleUpToDate,
			scheduleOldImage,
			scheduleNoImage,
		}
		scheduleList := backupv1.BackupScheduleList{
			Items: schedules,
		}
		backupScheduleClientMock.EXPECT().List(testCtx, metav1.ListOptions{}).Return(&scheduleList, nil)

		batchV1Mock := newMockBatchV1Interface(t)
		cronJobMock := newMockCronJobInterface(t)
		batchV1Mock.EXPECT().CronJobs(testNamespace).Return(cronJobMock)
		clientMock.EXPECT().BatchV1().Return(batchV1Mock)

		cronJobOldImage := &batchv1.CronJob{}
		cronJobOldImage.Spec.JobTemplate.Spec.Template.Spec.Containers = []corev1.Container{{}}
		cronJobMock.EXPECT().Get(testCtx, scheduleOldImage.CronJobName(), metav1.GetOptions{}).Return(cronJobOldImage, nil)
		cronJobMock.EXPECT().Update(testCtx, cronJobOldImage, metav1.UpdateOptions{}).Return(cronJobOldImage, nil)

		cronJobNoImage := &batchv1.CronJob{}
		cronJobNoImage.Spec.JobTemplate.Spec.Template.Spec.Containers = []corev1.Container{{}}
		cronJobMock.EXPECT().Get(testCtx, scheduleNoImage.CronJobName(), metav1.GetOptions{}).Return(cronJobNoImage, nil)
		cronJobMock.EXPECT().Update(testCtx, cronJobOldImage, metav1.UpdateOptions{}).Return(cronJobNoImage, nil)

		backupScheduleClientMock.EXPECT().UpdateStatus(testCtx, mock.Anything, metav1.UpdateOptions{}).
			Run(func(ctx context.Context, backupSchedule *backupv1.BackupSchedule, opts metav1.UpdateOptions) {
				assert.Equal(t, kubectlImage, backupSchedule.Status.CurrentKubectlImage)
			}).Return(&backupv1.BackupSchedule{}, nil).Times(2)

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Eventf(&scheduleOldImage, corev1.EventTypeNormal, imageUpdateEventReason,
			"Updated kubectl image in CronJob to %s.", kubectlImage)
		recorderMock.EXPECT().Eventf(&scheduleNoImage, corev1.EventTypeNormal, imageUpdateEventReason,
			"Updated kubectl image in CronJob to %s.", kubectlImage)

		sut := NewUpdater(clientMock, testNamespace, kubectlImage, recorderMock)

		// when
		err := sut.Update(testCtx)

		// then
		require.NoError(t, err)
		assert.Equal(t, kubectlImage, cronJobOldImage.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image)
		assert.Equal(t, kubectlImage, cronJobNoImage.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image)
	})

	t.Run("should failed to list backup schedules", func(t *testing.T) {
		// given
		backupScheduleClientMock := newMockBackupScheduleClient(t)
		v1Alpha1Mock := newMockEcosystemV1Alpha1Interface(t)
		v1Alpha1Mock.EXPECT().BackupSchedules(testNamespace).Return(backupScheduleClientMock)
		clientMock := newMockEcosystemClientSet(t)
		clientMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Mock)

		backupScheduleClientMock.EXPECT().List(testCtx, metav1.ListOptions{}).Return(nil, assert.AnError)

		sut := NewUpdater(clientMock, testNamespace, kubectlImage, nil)

		// when
		err := sut.Update(testCtx)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to list backup schedules whose images are not up to date:")
	})

	t.Run("should ignore not found error on cron job", func(t *testing.T) {
		// given
		backupScheduleClientMock := newMockBackupScheduleClient(t)
		v1Alpha1Mock := newMockEcosystemV1Alpha1Interface(t)
		v1Alpha1Mock.EXPECT().BackupSchedules(testNamespace).Return(backupScheduleClientMock)
		clientMock := newMockEcosystemClientSet(t)
		clientMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Mock)

		scheduleOldImage := backupv1.BackupSchedule{
			Status: backupv1.BackupScheduleStatus{
				CurrentKubectlImage: "bitnami/kubectl:1.1.0",
			},
		}
		schedules := []backupv1.BackupSchedule{
			scheduleOldImage,
		}
		scheduleList := backupv1.BackupScheduleList{
			Items: schedules,
		}
		backupScheduleClientMock.EXPECT().List(testCtx, metav1.ListOptions{}).Return(&scheduleList, nil)

		batchV1Mock := newMockBatchV1Interface(t)
		cronJobMock := newMockCronJobInterface(t)
		batchV1Mock.EXPECT().CronJobs(testNamespace).Return(cronJobMock)
		clientMock.EXPECT().BatchV1().Return(batchV1Mock)

		notFound := errors.NewNotFound(schema.GroupResource{}, scheduleOldImage.CronJobName())
		cronJobMock.EXPECT().Get(testCtx, scheduleOldImage.CronJobName(), metav1.GetOptions{}).Return(nil, notFound)

		backupScheduleClientMock.EXPECT().UpdateStatus(testCtx, mock.Anything, metav1.UpdateOptions{}).
			Run(func(ctx context.Context, backupSchedule *backupv1.BackupSchedule, opts metav1.UpdateOptions) {
				assert.Equal(t, kubectlImage, backupSchedule.Status.CurrentKubectlImage)
			}).Return(&backupv1.BackupSchedule{}, nil)

		recorderMock := newMockEventRecorder(t)
		message := fmt.Sprintf("Cron job %s for backup schedule %s does not exist. Skipping kubectl image update.", scheduleOldImage.CronJobName(), scheduleOldImage.Name)
		recorderMock.EXPECT().Event(&scheduleOldImage, corev1.EventTypeWarning, imageUpdateEventReason, message)

		sut := NewUpdater(clientMock, testNamespace, kubectlImage, recorderMock)

		// when
		err := sut.Update(testCtx)

		// then
		require.NoError(t, err)
	})

	t.Run("should fail on getting cron job", func(t *testing.T) {
		// given
		backupScheduleClientMock := newMockBackupScheduleClient(t)
		v1Alpha1Mock := newMockEcosystemV1Alpha1Interface(t)
		v1Alpha1Mock.EXPECT().BackupSchedules(testNamespace).Return(backupScheduleClientMock)
		clientMock := newMockEcosystemClientSet(t)
		clientMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Mock)

		scheduleGetError := backupv1.BackupSchedule{
			ObjectMeta: metav1.ObjectMeta{
				Name: "GetError",
			},
			Status: backupv1.BackupScheduleStatus{
				CurrentKubectlImage: "bitnami/kubectl:1.1.0",
			},
		}
		scheduleOldImage := backupv1.BackupSchedule{
			ObjectMeta: metav1.ObjectMeta{
				Name: "OldImage",
			},
			Status: backupv1.BackupScheduleStatus{
				CurrentKubectlImage: "bitnami/kubectl:1.1.0",
			},
		}
		schedules := []backupv1.BackupSchedule{
			scheduleOldImage,
			scheduleGetError,
		}
		scheduleList := backupv1.BackupScheduleList{
			Items: schedules,
		}
		backupScheduleClientMock.EXPECT().List(testCtx, metav1.ListOptions{}).Return(&scheduleList, nil)

		batchV1Mock := newMockBatchV1Interface(t)
		cronJobMock := newMockCronJobInterface(t)
		batchV1Mock.EXPECT().CronJobs(testNamespace).Return(cronJobMock)
		clientMock.EXPECT().BatchV1().Return(batchV1Mock)

		cronJobMock.EXPECT().Get(testCtx, scheduleGetError.CronJobName(), metav1.GetOptions{}).Return(nil, assert.AnError)

		cronJobOldImage := &batchv1.CronJob{}
		cronJobOldImage.Spec.JobTemplate.Spec.Template.Spec.Containers = []corev1.Container{{}}
		cronJobMock.EXPECT().Get(testCtx, scheduleOldImage.CronJobName(), metav1.GetOptions{}).Return(cronJobOldImage, nil)
		cronJobMock.EXPECT().Update(testCtx, cronJobOldImage, metav1.UpdateOptions{}).Return(cronJobOldImage, nil)

		backupScheduleClientMock.EXPECT().UpdateStatus(testCtx, mock.Anything, metav1.UpdateOptions{}).
			Run(func(ctx context.Context, backupSchedule *backupv1.BackupSchedule, opts metav1.UpdateOptions) {
				assert.Equal(t, scheduleOldImage.Name, backupSchedule.Name)
				assert.Equal(t, kubectlImage, backupSchedule.Status.CurrentKubectlImage)
			}).Return(&backupv1.BackupSchedule{}, nil).Once()

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Eventf(&scheduleOldImage, corev1.EventTypeNormal, imageUpdateEventReason,
			"Updated kubectl image in CronJob to %s.", kubectlImage)

		sut := NewUpdater(clientMock, testNamespace, kubectlImage, recorderMock)

		// when
		err := sut.Update(testCtx)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to update additional images in cron job")
		assert.ErrorContains(t, err, "failed to get cron job")
	})

	t.Run("should fail on updating cron job", func(t *testing.T) {
		// given
		backupScheduleClientMock := newMockBackupScheduleClient(t)
		v1Alpha1Mock := newMockEcosystemV1Alpha1Interface(t)
		v1Alpha1Mock.EXPECT().BackupSchedules(testNamespace).Return(backupScheduleClientMock)
		clientMock := newMockEcosystemClientSet(t)
		clientMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Mock)

		scheduleGetError := backupv1.BackupSchedule{
			ObjectMeta: metav1.ObjectMeta{
				Name: "GetError",
			},
			Status: backupv1.BackupScheduleStatus{
				CurrentKubectlImage: "bitnami/kubectl:1.1.0",
			},
		}
		scheduleOldImage := backupv1.BackupSchedule{
			ObjectMeta: metav1.ObjectMeta{
				Name: "OldImage",
			},
			Status: backupv1.BackupScheduleStatus{
				CurrentKubectlImage: "bitnami/kubectl:1.1.0",
			},
		}
		schedules := []backupv1.BackupSchedule{
			scheduleOldImage,
			scheduleGetError,
		}
		scheduleList := backupv1.BackupScheduleList{
			Items: schedules,
		}
		backupScheduleClientMock.EXPECT().List(testCtx, metav1.ListOptions{}).Return(&scheduleList, nil)

		batchV1Mock := newMockBatchV1Interface(t)
		cronJobMock := newMockCronJobInterface(t)
		batchV1Mock.EXPECT().CronJobs(testNamespace).Return(cronJobMock)
		clientMock.EXPECT().BatchV1().Return(batchV1Mock)

		cronJobUpdateError := &batchv1.CronJob{}
		cronJobUpdateError.Spec.JobTemplate.Spec.Template.Spec.Containers = []corev1.Container{{}}
		cronJobMock.EXPECT().Get(testCtx, scheduleGetError.CronJobName(), metav1.GetOptions{}).Return(cronJobUpdateError, nil)
		cronJobMock.EXPECT().Update(testCtx, cronJobUpdateError, metav1.UpdateOptions{}).Return(nil, assert.AnError)

		cronJobOldImage := &batchv1.CronJob{}
		cronJobOldImage.Spec.JobTemplate.Spec.Template.Spec.Containers = []corev1.Container{{}}
		cronJobMock.EXPECT().Get(testCtx, scheduleOldImage.CronJobName(), metav1.GetOptions{}).Return(cronJobOldImage, nil)
		cronJobMock.EXPECT().Update(testCtx, cronJobOldImage, metav1.UpdateOptions{}).Return(cronJobOldImage, nil)

		backupScheduleClientMock.EXPECT().UpdateStatus(testCtx, mock.Anything, metav1.UpdateOptions{}).
			Run(func(ctx context.Context, backupSchedule *backupv1.BackupSchedule, opts metav1.UpdateOptions) {
				assert.Equal(t, scheduleOldImage.Name, backupSchedule.Name)
				assert.Equal(t, kubectlImage, backupSchedule.Status.CurrentKubectlImage)
			}).Return(&backupv1.BackupSchedule{}, nil).Once()

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Eventf(&scheduleOldImage, corev1.EventTypeNormal, imageUpdateEventReason,
			"Updated kubectl image in CronJob to %s.", kubectlImage)

		sut := NewUpdater(clientMock, testNamespace, kubectlImage, recorderMock)

		// when
		err := sut.Update(testCtx)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to update additional images in cron job")
		assert.ErrorContains(t, err, "failed to update kubectl image in cron job")
	})

	t.Run("should fail on updating the status of backup schedule resource", func(t *testing.T) {
		// given
		backupScheduleClientMock := newMockBackupScheduleClient(t)
		v1Alpha1Mock := newMockEcosystemV1Alpha1Interface(t)
		v1Alpha1Mock.EXPECT().BackupSchedules(testNamespace).Return(backupScheduleClientMock)
		clientMock := newMockEcosystemClientSet(t)
		clientMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Mock)

		scheduleGetError := backupv1.BackupSchedule{
			ObjectMeta: metav1.ObjectMeta{
				Name: "GetError",
			},
			Status: backupv1.BackupScheduleStatus{
				CurrentKubectlImage: "bitnami/kubectl:1.1.0",
			},
		}
		scheduleOldImage := backupv1.BackupSchedule{
			ObjectMeta: metav1.ObjectMeta{
				Name: "OldImage",
			},
			Status: backupv1.BackupScheduleStatus{
				CurrentKubectlImage: "bitnami/kubectl:1.1.0",
			},
		}
		schedules := []backupv1.BackupSchedule{
			scheduleOldImage,
			scheduleGetError,
		}
		scheduleList := backupv1.BackupScheduleList{
			Items: schedules,
		}
		backupScheduleClientMock.EXPECT().List(testCtx, metav1.ListOptions{}).Return(&scheduleList, nil)

		batchV1Mock := newMockBatchV1Interface(t)
		cronJobMock := newMockCronJobInterface(t)
		batchV1Mock.EXPECT().CronJobs(testNamespace).Return(cronJobMock)
		clientMock.EXPECT().BatchV1().Return(batchV1Mock)

		notFound := errors.NewNotFound(schema.GroupResource{}, scheduleOldImage.CronJobName())
		cronJobMock.EXPECT().Get(testCtx, mock.Anything, metav1.GetOptions{}).Return(nil, notFound).Times(2)

		scheduleGetError.Status.CurrentKubectlImage = kubectlImage
		backupScheduleClientMock.EXPECT().UpdateStatus(testCtx, &scheduleGetError, metav1.UpdateOptions{}).Return(nil, assert.AnError).Once()
		backupScheduleClientMock.EXPECT().UpdateStatus(testCtx, mock.Anything, metav1.UpdateOptions{}).
			Run(func(ctx context.Context, backupSchedule *backupv1.BackupSchedule, opts metav1.UpdateOptions) {
				assert.Equal(t, scheduleOldImage.Name, backupSchedule.Name)
				assert.Equal(t, kubectlImage, backupSchedule.Status.CurrentKubectlImage)
			}).Return(&backupv1.BackupSchedule{}, nil).Once()

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(mock.Anything, mock.Anything, mock.Anything, mock.Anything)

		sut := NewUpdater(clientMock, testNamespace, kubectlImage, recorderMock)

		// when
		err := sut.Update(testCtx)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to update current kubectl image in status of backup schedule")
	})
}
