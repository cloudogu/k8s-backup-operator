//go:build acceptance

package specs

import (
	"fmt"
	"time"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	schedulecontroller "github.com/cloudogu/k8s-backup-operator/internal/controller/schedule"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	batchv1 "k8s.io/api/batch/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	backupScheduleTestNamespace = "ecosystem"
	initialSchedule             = "0 2 * * *"
	updatedSchedule             = "15 3 * * *"
)

var _ = Describe("BackupSchedule", Label("backupschedule"), func() {
	Describe("Creating a BackupSchedule", Ordered, func() {
		backupScheduleObjectKey := newBackupScheduleObjectKey()

		AfterAll(func(ctx SpecContext) {
			deleteAndIgnoreNotFound(ctx, newBackupSchedule(backupScheduleObjectKey, initialSchedule))
		})

		It("when a BackupSchedule is created", func(ctx SpecContext) {
			backupSchedule := newBackupSchedule(backupScheduleObjectKey, initialSchedule)
			Expect(k8sClient.Create(ctx, backupSchedule)).Should(Succeed())
		})

		It("its CronJob is also created and ready", func(ctx SpecContext) {
			expectCronJobReady(ctx, backupScheduleObjectKey, initialSchedule)
		})
	})

	Describe("Updating a BackupSchedule", Ordered, func() {
		backupScheduleObjectKey := newBackupScheduleObjectKey()

		BeforeAll(func(ctx SpecContext) {
			By("creating a BackupSchedule and waiting for its CronJob")
			backupSchedule := newBackupSchedule(backupScheduleObjectKey, initialSchedule)
			Expect(k8sClient.Create(ctx, backupSchedule)).Should(Succeed())
			expectCronJobReady(ctx, backupScheduleObjectKey, initialSchedule)
		})
		AfterAll(func(ctx SpecContext) {
			deleteAndIgnoreNotFound(ctx, newBackupSchedule(backupScheduleObjectKey, initialSchedule))
		})

		It("when its schedule is updated", func(ctx SpecContext) {
			persistedBackupSchedule := &backupv1.BackupSchedule{}
			Expect(k8sClient.Get(ctx, backupScheduleObjectKey, persistedBackupSchedule)).Should(Succeed())
			persistedBackupSchedule.Spec.Schedule = updatedSchedule
			Expect(k8sClient.Update(ctx, persistedBackupSchedule)).Should(Succeed())
		})

		It("its CronJob schedule is also updated", func(ctx SpecContext) {
			expectCronJobReady(ctx, backupScheduleObjectKey, updatedSchedule)
		})
	})

	Describe("Deleting a BackupSchedule", Ordered, func() {
		backupScheduleObjectKey := newBackupScheduleObjectKey()

		BeforeAll(func(ctx SpecContext) {
			By("creating a BackupSchedule and waiting for its CronJob")
			backupSchedule := newBackupSchedule(backupScheduleObjectKey, initialSchedule)
			Expect(k8sClient.Create(ctx, backupSchedule)).Should(Succeed())
			expectCronJobReady(ctx, backupScheduleObjectKey, initialSchedule)
		})
		AfterAll(func(ctx SpecContext) {
			deleteAndIgnoreNotFound(ctx, newBackupSchedule(backupScheduleObjectKey, initialSchedule))
		})

		It("when the BackupSchedule is deleted", func(ctx SpecContext) {
			backupSchedule := newBackupSchedule(backupScheduleObjectKey, initialSchedule)
			Expect(k8sClient.Delete(ctx, backupSchedule)).Should(Succeed())
		})

		It("its CronJob is also deleted", func(ctx SpecContext) {
			Eventually(func(g Gomega) {
				cronJob := &batchv1.CronJob{}
				err := k8sClient.Get(ctx, cronJobObjectKey(backupScheduleObjectKey), cronJob)
				g.Expect(apierrors.IsNotFound(err)).To(BeTrue())
			}).
				WithTimeout(2 * time.Minute).
				WithPolling(time.Second).
				Should(Succeed())
		})

		It("the BackupSchedule is finalized and removed", func(ctx SpecContext) {
			Eventually(func(g Gomega) {
				persistedBackupSchedule := &backupv1.BackupSchedule{}
				err := k8sClient.Get(ctx, backupScheduleObjectKey, persistedBackupSchedule)
				g.Expect(apierrors.IsNotFound(err)).To(BeTrue())
			}).
				WithTimeout(2 * time.Minute).
				WithPolling(time.Second).
				Should(Succeed())
		})
	})
})

func expectCronJobReady(ctx SpecContext, backupScheduleObjectKey client.ObjectKey, expectedSchedule string) {
	Eventually(func(g Gomega) {
		cronJob := &batchv1.CronJob{}
		g.Expect(k8sClient.Get(ctx, cronJobObjectKey(backupScheduleObjectKey), cronJob)).Should(Succeed())
		g.Expect(cronJob.Spec.Schedule).Should(Equal(expectedSchedule))

		backupSchedule := &backupv1.BackupSchedule{}
		g.Expect(k8sClient.Get(ctx, backupScheduleObjectKey, backupSchedule)).Should(Succeed())
		readyCondition := meta.FindStatusCondition(backupSchedule.Status.Conditions, schedulecontroller.ReadyCondition)
		g.Expect(readyCondition).ShouldNot(BeNil())
		g.Expect(readyCondition.Status).Should(Equal(metav1.ConditionTrue))
		g.Expect(readyCondition.Reason).Should(Equal(schedulecontroller.ReasonReady))
		g.Expect(backupSchedule.Status.Conditions).Should(HaveLen(2))
	}).
		WithTimeout(2 * time.Minute).
		WithPolling(time.Second).
		Should(Succeed())
}

func newBackupScheduleObjectKey() client.ObjectKey {
	return client.ObjectKey{
		Namespace: backupScheduleTestNamespace,
		Name:      fmt.Sprintf("bs-spec-%s", uuid.New().String()[:8]),
	}
}

func newBackupSchedule(objectKey client.ObjectKey, cronSchedule string) *backupv1.BackupSchedule {
	return &backupv1.BackupSchedule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      objectKey.Name,
			Namespace: objectKey.Namespace,
		},
		Spec: backupv1.BackupScheduleSpec{
			Schedule: cronSchedule,
			Provider: backupv1.ProviderVelero,
		},
	}
}

func cronJobObjectKey(backupScheduleObjectKey client.ObjectKey) client.ObjectKey {
	backupSchedule := newBackupSchedule(backupScheduleObjectKey, "")
	return client.ObjectKey{
		Name:      backupSchedule.CronJobName(),
		Namespace: backupScheduleObjectKey.Namespace,
	}
}
