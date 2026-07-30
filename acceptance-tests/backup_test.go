//go:build acceptance

package specs

import (
	"fmt"
	"time"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Backup", Label("backup"), Ordered, func() {
	FDescribe("Creating a backup", Ordered, func() {
		var backupObjectKey = client.ObjectKey{Namespace: "ecosystem", Name: fmt.Sprintf("backup-spec-%s", uuid.New().String())}

		AfterAll(func(ctx SpecContext) {
			By("deleting the backup resource")
			backupCr := createBackupWithObjectKey(backupObjectKey)
			err := k8sClient.Delete(ctx, backupCr, &client.DeleteOptions{})
			Expect(err).ShouldNot(HaveOccurred())

			By("waiting until the backup is deleted")
			Eventually(func(g Gomega) {
				backup := &backupv1.Backup{}
				err := k8sClient.Get(ctx, backupObjectKey, backup)
				g.Expect(apierrors.IsNotFound(err)).To(BeTrue())
			}).
				WithTimeout(10 * time.Minute).
				WithPolling(10 * time.Second).
				Should(Succeed())
		})

		It("creates the backup resource", func(ctx SpecContext) {
			backupCr := createBackupWithObjectKey(backupObjectKey)
			err := k8sClient.Create(ctx, backupCr, &client.CreateOptions{})
			Expect(err).ShouldNot(HaveOccurred())
		})

		It("creates the provider's backup resource", func(ctx SpecContext) {
			Eventually(func(g Gomega) {
				veleroBackup := &velerov1.Backup{}
				err := k8sClient.Get(ctx, backupObjectKey, veleroBackup)
				g.Expect(err).ShouldNot(HaveOccurred())
			}).
				WithTimeout(10 * time.Minute).
				WithPolling(5 * time.Second).
				Should(Succeed())
		})

		It("completes successfully", func(ctx SpecContext) {
			Eventually(func(g Gomega) {
				backup := &backupv1.Backup{}
				err := k8sClient.Get(ctx, backupObjectKey, backup)
				Expect(err).ShouldNot(HaveOccurred())

				completed := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionCompleted)
				g.Expect(completed.Status).To(Equal(metav1.ConditionTrue))
			}).
				WithTimeout(10 * time.Minute).
				WithPolling(10 * time.Second).
				Should(Succeed())
		})

	})

	Describe("Deleting a backup", Ordered, Label("backup"), func() {
		var backupObjectKey = client.ObjectKey{Namespace: "ecosystem", Name: fmt.Sprintf("backup-spec-%s", uuid.New().String())}

		BeforeAll(func(ctx SpecContext) {
			backup := createBackupWithObjectKey(backupObjectKey)
			err := k8sClient.Create(ctx, backup, &client.CreateOptions{})
			Expect(err).ShouldNot(HaveOccurred())

			Eventually(func(g Gomega) {
				backup := &backupv1.Backup{}
				err := k8sClient.Get(ctx, backupObjectKey, backup)
				Expect(err).ShouldNot(HaveOccurred())

				completed := meta.FindStatusCondition(backup.Status.Conditions, backupv1.ConditionCompleted)
				g.Expect(completed.Status).To(Equal(metav1.ConditionTrue))
			}).
				WithTimeout(10 * time.Minute).
				WithPolling(10 * time.Second).
				Should(Succeed())
		})

		It("if the backup is deleted", func(ctx SpecContext) {
			backupCr := createBackupWithObjectKey(backupObjectKey)
			err := k8sClient.Delete(ctx, backupCr, &client.DeleteOptions{})
			Expect(err).ShouldNot(HaveOccurred())
		})

		It("the provider's backup is also deleted", func(ctx SpecContext) {
			Eventually(func(g Gomega) {
				veleroBackup := &velerov1.Backup{}
				err := k8sClient.Get(ctx, backupObjectKey, veleroBackup)
				g.Expect(apierrors.IsNotFound(err)).To(BeTrue())
			}).
				WithTimeout(10 * time.Minute).
				WithPolling(10 * time.Second).
				Should(Succeed())
		})
	})
})

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
