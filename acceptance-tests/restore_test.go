//go:build acceptance

package specs

import (
	"fmt"
	"time"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/k8s-registry-lib/repository"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	appsv1 "k8s.io/api/apps/v1"
	coordinationv1 "k8s.io/api/coordination/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	// scaledownScopeLabel marks workloads that the restore flow scales to zero
	// and back up again.
	scaledownScopeLabel = "k8s.cloudogu.com/restore-scaledown-scope"
	// scaledownReplicasLabel holds the replica count captured at scale-down
	// time. A leftover value means a previous restore did not finish.
	scaledownReplicasLabel = "k8s.cloudogu.com/restore-scaledown-replicas"
	// backupScopeLabel marks additional resources that the restore flow deletes
	// before restoring.
	backupScopeLabel = "k8s.cloudogu.com/backup-scope"
	// acceptanceProviderHoldFinalizer keeps the Velero restore observable after its deletion was
	// requested, so the spec can verify that the parent remains protected until the child is gone.
	acceptanceProviderHoldFinalizer = "k8s.cloudogu.com/acceptance-provider-hold"
	// acceptanceParentHoldFinalizer keeps the parent observable after the operator released it, so
	// the deleting status and the operator finalizer removal can be asserted independently.
	acceptanceParentHoldFinalizer = "k8s.cloudogu.com/acceptance-parent-hold"
	// These values mirror the restore controller's namespace-wide Lease contract. Keeping them in
	// the black-box spec avoids coupling the acceptance test to controller implementation types.
	restoreLeaseName                 = "k8s-backup-operator-restore"
	restoreLeaseHolderNameAnnotation = "k8s.cloudogu.com/restore-lease-holder-name"
	reasonWaitingForActiveRestore    = "WaitingForActiveRestore"

	restoreTestNamespace       = "ecosystem"
	throwawayReplicas    int32 = 2
)

// Restore is a namespace-wide destructive operation: it scales down every
// workload labeled with the scaledown scope, deletes *all* dogus, and deletes
// every ConfigMap, Secret and PVC labeled with the backup scope. It therefore
// must never run concurrently with the Backup specs or with itself, hence
// Serial. Run this suite only against a disposable cluster.
var _ = Describe("Restore", Serial, Ordered, Label("restore"), func() {
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

		// This backup is deliberately NOT deleted afterwards. The snapshots it captured were deleted faster
		// than the restore of the dogus happened, so there were always dogus left stuck without pvcs.
		By("creating a backup for the restore to reference")
		backup := createBackupWithObjectKey(backupKey)
		Expect(k8sClient.Create(ctx, backup)).Should(Succeed())
		expectBackupStatus(ctx, backupKey, backupv1.BackupStatusCompleted)

		// Created only after the backup completed, so it is absent
		// from the backup. Cleanup must delete it and we assert that nothing brings it back.
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

				maintenanceMode := repository.NewMaintenanceModeAdapter("k8s-backup-operator", k8sClient, restoreTestNamespace)
				description, active, err := maintenanceMode.GetStatus(ctx)
				g.Expect(err).ShouldNot(HaveOccurred())
				g.Expect(active).Should(BeTrue(),
					"maintenance mode must be active before the provider restore starts")
				g.Expect(description.Title).Should(Equal("Service temporary unavailable"))
				g.Expect(description.Text).Should(Equal("Restore in progress"))
			}).
				WithTimeout(10 * time.Minute).
				WithPolling(5 * time.Second).
				Should(Succeed())
		})

		It("the restore reaches the completed status", func(ctx SpecContext) {
			expectRestoreStatus(ctx, restoreKey, backupv1.RestoreStatusCompleted)

			maintenanceMode := repository.NewMaintenanceModeAdapter("k8s-backup-operator", k8sClient, restoreTestNamespace)
			Eventually(func(g Gomega) {
				_, active, err := maintenanceMode.GetStatus(ctx)
				g.Expect(err).ShouldNot(HaveOccurred())
				g.Expect(active).Should(BeFalse(),
					"a completed restore must have deactivated maintenance mode")
			}).
				WithTimeout(2 * time.Minute).
				WithPolling(2 * time.Second).
				Should(Succeed())
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
		BeforeAll(func(ctx SpecContext) {
			DeferCleanup(func(ctx SpecContext) {
				removeObjectFinalizerAndIgnoreNotFound(ctx, restoreKey, &velerov1.Restore{}, acceptanceProviderHoldFinalizer)
				removeObjectFinalizerAndIgnoreNotFound(ctx, restoreKey, &backupv1.Restore{}, acceptanceParentHoldFinalizer)
				deleteAndIgnoreNotFound(ctx, &velerov1.Restore{ObjectMeta: metav1.ObjectMeta{Name: restoreKey.Name, Namespace: restoreKey.Namespace}})
				deleteAndIgnoreNotFound(ctx, newRestoreWithObjectKey(restoreKey, backupKey.Name))
			})

			By("holding the parent and provider restore so every deletion stage remains observable")
			ensureObjectFinalizer(ctx, restoreKey, &backupv1.Restore{}, acceptanceParentHoldFinalizer)
			ensureObjectFinalizer(ctx, restoreKey, &velerov1.Restore{}, acceptanceProviderHoldFinalizer)
		})

		It("if the restore is deleted", func(ctx SpecContext) {
			restore := newRestoreWithObjectKey(restoreKey, backupKey.Name)
			Expect(k8sClient.Delete(ctx, restore)).Should(Succeed())
		})

		It("the provider restore starts terminating while the parent remains protected", func(ctx SpecContext) {
			Eventually(func(g Gomega) {
				veleroRestore := &velerov1.Restore{}
				g.Expect(k8sClient.Get(ctx, restoreKey, veleroRestore)).Should(Succeed())
				g.Expect(veleroRestore.DeletionTimestamp).ShouldNot(BeNil())

				restore := &backupv1.Restore{}
				g.Expect(k8sClient.Get(ctx, restoreKey, restore)).Should(Succeed())
				g.Expect(restore.Finalizers).Should(ContainElement(backupv1.RestoreFinalizer),
					"the parent must remain protected while its provider restore still exists")
			}).
				WithTimeout(10 * time.Minute).
				WithPolling(5 * time.Second).
				Should(Succeed())
		})

		It("the provider restore is deleted after its hold is released", func(ctx SpecContext) {
			removeObjectFinalizerAndIgnoreNotFound(ctx, restoreKey, &velerov1.Restore{}, acceptanceProviderHoldFinalizer)

			Eventually(func(g Gomega) {
				veleroRestore := &velerov1.Restore{}
				err := k8sClient.Get(ctx, restoreKey, veleroRestore)
				g.Expect(apierrors.IsNotFound(err)).To(BeTrue())
			}).
				WithTimeout(10 * time.Minute).
				WithPolling(10 * time.Second).
				Should(Succeed())
		})

		It("the parent persists the deleting status and releases only the operator finalizer", func(ctx SpecContext) {
			Eventually(func(g Gomega) {
				restore := &backupv1.Restore{}
				g.Expect(k8sClient.Get(ctx, restoreKey, restore)).Should(Succeed())
				g.Expect(restore.Status.Status).Should(Equal(backupv1.RestoreStatusDeleting))
				g.Expect(restore.Finalizers).ShouldNot(ContainElement(backupv1.RestoreFinalizer))
				g.Expect(restore.Finalizers).Should(ContainElement(acceptanceParentHoldFinalizer),
					"the operator must not remove finalizers owned by another controller")
			}).
				WithTimeout(10 * time.Minute).
				WithPolling(5 * time.Second).
				Should(Succeed())
		})

		It("the parent is deleted after the acceptance hold is released", func(ctx SpecContext) {
			removeObjectFinalizerAndIgnoreNotFound(ctx, restoreKey, &backupv1.Restore{}, acceptanceParentHoldFinalizer)

			Eventually(func(g Gomega) {
				restore := &backupv1.Restore{}
				err := k8sClient.Get(ctx, restoreKey, restore)
				g.Expect(apierrors.IsNotFound(err)).To(BeTrue())
			}).
				WithTimeout(2 * time.Minute).
				WithPolling(2 * time.Second).
				Should(Succeed())
		})
	})

	Describe("Serializing concurrent Restores with a Lease", Ordered, func() {
		firstKey := client.ObjectKey{Namespace: restoreTestNamespace, Name: fmt.Sprintf("restore-lease-first-%s", suffix)}
		secondKey := client.ObjectKey{Namespace: restoreTestNamespace, Name: fmt.Sprintf("restore-lease-second-%s", suffix)}
		leaseKey := client.ObjectKey{Namespace: restoreTestNamespace, Name: restoreLeaseName}
		var holderKey client.ObjectKey
		var waiterKey client.ObjectKey
		var initialLeaseTransitions int32

		BeforeAll(func(ctx SpecContext) {
			DeferCleanup(func(ctx SpecContext) {
				deleteAndIgnoreNotFound(ctx, newRestoreWithObjectKey(firstKey, backupKey.Name))
				deleteAndIgnoreNotFound(ctx, newRestoreWithObjectKey(secondKey, backupKey.Name))
			})
		})

		It("creates two competing restores", func(ctx SpecContext) {
			Expect(k8sClient.Create(ctx, newRestoreWithObjectKey(firstKey, backupKey.Name))).Should(Succeed())
			Expect(k8sClient.Create(ctx, newRestoreWithObjectKey(secondKey, backupKey.Name))).Should(Succeed())
		})

		It("allows only the lease holder to start its provider restore", func(ctx SpecContext) {
			Eventually(func(g Gomega) {
				lease := &coordinationv1.Lease{}
				g.Expect(k8sClient.Get(ctx, leaseKey, lease)).Should(Succeed())
				g.Expect(lease.Spec.HolderIdentity).ShouldNot(BeNil())
				g.Expect(lease.Spec.LeaseTransitions).ShouldNot(BeNil())

				var observedHolderKey client.ObjectKey
				var observedWaiterKey client.ObjectKey
				switch lease.Annotations[restoreLeaseHolderNameAnnotation] {
				case firstKey.Name:
					observedHolderKey, observedWaiterKey = firstKey, secondKey
				case secondKey.Name:
					observedHolderKey, observedWaiterKey = secondKey, firstKey
				default:
					g.Expect(lease.Annotations[restoreLeaseHolderNameAnnotation]).Should(
						Or(Equal(firstKey.Name), Equal(secondKey.Name)),
						"the lease must be held by one of the competing restores")
					return
				}

				holder := &backupv1.Restore{}
				g.Expect(k8sClient.Get(ctx, observedHolderKey, holder)).Should(Succeed())
				g.Expect(*lease.Spec.HolderIdentity).Should(Equal(string(holder.UID)))

				waiter := &backupv1.Restore{}
				g.Expect(k8sClient.Get(ctx, observedWaiterKey, waiter)).Should(Succeed())
				condition := meta.FindStatusCondition(waiter.Status.Conditions, backupv1.ConditionSuccessful)
				g.Expect(condition).ShouldNot(BeNil())
				g.Expect(condition.Status).Should(Equal(metav1.ConditionUnknown))
				g.Expect(condition.Reason).Should(Equal(reasonWaitingForActiveRestore))

				providerRestore := &velerov1.Restore{}
				err := k8sClient.Get(ctx, observedWaiterKey, providerRestore)
				g.Expect(apierrors.IsNotFound(err)).Should(BeTrue(),
					"the waiting restore must not start a provider restore before it owns the lease")

				holderKey, waiterKey = observedHolderKey, observedWaiterKey
				initialLeaseTransitions = *lease.Spec.LeaseTransitions
			}).
				WithTimeout(10 * time.Minute).
				WithPolling(2 * time.Second).
				Should(Succeed())
		})

		It("completes the initial lease holder", func(ctx SpecContext) {
			expectRestoreStatus(ctx, holderKey, backupv1.RestoreStatusCompleted)
		})

		It("transfers the lease to the waiting restore", func(ctx SpecContext) {
			Eventually(func(g Gomega) {
				waiter := &backupv1.Restore{}
				g.Expect(k8sClient.Get(ctx, waiterKey, waiter)).Should(Succeed())

				lease := &coordinationv1.Lease{}
				g.Expect(k8sClient.Get(ctx, leaseKey, lease)).Should(Succeed())
				g.Expect(lease.Annotations[restoreLeaseHolderNameAnnotation]).Should(Equal(waiterKey.Name))
				g.Expect(lease.Spec.HolderIdentity).ShouldNot(BeNil())
				g.Expect(*lease.Spec.HolderIdentity).Should(Equal(string(waiter.UID)))
				g.Expect(lease.Spec.LeaseTransitions).ShouldNot(BeNil())
				g.Expect(*lease.Spec.LeaseTransitions).Should(BeNumerically(">", initialLeaseTransitions))
			}).
				WithTimeout(10 * time.Minute).
				WithPolling(2 * time.Second).
				Should(Succeed())
		})

		It("starts and completes the restore after acquiring the lease", func(ctx SpecContext) {
			Eventually(func(g Gomega) {
				providerRestore := &velerov1.Restore{}
				g.Expect(k8sClient.Get(ctx, waiterKey, providerRestore)).Should(Succeed())
			}).
				WithTimeout(10 * time.Minute).
				WithPolling(5 * time.Second).
				Should(Succeed())

			expectRestoreStatus(ctx, waiterKey, backupv1.RestoreStatusCompleted)
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

// ensureObjectFinalizer adds a test-owned finalizer and retries conflicts with other controllers.
func ensureObjectFinalizer(ctx SpecContext, key client.ObjectKey, object client.Object, finalizer string) {
	Eventually(func(g Gomega) {
		g.Expect(k8sClient.Get(ctx, key, object)).Should(Succeed())
		before := object.DeepCopyObject().(client.Object)
		if !controllerutil.AddFinalizer(object, finalizer) {
			return
		}

		g.Expect(k8sClient.Patch(ctx, object, client.MergeFrom(before))).Should(Succeed())
	}).
		WithTimeout(2 * time.Minute).
		WithPolling(time.Second).
		Should(Succeed())
}

// removeObjectFinalizerAndIgnoreNotFound releases a test-owned hold during the spec and cleanup.
func removeObjectFinalizerAndIgnoreNotFound(ctx SpecContext, key client.ObjectKey, object client.Object, finalizer string) {
	Eventually(func(g Gomega) {
		err := k8sClient.Get(ctx, key, object)
		if apierrors.IsNotFound(err) {
			return
		}
		g.Expect(err).ShouldNot(HaveOccurred())

		before := object.DeepCopyObject().(client.Object)
		if !controllerutil.RemoveFinalizer(object, finalizer) {
			return
		}

		g.Expect(k8sClient.Patch(ctx, object, client.MergeFrom(before))).Should(Succeed())
	}).
		WithTimeout(2 * time.Minute).
		WithPolling(time.Second).
		Should(Succeed())
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
