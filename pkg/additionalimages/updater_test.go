package additionalimages

import (
	context "context"
	backupv1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func Test_updater_Update(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		backupScheduleClientMock := newMockBackupScheduleClient(t)
		v1Alpha1Mock := newMockEcosystemV1Alpha1Interface(t)
		v1Alpha1Mock.EXPECT().BackupSchedules(testNamespace).Return(backupScheduleClientMock)
		clientMock := newMockEcosystemClientSet(t)
		clientMock.EXPECT().EcosystemV1Alpha1().Return(v1Alpha1Mock)
		kubectlImage := "bitnami/kubectl:1.1.1"
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

		cronJobOldImage := &batchv1.CronJob{
			Spec: batchv1.CronJobSpec{JobTemplate: batchv1.JobTemplateSpec{Spec: batchv1.JobSpec{Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{Containers: []corev1.Container{{}}}}}}},
		}
		cronJobMock.EXPECT().Get(testCtx, scheduleOldImage.CronJobName(), metav1.GetOptions{}).Return(cronJobOldImage, nil)
		cronJobMock.EXPECT().Update(testCtx, cronJobOldImage, metav1.UpdateOptions{}).Return(cronJobOldImage, nil)

		cronJobNoImage := &batchv1.CronJob{
			Spec: batchv1.CronJobSpec{JobTemplate: batchv1.JobTemplateSpec{Spec: batchv1.JobSpec{Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{Containers: []corev1.Container{{}}}}}}},
		}
		cronJobMock.EXPECT().Get(testCtx, scheduleNoImage.CronJobName(), metav1.GetOptions{}).Return(cronJobNoImage, nil)
		cronJobMock.EXPECT().Update(testCtx, cronJobOldImage, metav1.UpdateOptions{}).Return(cronJobNoImage, nil)
		backupScheduleClientMock.EXPECT().UpdateStatus(testCtx, mock.Anything, metav1.UpdateOptions{}).
			Run(func(ctx context.Context, backupSchedule *backupv1.BackupSchedule, opts metav1.UpdateOptions) {
				assert.Equal(t, kubectlImage, backupSchedule.Status.CurrentKubectlImage)
			}).Return(&backupv1.BackupSchedule{}, nil).Times(2)

		sut := NewUpdater(clientMock, testNamespace, kubectlImage)

		// when
		err := sut.Update(testCtx)

		// then
		require.NoError(t, err)
		assert.Equal(t, kubectlImage, cronJobOldImage.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image)
		assert.Equal(t, kubectlImage, cronJobNoImage.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image)
	})
}
