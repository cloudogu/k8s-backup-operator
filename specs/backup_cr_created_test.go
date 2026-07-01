package specs

import (
	"context"
	"os"
	"os/exec"
	"path"
	"time"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("BackupReconciler", func() {
	It("should create velero backup cr", func() {
		wd, err := os.Getwd()
		Expect(err).ShouldNot(HaveOccurred())
		yamlPath := path.Join(wd, "backup01.yaml")

		home, err := os.UserHomeDir()
		Expect(err).ShouldNot(HaveOccurred())

		kubeConfPath := path.Join(home, ".kube/k3ces.localdomain")

		command := exec.Command("/usr/bin/kubectl", "--kubeconfig", kubeConfPath, "apply", "-f", yamlPath)
		session, err := Start(command, GinkgoWriter, GinkgoWriter)
		Expect(err).ShouldNot(HaveOccurred())
		Eventually(session).Should(Exit(0))
		session.Wait()

	})

	It("should use k8s ", func() {
		pods, err := k8sClientSet.CoreV1().Pods("ecosystem").List(context.TODO(), metav1.ListOptions{})
		Expect(err).ShouldNot(HaveOccurred())

		Expect(len(pods.Items)).Should(Equal(20))
	})

	It("should create velero backup CR if backup cr was created", func() {
		var testCtx = context.TODO()
		var backupObjectKey = client.ObjectKey{Namespace: "ecosystem", Name: "backup-test-101"}

		backupCr := &backupv1.Backup{
			ObjectMeta: metav1.ObjectMeta{
				Name:      backupObjectKey.Name,
				Namespace: backupObjectKey.Namespace,
			},
			Spec: backupv1.BackupSpec{
				Provider: "velero",
			},
		}

		err := k8sClient.Create(testCtx, backupCr, &client.CreateOptions{})
		Expect(err).ShouldNot(HaveOccurred())

		Eventually(func() (velerov1.BackupPhase, error) {
			veleroBackup := &velerov1.Backup{}
			err = k8sClient.Get(testCtx, backupObjectKey, veleroBackup)
			if err != client.IgnoreNotFound(err) {
				return velerov1.BackupPhaseFailed, err
			}
			return veleroBackup.Status.Phase, nil
		}).
			WithTimeout(10 * time.Minute).
			WithPolling(10 * time.Second).
			Should(Equal(velerov1.BackupPhaseCompleted))
	})
})
