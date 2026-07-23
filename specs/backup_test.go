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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Backup", func() {
	Describe("Creating a Backup", Ordered, func() {
		var backupObjectKey = client.ObjectKey{Namespace: "ecosystem", Name: fmt.Sprintf("backup-spec-%s", uuid.New().String())}

		It("When a backup is created", func(ctx SpecContext) {
			backupCr := createBackupWithObjectKey(backupObjectKey)
			err := k8sClient.Create(ctx, backupCr, &client.CreateOptions{})
			Expect(err).ShouldNot(HaveOccurred())
		})

		It("the provider's backup is also created", func(ctx SpecContext) {
			Eventually(func(g Gomega) {
				veleroBackup := &velerov1.Backup{}
				err := k8sClient.Get(ctx, backupObjectKey, veleroBackup)
				g.Expect(err).ShouldNot(HaveOccurred())
			}).
				WithTimeout(10 * time.Minute).
				WithPolling(5 * time.Second).
				Should(Succeed())
		})
	})

	Describe("Deleting a backup", Ordered, func() {
		var backupObjectKey = client.ObjectKey{Namespace: "ecosystem", Name: fmt.Sprintf("backup-spec-%s", uuid.New().String())}

		BeforeAll(func(ctx SpecContext) {
			backupCr := createBackupWithObjectKey(backupObjectKey)
			err := k8sClient.Create(ctx, backupCr, &client.CreateOptions{})
			Expect(err).ShouldNot(HaveOccurred())
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
