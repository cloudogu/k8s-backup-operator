//go:build acceptance

package specs

import (
	"context"
	"fmt"
	"strconv"
	"time"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Backup", Label("backup"), Ordered, func() {
	Describe("Creating a backup", Ordered, func() {
		var backupObjectKey = client.ObjectKey{
			Namespace: "ecosystem",
			Name:      fmt.Sprintf("backup-spec-creating-backup%s", uuid.New().String()),
		}

		AfterAll(func(ctx SpecContext) {
			By("deletes the backup resource")
			backup := createBackupWithObjectKey(backupObjectKey)
			err := k8sClient.Delete(ctx, backup, &client.DeleteOptions{})
			Expect(err).ShouldNot(HaveOccurred())

			By("waits until the backup is deleted")
			EventuallyShouldSucceed(func(g Gomega) {
				backup := &backupv1.Backup{}
				err := k8sClient.Get(ctx, backupObjectKey, backup)
				g.Expect(apierrors.IsNotFound(err)).To(BeTrue())
			})
		})

		It("creates the backup resource", func(ctx SpecContext) {
			backup := createBackupWithObjectKey(backupObjectKey)
			err := k8sClient.Create(ctx, backup, &client.CreateOptions{})
			Expect(err).ShouldNot(HaveOccurred())
		})

		It("checks whether the provider's backup resource was created", func(ctx SpecContext) {
			EventuallyShouldSucceed(func(g Gomega) {
				veleroBackup := &velerov1.Backup{}
				err := k8sClient.Get(ctx, backupObjectKey, veleroBackup)
				g.Expect(err).ShouldNot(HaveOccurred())
			})
		})

		It("waits until the backup has successfully completed", func(ctx SpecContext) {
			EventuallyShouldSucceed(func(g Gomega) {
				backup := &backupv1.Backup{}
				err := k8sClient.Get(ctx, backupObjectKey, backup)
				Expect(err).ShouldNot(HaveOccurred())

				completed := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionCompleted)
				g.Expect(completed.Status).To(Equal(metav1.ConditionTrue))
			})
		})

	})

	Describe("Deleting a backup", Ordered, Label("backup"), func() {
		var backupObjectKey = client.ObjectKey{
			Namespace: "ecosystem",
			Name:      fmt.Sprintf("backup-spec-deleting-backup-%s", uuid.New().String()),
		}

		It("creates the backup resource", func(ctx SpecContext) {
			backup := createBackupWithObjectKey(backupObjectKey)
			err := k8sClient.Create(ctx, backup, &client.CreateOptions{})
			Expect(err).ShouldNot(HaveOccurred())
		})

		It("waits until the backup has successfully completed", func(ctx SpecContext) {
			EventuallyShouldSucceed(func(g Gomega) {
				backup := &backupv1.Backup{}
				err := k8sClient.Get(ctx, backupObjectKey, backup)
				g.Expect(err).ShouldNot(HaveOccurred())

				completed := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionCompleted)
				g.Expect(completed).ToNot(BeNil())
				g.Expect(completed.Status).To(Equal(metav1.ConditionTrue))
			})
		})

		It("deletes the backup", func(ctx SpecContext) {
			backup := createBackupWithObjectKey(backupObjectKey)
			err := k8sClient.Delete(ctx, backup, &client.DeleteOptions{})
			Expect(err).ShouldNot(HaveOccurred())
		})

		It("waits until the backup is deleted", func(ctx SpecContext) {
			EventuallyShouldSucceed(func(g Gomega) {
				backup := &backupv1.Backup{}
				err := k8sClient.Get(ctx, backupObjectKey, backup)
				g.Expect(apierrors.IsNotFound(err)).To(BeTrue())
			})
		})

		It("checks if the provider's backup is also deleted", func(ctx SpecContext) {
			veleroBackup := &velerov1.Backup{}
			err := k8sClient.Get(ctx, backupObjectKey, veleroBackup)
			Expect(apierrors.IsNotFound(err)).To(BeTrue())
		})
	})

	Describe("Canceling a backup", Ordered, Label("backup"), func() {
		var backupObjectKey = client.ObjectKey{
			Namespace: "ecosystem",
			Name:      fmt.Sprintf("backup-spec-canceling-backup%s", uuid.New().String()),
		}
		var veleroBackupStoreLocationObjectKey = client.ObjectKey{
			Namespace: "ecosystem",
			Name:      "default",
		}
		var veleroBackupStoreLocationS3Region = ""
		var backupTimeLimitInMinutesForTest = 1

		AfterAll(func(ctx SpecContext) {
			By("deletes the backup resource")
			backup := createBackupWithObjectKey(backupObjectKey)
			err := k8sClient.Delete(ctx, backup, &client.DeleteOptions{})
			Expect(err).ShouldNot(HaveOccurred())

			By("waits until the backup is deleted")
			EventuallyShouldSucceed(func(g Gomega) {
				backup := &backupv1.Backup{}
				err := k8sClient.Get(ctx, backupObjectKey, backup)
				g.Expect(apierrors.IsNotFound(err)).To(BeTrue())
			})

			By("resets the backup time limit to the default value")
			err = configureBackupTimeLimit(ctx, 60)
			Expect(err).ShouldNot(HaveOccurred())
		})

		It("configures the backup time limit", func(ctx SpecContext) {
			err := configureBackupTimeLimit(ctx, backupTimeLimitInMinutesForTest)
			Expect(err).ShouldNot(HaveOccurred())
		})

		It("ensures the provider backup storage is unavailable", func(ctx SpecContext) {
			veleroBackupStorageLocation := &velerov1.BackupStorageLocation{}
			err := k8sClient.Get(ctx, veleroBackupStoreLocationObjectKey, veleroBackupStorageLocation)
			Expect(err).ShouldNot(HaveOccurred())

			veleroBackupStoreLocationS3Region = veleroBackupStorageLocation.Spec.Config["region"]
			veleroBackupStorageLocation.Spec.Config["region"] = "region_that_not_exist"
			err = k8sClient.Update(ctx, veleroBackupStorageLocation)
			Expect(err).ShouldNot(HaveOccurred())

			// We are waiting for the velero reconciler to update the status of the backup storage location
			Eventually(func(g Gomega) {
				veleroBackupStorageLocation := &velerov1.BackupStorageLocation{}
				err := k8sClient.Get(ctx, veleroBackupStoreLocationObjectKey, veleroBackupStorageLocation)
				g.Expect(err).ShouldNot(HaveOccurred())
				g.Expect(veleroBackupStorageLocation.Status.Phase).To(Equal(velerov1.BackupStorageLocationPhaseUnavailable))
			}).
				WithTimeout(1 * time.Minute).
				WithPolling(10 * time.Second).
				Should(Succeed())
		})

		It("creates the backup resource", func(ctx SpecContext) {
			backup := createBackupWithObjectKey(backupObjectKey)
			err := k8sClient.Create(ctx, backup, &client.CreateOptions{})
			Expect(err).ShouldNot(HaveOccurred())
		})

		It("checks whether the backup preparation does not meet the prepared condition", func(ctx SpecContext) {
			EventuallyShouldSucceed(func(g Gomega) {
				backup := &backupv1.Backup{}
				err := k8sClient.Get(ctx, backupObjectKey, backup)
				g.Expect(err).ShouldNot(HaveOccurred())

				completed := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionPrepared)
				g.Expect(completed).ToNot(BeNil())
				g.Expect(completed.Status).To(Equal(metav1.ConditionFalse))
			})
		})

		It("ensures the provider backup storage is available after time window expired", func(ctx SpecContext) {
			time.Sleep(time.Duration(backupTimeLimitInMinutesForTest)*time.Minute + 30*time.Second)

			veleroBackupStorageLocation := &velerov1.BackupStorageLocation{}
			err := k8sClient.Get(ctx, veleroBackupStoreLocationObjectKey, veleroBackupStorageLocation)
			Expect(err).ShouldNot(HaveOccurred())

			veleroBackupStorageLocation.Spec.Config["region"] = veleroBackupStoreLocationS3Region
			err = k8sClient.Update(ctx, veleroBackupStorageLocation)
			Expect(err).ShouldNot(HaveOccurred())
		})

		It("checks whether the backup was cancelled and did not start", func(ctx SpecContext) {
			EventuallyShouldSucceed(func(g Gomega) {
				backup := &backupv1.Backup{}
				err := k8sClient.Get(ctx, backupObjectKey, backup)
				g.Expect(err).ShouldNot(HaveOccurred())

				completed := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionCanceled)
				g.Expect(completed).ToNot(BeNil())
				g.Expect(completed.Status).To(Equal(metav1.ConditionTrue))

				g.Expect(backup.Status.StartTimestamp.IsZero()).To(BeTrue())
			})
		})
	})

	FDescribe("Canceling a running backup", Ordered, Label("backup"), func() {
		var backupObjectKey = client.ObjectKey{
			Namespace: "ecosystem",
			Name:      fmt.Sprintf("backup-spec-canceling-backup%s", uuid.New().String()),
		}
		var backupTimeLimitInMinutesForTest = 1

		AfterAll(func(ctx SpecContext) {
			By("deletes the backup resource")
			backup := createBackupWithObjectKey(backupObjectKey)
			err := k8sClient.Delete(ctx, backup, &client.DeleteOptions{})
			Expect(err).ShouldNot(HaveOccurred())

			By("waits until the backup is deleted")
			EventuallyShouldSucceed(func(g Gomega) {
				backup := &backupv1.Backup{}
				err := k8sClient.Get(ctx, backupObjectKey, backup)
				g.Expect(apierrors.IsNotFound(err)).To(BeTrue())
			})

			By("resets the backup time limit to the default value")
			err = configureBackupTimeLimit(ctx, 60)
			Expect(err).ShouldNot(HaveOccurred())
		})

		It("configures the backup time limit", func(ctx SpecContext) {
			err := configureBackupTimeLimit(ctx, backupTimeLimitInMinutesForTest)
			Expect(err).ShouldNot(HaveOccurred())
		})

		It("creates the backup resource", func(ctx SpecContext) {
			backup := createBackupWithObjectKey(backupObjectKey)
			err := k8sClient.Create(ctx, backup, &client.CreateOptions{})
			Expect(err).ShouldNot(HaveOccurred())
		})

		// This test expects a backup with a duration that exceeds the elapsed time window.
		It("checks whether the backup is still running after time window expired", func(ctx SpecContext) {
			time.Sleep(time.Duration(backupTimeLimitInMinutesForTest)*time.Minute + 10*time.Second)

			EventuallyShouldSucceed(func(g Gomega) {
				backup := &backupv1.Backup{}
				err := k8sClient.Get(ctx, backupObjectKey, backup)
				g.Expect(err).ShouldNot(HaveOccurred())

				completed := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionCanceled)
				g.Expect(completed).ToNot(BeNil())
				g.Expect(completed.Status).To(Equal(metav1.ConditionFalse))
				g.Expect(completed.Reason).To(Equal("TimeWindowExpiredBackupIsRunning"))

				g.Expect(backup.Status.StartTimestamp.IsZero()).To(BeFalse())
			})
		})

		It("waits until the backup has successfully completed", func(ctx SpecContext) {
			EventuallyShouldSucceed(func(g Gomega) {
				backup := &backupv1.Backup{}
				err := k8sClient.Get(ctx, backupObjectKey, backup)
				Expect(err).ShouldNot(HaveOccurred())

				completed := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionCompleted)
				g.Expect(completed).ToNot(BeNil())
				g.Expect(completed.Status).To(Equal(metav1.ConditionTrue))
			})
		})

	})
})

func EventuallyShouldSucceed(fn func(g Gomega)) bool {
	return Eventually(fn).
		WithTimeout(5 * time.Minute).
		WithPolling(5 * time.Second).
		Should(Succeed())
}

func createBackupWithObjectKey(objectKey client.ObjectKey) *backupv1.Backup {
	return &backupv1.Backup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      objectKey.Name,
			Namespace: objectKey.Namespace,
		},
		Spec: backupv1.BackupSpec{
			Provider: "velero",
		},
	}
}

func configureBackupTimeLimit(ctx context.Context, retryLimitInMinutes int) error {
	objectKey := client.ObjectKey{Namespace: "ecosystem", Name: "k8s-backup-operator-backup-config"}

	backupOperatorConfigMap := &corev1.ConfigMap{}
	err := k8sClient.Get(ctx, objectKey, backupOperatorConfigMap)
	if err != nil {
		return err
	}

	backupOperatorConfigMap.Data["retryTimeLimit"] = strconv.Itoa(retryLimitInMinutes) // minutes
	err = k8sClient.Update(ctx, backupOperatorConfigMap)
	if err != nil {
		return err
	}

	return nil
}
