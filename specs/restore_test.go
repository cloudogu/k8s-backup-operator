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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// scaledownScopeLabel marks workloads that the restore flow scales to zero
	// and back up again. See pkg/scale.
	scaledownScopeLabel = "k8s.cloudogu.com/restore-scaledown-scope"
	// scaledownReplicasLabel holds the replica count captured at scale-down
	// time. A leftover value means a previous restore did not finish.
	scaledownReplicasLabel = "k8s.cloudogu.com/restore-scaledown-replicas"
	// backupScopeLabel marks additional resources that the restore flow deletes
	// before restoring. See pkg/cleanup.
	backupScopeLabel = "k8s.cloudogu.com/backup-scope"

	restoreTestNamespace = "ecosystem"
	throwawayReplicas    = int32(2)
)

// Restore is a namespace-wide destructive operation: it scales down every
// workload labeled with the scaledown scope, deletes *all* dogus, and deletes
// every ConfigMap, Secret and PVC labeled with the backup scope. It therefore
// must never run concurrently with the Backup specs or with itself, hence
// Serial. Run this suite only against a disposable cluster.
var _ = Describe("Restore", Serial, Ordered, func() {
	suffix := uuid.New().String()
	backupKey := client.ObjectKey{Namespace: restoreTestNamespace, Name: fmt.Sprintf("restore-spec-backup-%s", suffix)}
	restoreKey := client.ObjectKey{Namespace: restoreTestNamespace, Name: fmt.Sprintf("restore-spec-%s", suffix)}
	deploymentKey := client.ObjectKey{Namespace: restoreTestNamespace, Name: fmt.Sprintf("restore-spec-workload-%s", suffix)}
	configMapKey := client.ObjectKey{Namespace: restoreTestNamespace, Name: fmt.Sprintf("restore-spec-resource-%s", suffix)}

	BeforeAll(func(ctx SpecContext) {
		expectNoLeftoverScaledownLabels(ctx)

		By("creating a throwaway workload labeled for restore scaledown")
		deployment := newThrowawayDeployment(deploymentKey, throwawayReplicas)
		Expect(k8sClient.Create(ctx, deployment)).Should(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			deleteAndIgnoreNotFound(ctx, newThrowawayDeployment(deploymentKey, throwawayReplicas))
		})

		// This backup is deliberately NOT deleted afterwards. Deleting a Cloudogu
		// Backup issues a Velero DeleteBackupRequest, and Velero deletes the
		// backup's volume snapshots along with it. A restore keeps consuming those
		// snapshots after it reports completed, because the CSI provisioner
		// materializes volumes asynchronously. This prevents missing snapshots after the restore.
		By("creating a backup for the restore to reference")
		backup := createBackupWithObjectKey(backupKey)
		Expect(k8sClient.Create(ctx, backup)).Should(Succeed())
		expectBackupStatus(ctx, backupKey, backupv1.BackupStatusCompleted)

		// Created only after the backup completed, so it is deliberately absent
		// from the backup. Cleanup must delete it and nothing must bring it
		// back, which makes the assertion unambiguous.
		By("creating a throwaway additional resource labeled for backup scope")
		configMap := newThrowawayConfigMap(configMapKey)
		Expect(k8sClient.Create(ctx, configMap)).Should(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			deleteAndIgnoreNotFound(ctx, newThrowawayConfigMap(configMapKey))
		})
	})

	Describe("Creating a Restore", Ordered, func() {
		It("when a restore is created", func(ctx SpecContext) {
			restore := newRestoreWithObjectKey(restoreKey, backupKey.Name)
			Expect(k8sClient.Create(ctx, restore)).Should(Succeed())
		})

		It("the provider's restore is also created", func(ctx SpecContext) {
			Eventually(func(g Gomega) {
				veleroRestore := &velerov1.Restore{}
				g.Expect(k8sClient.Get(ctx, restoreKey, veleroRestore)).Should(Succeed())
			}).
				WithTimeout(10 * time.Minute).
				WithPolling(5 * time.Second).
				Should(Succeed())
		})

		It("the restore reaches the completed status", func(ctx SpecContext) {
			expectRestoreStatus(ctx, restoreKey, backupv1.RestoreStatusCompleted)
		})

		It("the labeled additional resource was deleted by the cleanup", func(ctx SpecContext) {
			configMap := &corev1.ConfigMap{}
			err := k8sClient.Get(ctx, configMapKey, configMap)
			Expect(apierrors.IsNotFound(err)).To(BeTrue(),
				"ConfigMap %s carries the backup scope label and was not part of the backup, "+
					"so the restore cleanup must have deleted it permanently", configMapKey.Name)
		})

		It("the scaled-down workload is back at its original replica count", func(ctx SpecContext) {
			Eventually(func(g Gomega) {
				deployment := &appsv1.Deployment{}
				g.Expect(k8sClient.Get(ctx, deploymentKey, deployment)).Should(Succeed())
				g.Expect(deployment.Spec.Replicas).ShouldNot(BeNil())
				g.Expect(*deployment.Spec.Replicas).Should(Equal(throwawayReplicas),
					"the restore must scale the workload back to the replica count it captured, not to zero or one")
				g.Expect(deployment.Labels).ShouldNot(HaveKey(scaledownReplicasLabel),
					"the scaledown bookkeeping label must be removed once the workload is scaled up again")
			}).
				WithTimeout(10 * time.Minute).
				WithPolling(5 * time.Second).
				Should(Succeed())
		})

		It("every restored volume is bound and every workload becomes ready", func(ctx SpecContext) {
			expectWorkloadsConverged(ctx)
		})

		It("reconciling the completed restore again changes nothing", func(ctx SpecContext) {
			By("forcing a new reconcile of the already completed restore")
			restore := &backupv1.Restore{}
			Expect(k8sClient.Get(ctx, restoreKey, restore)).Should(Succeed())
			patch := client.MergeFrom(restore.DeepCopy())
			if restore.Annotations == nil {
				restore.Annotations = map[string]string{}
			}
			restore.Annotations["k8s.cloudogu.com/restore-spec-touch"] = suffix
			Expect(k8sClient.Patch(ctx, restore, patch)).Should(Succeed())

			By("verifying the restore stays completed and the workload stays scaled up")
			Consistently(func(g Gomega) {
				persisted := &backupv1.Restore{}
				g.Expect(k8sClient.Get(ctx, restoreKey, persisted)).Should(Succeed())
				g.Expect(persisted.Status.Status).Should(Equal(backupv1.RestoreStatusCompleted))

				deployment := &appsv1.Deployment{}
				g.Expect(k8sClient.Get(ctx, deploymentKey, deployment)).Should(Succeed())
				g.Expect(deployment.Spec.Replicas).ShouldNot(BeNil())
				g.Expect(*deployment.Spec.Replicas).Should(Equal(throwawayReplicas))
			}).
				WithTimeout(30 * time.Second).
				WithPolling(5 * time.Second).
				Should(Succeed())
		})
	})

	Describe("Deleting a restore", Ordered, func() {
		It("if the restore is deleted", func(ctx SpecContext) {
			restore := newRestoreWithObjectKey(restoreKey, backupKey.Name)
			Expect(k8sClient.Delete(ctx, restore)).Should(Succeed())
		})

		It("the provider's restore is also deleted", func(ctx SpecContext) {
			Eventually(func(g Gomega) {
				veleroRestore := &velerov1.Restore{}
				err := k8sClient.Get(ctx, restoreKey, veleroRestore)
				g.Expect(apierrors.IsNotFound(err)).To(BeTrue())
			}).
				WithTimeout(10 * time.Minute).
				WithPolling(10 * time.Second).
				Should(Succeed())
		})
	})
})

// expectNoLeftoverScaledownLabels fails fast when a previous restore left
// backup labels behind. Scale-down skips workloads that already carry the
// replicas label and scale-up would then write back a stale count, so running
// on top of dirty state produces misleading failures.
func expectNoLeftoverScaledownLabels(ctx SpecContext) {
	deployments := &appsv1.DeploymentList{}
	Expect(k8sClient.List(ctx, deployments, client.InNamespace(restoreTestNamespace))).Should(Succeed())

	var dirty []string
	for _, deployment := range deployments.Items {
		if _, exists := deployment.Labels[scaledownReplicasLabel]; exists {
			dirty = append(dirty, deployment.Name)
		}
	}
	Expect(dirty).Should(BeEmpty(),
		"workloads still carry %s from an unfinished restore; clean the cluster before running these specs",
		scaledownReplicasLabel)
}

// expectWorkloadsConverged waits until the namespace is actually usable again.
//
// A Restore reports completed as soon as Velero finishes creating API objects.
// With CSI snapshots the PVCs are created referencing a VolumeSnapshot and the
// provisioner materializes the volumes afterwards, so completed does not mean
// the ecosystem is back. Without this check the suite would call a restore
// successful even if every volume failed to attach.
//
// The timeout is generous because this gates on real volume provisioning and
// container startup rather than on API round trips.
func expectWorkloadsConverged(ctx SpecContext) {
	Eventually(func(g Gomega) {
		claims := &corev1.PersistentVolumeClaimList{}
		g.Expect(k8sClient.List(ctx, claims, client.InNamespace(restoreTestNamespace))).Should(Succeed())
		for _, claim := range claims.Items {
			g.Expect(claim.Status.Phase).Should(Equal(corev1.ClaimBound),
				"PersistentVolumeClaim %s is %s; a restored volume that never binds usually means its "+
					"snapshot is gone", claim.Name, claim.Status.Phase)
		}

		deployments := &appsv1.DeploymentList{}
		g.Expect(k8sClient.List(ctx, deployments, client.InNamespace(restoreTestNamespace))).Should(Succeed())
		for _, deployment := range deployments.Items {
			desired := int32(1)
			if deployment.Spec.Replicas != nil {
				desired = *deployment.Spec.Replicas
			}
			g.Expect(deployment.Status.ReadyReplicas).Should(Equal(desired),
				"Deployment %s has %d/%d ready replicas", deployment.Name,
				deployment.Status.ReadyReplicas, desired)
		}

		statefulSets := &appsv1.StatefulSetList{}
		g.Expect(k8sClient.List(ctx, statefulSets, client.InNamespace(restoreTestNamespace))).Should(Succeed())
		for _, statefulSet := range statefulSets.Items {
			desired := int32(1)
			if statefulSet.Spec.Replicas != nil {
				desired = *statefulSet.Spec.Replicas
			}
			g.Expect(statefulSet.Status.ReadyReplicas).Should(Equal(desired),
				"StatefulSet %s has %d/%d ready replicas", statefulSet.Name,
				statefulSet.Status.ReadyReplicas, desired)
		}
	}).
		WithTimeout(15 * time.Minute).
		WithPolling(10 * time.Second).
		Should(Succeed())
}

// expectBackupStatus waits for the backup to reach status.
//
// A failed assertion inside Eventually only means "not yet", so a backup that
// has already failed would otherwise be polled until the timeout expires. The
// failed status is terminal, so abort the retry loop with StopTrying instead of
// waiting ten minutes for a state that can no longer change.
func expectBackupStatus(ctx SpecContext, key client.ObjectKey, status string) {
	Eventually(func(g Gomega) {
		backup := &backupv1.Backup{}
		g.Expect(k8sClient.Get(ctx, key, backup)).Should(Succeed())

		if status != backupv1.BackupStatusFailed && backup.Status.Status == backupv1.BackupStatusFailed {
			StopTrying(fmt.Sprintf("backup %s reached the terminal status %q while waiting for %q",
				key.Name, backupv1.BackupStatusFailed, status)).
				Attach("backup status", backup.Status).
				Now()
		}

		g.Expect(backup.Status.Status).Should(Equal(status))
	}).
		WithTimeout(10 * time.Minute).
		WithPolling(5 * time.Second).
		Should(Succeed())
}

// expectRestoreStatus waits for the restore to reach status. See
// expectBackupStatus for why the terminal failed status aborts the retry loop.
func expectRestoreStatus(ctx SpecContext, key client.ObjectKey, status string) {
	Eventually(func(g Gomega) {
		restore := &backupv1.Restore{}
		g.Expect(k8sClient.Get(ctx, key, restore)).Should(Succeed())

		if status != backupv1.RestoreStatusFailed && restore.Status.Status == backupv1.RestoreStatusFailed {
			StopTrying(fmt.Sprintf("restore %s reached the terminal status %q while waiting for %q",
				key.Name, backupv1.RestoreStatusFailed, status)).
				Attach("restore status", restore.Status).
				Now()
		}

		g.Expect(restore.Status.Status).Should(Equal(status))
	}).
		WithTimeout(10 * time.Minute).
		WithPolling(5 * time.Second).
		Should(Succeed())
}

func deleteAndIgnoreNotFound(ctx SpecContext, object client.Object) {
	err := k8sClient.Delete(ctx, object)
	if err != nil && !apierrors.IsNotFound(err) {
		Expect(err).ShouldNot(HaveOccurred())
	}
}

func newRestoreWithObjectKey(objectKey client.ObjectKey, backupName string) *backupv1.Restore {
	return &backupv1.Restore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      objectKey.Name,
			Namespace: objectKey.Namespace,
		},
		Spec: backupv1.RestoreSpec{
			Provider:   backupv1.ProviderVelero,
			BackupName: backupName,
		},
	}
}

// newThrowawayDeployment builds a workload that only carries the scaledown
// scope label to test the scale down and correct scale up again.
func newThrowawayDeployment(objectKey client.ObjectKey, replicas int32) *appsv1.Deployment {
	selector := map[string]string{"app": objectKey.Name}

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      objectKey.Name,
			Namespace: objectKey.Namespace,
			Labels:    map[string]string{scaledownScopeLabel: "restore"},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: selector},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: selector},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:    "pause",
						Image:   "registry.k8s.io/pause:3.9",
						Command: nil,
					}},
				},
			},
		},
	}
}

// newThrowawayConfigMap builds an additional resource that the restore cleanup
// must delete because of its backup scope label.
func newThrowawayConfigMap(objectKey client.ObjectKey) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      objectKey.Name,
			Namespace: objectKey.Namespace,
			Labels:    map[string]string{backupScopeLabel: "restore"},
		},
		Data: map[string]string{"spec": "throwaway"},
	}
}
