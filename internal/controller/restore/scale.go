package restore

import (
	"context"
	"fmt"
	"strconv"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/k8s-backup-operator/internal/logging"
	componentv1 "github.com/cloudogu/k8s-component-lib/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	labelScaledownScope    = "k8s.cloudogu.com/restore-scaledown-scope"
	labelScaledownReplicas = "k8s.cloudogu.com/restore-scaledown-replicas"
)

// DefaultManager scales workloads down before restore and back up after restore.
type DefaultManager struct {
	k8sClient k8sClient
	namespace string
}

// NewScaleManager creates a new instance of DefaultManager.
func NewScaleManager(k8sClient k8sClient, namespace string) *DefaultManager {
	return &DefaultManager{k8sClient: k8sClient, namespace: namespace}
}

// ScaleDown finds all Deployments, StatefulSets, ReplicaSets, and ReplicationControllers
// labeled with the scaledown scope label, stores their current replica count in a label,
// and scales them to zero.
func (m *DefaultManager) ScaleDown(ctx context.Context, restoredDogus, restoredComponents []string) error {
	logging.Debug(ctx, "scaling down workloads labeled for restore scaledown...")

	doguSelect, componentsSelect, err := m.scaleDownListOptions(restoredDogus, restoredComponents)
	if err != nil {
		return fmt.Errorf("create list options: %w", err)
	}

	if len(restoredDogus) > 0 {
		err = m.scaleDownWorkloads(ctx, doguSelect)
		if err != nil {
			return fmt.Errorf("scale down dogus: %w", err)
		}
	}

	if len(restoredComponents) > 0 {
		err = m.scaleDownWorkloads(ctx, componentsSelect)
		if err != nil {
			return fmt.Errorf("scale down components: %w", err)
		}
	}

	logging.Debug(ctx, "workload scaledown complete...")

	return nil
}

func (m *DefaultManager) scaleDownListOptions(restoredDogus, restoredComponents []string) (doguSelect []client.ListOption, componentsSelect []client.ListOption, err error) {
	hasScaledownScope, err := labels.NewRequirement(labelScaledownScope, selection.Exists, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("create scaledown-scope requirement: %w", err)
	}

	// Select nothing if no dogus are restored
	doguSelector := labels.Nothing()
	if len(restoredDogus) > 0 {
		inRestoredDogus, err := labels.NewRequirement(backupv1.DoguNameLabelKey, selection.In, restoredDogus)
		if err != nil {
			return nil, nil, fmt.Errorf("create restored dogus requirement: %w", err)
		}

		doguSelector = labels.NewSelector()
		doguSelector = doguSelector.Add(*hasScaledownScope, *inRestoredDogus)
	}

	// Select nothing if no components are restored
	componentSelector := labels.Nothing()
	if len(restoredComponents) > 0 {
		inRestoredComponents, err := labels.NewRequirement(componentv1.ComponentNameLabelKey, selection.In, restoredComponents)
		if err != nil {
			return nil, nil, fmt.Errorf("create restored components requirement: %w", err)
		}

		componentSelector = labels.NewSelector()
		componentSelector = componentSelector.Add(*hasScaledownScope, *inRestoredComponents)
	}

	return []client.ListOption{
		client.InNamespace(m.namespace),
		client.MatchingLabelsSelector{Selector: doguSelector},
	}, []client.ListOption{
		client.InNamespace(m.namespace),
		client.MatchingLabelsSelector{Selector: componentSelector},
	}, nil
}

func (m *DefaultManager) scaleDownWorkloads(ctx context.Context, listOpts []client.ListOption) error {
	if err := m.scaleDownDeployments(ctx, listOpts); err != nil {
		return fmt.Errorf("scale down Deployments: %w", err)
	}

	if err := m.scaleDownStatefulSets(ctx, listOpts); err != nil {
		return fmt.Errorf("scale down StatefulSets: %w", err)
	}

	if err := m.scaleDownReplicaSets(ctx, listOpts); err != nil {
		return fmt.Errorf("scale down ReplicaSets: %w", err)
	}

	if err := m.scaleDownReplicationControllers(ctx, listOpts); err != nil {
		return fmt.Errorf("scale down ReplicationControllers: %w", err)
	}
	return nil
}

// ScaleUp finds all Deployments, StatefulSets, ReplicaSets, and ReplicationControllers
// labeled with the scaledown scope label and restores the stored replica count.
// The replicas label is retained so later recovery stages can identify and observe the workloads.
func (m *DefaultManager) ScaleUp(ctx context.Context) error {
	logging.Debug(ctx, "scaling up workloads after restore...")

	listOpts := []client.ListOption{
		client.InNamespace(m.namespace),
		client.HasLabels{labelScaledownScope},
	}

	if err := m.scaleUpDeployments(ctx, listOpts); err != nil {
		return fmt.Errorf("scale up Deployments: %w", err)
	}

	if err := m.scaleUpStatefulSets(ctx, listOpts); err != nil {
		return fmt.Errorf("scale up StatefulSets: %w", err)
	}

	if err := m.scaleUpReplicaSets(ctx, listOpts); err != nil {
		return fmt.Errorf("scale up ReplicaSets: %w", err)
	}

	if err := m.scaleUpReplicationControllers(ctx, listOpts); err != nil {
		return fmt.Errorf("scale up ReplicationControllers: %w", err)
	}

	logging.Debug(ctx, "workload scaleup complete...")

	return nil
}

func (m *DefaultManager) scaleDownDeployments(ctx context.Context, listOpts []client.ListOption) error {
	list := &appsv1.DeploymentList{}
	return forEachWorkload(ctx, m.k8sClient, list, listOpts, "list deployments for scaledown", func() []appsv1.Deployment {
		return list.Items
	}, func(deploy *appsv1.Deployment) error {
		return m.scaleDownWorkload(ctx, deploy, &deploy.Spec.Replicas, "deployment")
	})
}

func (m *DefaultManager) scaleDownStatefulSets(ctx context.Context, listOpts []client.ListOption) error {
	list := &appsv1.StatefulSetList{}
	return forEachWorkload(ctx, m.k8sClient, list, listOpts, "list statefulsets for scaledown", func() []appsv1.StatefulSet {
		return list.Items
	}, func(sts *appsv1.StatefulSet) error {
		return m.scaleDownWorkload(ctx, sts, &sts.Spec.Replicas, "statefulset")
	})
}

func (m *DefaultManager) scaleDownReplicaSets(ctx context.Context, listOpts []client.ListOption) error {

	list := &appsv1.ReplicaSetList{}
	return forEachWorkload(ctx, m.k8sClient, list, listOpts, "list replicasets for scaledown", func() []appsv1.ReplicaSet {
		return list.Items
	}, func(rs *appsv1.ReplicaSet) error {
		if len(rs.OwnerReferences) > 0 {
			logging.Debug(ctx, "skipping replicaset with owner references for scaledown", "name", rs.Name)
			return nil
		}
		return m.scaleDownWorkload(ctx, rs, &rs.Spec.Replicas, "replicaset")
	})
}

func (m *DefaultManager) scaleDownReplicationControllers(ctx context.Context, listOpts []client.ListOption) error {
	list := &corev1.ReplicationControllerList{}
	return forEachWorkload(ctx, m.k8sClient, list, listOpts, "list replicationcontrollers for scaledown", func() []corev1.ReplicationController {
		return list.Items
	}, func(rc *corev1.ReplicationController) error {
		return m.scaleDownWorkload(ctx, rc, &rc.Spec.Replicas, "replicationcontroller")
	})
}

func (m *DefaultManager) scaleUpDeployments(ctx context.Context, listOpts []client.ListOption) error {
	list := &appsv1.DeploymentList{}
	return forEachWorkload(ctx, m.k8sClient, list, listOpts, "list deployments for scaleup", func() []appsv1.Deployment {
		return list.Items
	}, func(deploy *appsv1.Deployment) error {
		return m.scaleUpWorkload(ctx, deploy, &deploy.Spec.Replicas, "deployment")
	})
}

func (m *DefaultManager) scaleUpStatefulSets(ctx context.Context, listOpts []client.ListOption) error {
	list := &appsv1.StatefulSetList{}
	return forEachWorkload(ctx, m.k8sClient, list, listOpts, "list statefulsets for scaleup", func() []appsv1.StatefulSet {
		return list.Items
	}, func(sts *appsv1.StatefulSet) error {
		return m.scaleUpWorkload(ctx, sts, &sts.Spec.Replicas, "statefulset")
	})
}

func (m *DefaultManager) scaleUpReplicaSets(ctx context.Context, listOpts []client.ListOption) error {
	list := &appsv1.ReplicaSetList{}
	return forEachWorkload(ctx, m.k8sClient, list, listOpts, "list replicasets for scaleup", func() []appsv1.ReplicaSet {
		return list.Items
	}, func(rs *appsv1.ReplicaSet) error {
		return m.scaleUpWorkload(ctx, rs, &rs.Spec.Replicas, "replicaset")
	})
}

func (m *DefaultManager) scaleUpReplicationControllers(ctx context.Context, listOpts []client.ListOption) error {
	list := &corev1.ReplicationControllerList{}
	return forEachWorkload(ctx, m.k8sClient, list, listOpts, "list replicationcontrollers for scaleup", func() []corev1.ReplicationController {
		return list.Items
	}, func(rc *corev1.ReplicationController) error {
		return m.scaleUpWorkload(ctx, rc, &rc.Spec.Replicas, "replicationcontroller")
	})
}

func forEachWorkload[T any](
	ctx context.Context,
	k8sClient k8sClient,
	list client.ObjectList,
	listOpts []client.ListOption,
	listError string,
	items func() []T,
	apply func(*T) error,
) error {
	if err := k8sClient.List(ctx, list, listOpts...); err != nil {
		return fmt.Errorf("%s: %w", listError, err)
	}
	workloadItems := items()
	for i := range workloadItems {
		if err := apply(&workloadItems[i]); err != nil {
			return err
		}
	}
	return nil
}

// FinalizeScaleUp removes the temporary replica labels after every workload became ready.
// Repeated calls are safe and finish a partially completed label removal.
func (m *DefaultManager) FinalizeScaleUp(ctx context.Context) error {
	// scaledown-replicas should only be set for workloads of the restore so we don't have to restrict this any further
	listOpts := []client.ListOption{
		client.InNamespace(m.namespace),
		client.HasLabels{labelScaledownScope, labelScaledownReplicas},
	}

	deployments := &appsv1.DeploymentList{}
	if err := finalizeWorkloads(ctx, m, deployments, listOpts, "list deployments for scale-up finalization", "deployment", func() []appsv1.Deployment {
		return deployments.Items
	}, func(deployment *appsv1.Deployment) client.Object { return deployment }); err != nil {
		return err
	}

	statefulSets := &appsv1.StatefulSetList{}
	if err := finalizeWorkloads(ctx, m, statefulSets, listOpts, "list statefulsets for scale-up finalization", "statefulset", func() []appsv1.StatefulSet {
		return statefulSets.Items
	}, func(statefulSet *appsv1.StatefulSet) client.Object { return statefulSet }); err != nil {
		return err
	}

	replicaSets := &appsv1.ReplicaSetList{}
	if err := finalizeWorkloads(ctx, m, replicaSets, listOpts, "list replicasets for scale-up finalization", "replicaset", func() []appsv1.ReplicaSet {
		return replicaSets.Items
	}, func(replicaSet *appsv1.ReplicaSet) client.Object { return replicaSet }); err != nil {
		return err
	}

	replicationControllers := &corev1.ReplicationControllerList{}
	return finalizeWorkloads(ctx, m, replicationControllers, listOpts, "list replicationcontrollers for scale-up finalization", "replicationcontroller", func() []corev1.ReplicationController {
		return replicationControllers.Items
	}, func(replicationController *corev1.ReplicationController) client.Object { return replicationController })
}

func finalizeWorkloads[T any](
	ctx context.Context,
	manager *DefaultManager,
	list client.ObjectList,
	listOpts []client.ListOption,
	listError string,
	resourceKind string,
	items func() []T,
	object func(*T) client.Object,
) error {
	return forEachWorkload(ctx, manager.k8sClient, list, listOpts, listError, items, func(workload *T) error {
		return manager.removeStoredReplicasLabel(ctx, object(workload), resourceKind)
	})
}

func (m *DefaultManager) removeStoredReplicasLabel(ctx context.Context, workload client.Object, resourceKind string) error {
	delete(workload.GetLabels(), labelScaledownReplicas)
	if err := m.k8sClient.Update(ctx, workload); err != nil {
		return fmt.Errorf("finalize scale-up for %s %s: %w", resourceKind, workload.GetName(), err)
	}

	return nil
}

// AreWorkloadsReady checks whether all workloads in the scale-down scope reached their target replica
// count. While the recovery label exists it is the target source; after finalization spec.replicas keeps
// the workload observable across partial cleanup and later reconciliations. The method never mutates.
func (m *DefaultManager) AreWorkloadsReady(ctx context.Context) (bool, error) {
	listOpts := []client.ListOption{
		client.InNamespace(m.namespace),
		client.HasLabels{labelScaledownScope},
	}

	deploymentsReady, err := m.deploymentsReady(ctx, listOpts)
	if err != nil {
		return false, fmt.Errorf("check Deployment readiness: %w", err)
	}

	statefulSetsReady, err := m.statefulSetsReady(ctx, listOpts)
	if err != nil {
		return false, fmt.Errorf("check StatefulSet readiness: %w", err)
	}

	replicaSetsReady, err := m.replicaSetsReady(ctx, listOpts)
	if err != nil {
		return false, fmt.Errorf("check ReplicaSet readiness: %w", err)
	}

	replicationControllersReady, err := m.replicationControllersReady(ctx, listOpts)
	if err != nil {
		return false, fmt.Errorf("check ReplicationController readiness: %w", err)
	}

	return deploymentsReady && statefulSetsReady && replicaSetsReady && replicationControllersReady, nil
}

func (m *DefaultManager) deploymentsReady(ctx context.Context, listOpts []client.ListOption) (bool, error) {
	list := &appsv1.DeploymentList{}

	ready, err := workloadsReady(ctx, m.k8sClient, list, listOpts, "list deployments for readiness check", func() []appsv1.Deployment {
		return list.Items
	}, func(deployment *appsv1.Deployment) workloadReadiness {
		return workloadReadiness{
			labels: deployment.Labels, desiredReplicas: deployment.Spec.Replicas,
			resourceKind: "deployment", resourceName: deployment.Name,
			generation: deployment.Generation, observedGeneration: deployment.Status.ObservedGeneration,
			replicas: deployment.Status.Replicas, readyReplicas: deployment.Status.ReadyReplicas,
			availableReplicas: deployment.Status.AvailableReplicas,
		}
	})
	if err != nil || !ready {
		return false, err
	}

	// At this point we know that status.replicas is equal to the target for every deployment,
	// so status.replicas is the rollout target here.
	for _, deployment := range list.Items {
		if deployment.Status.UpdatedReplicas != deployment.Status.Replicas || deployment.Status.UnavailableReplicas != 0 {
			return false, nil
		}
	}

	return true, nil
}

func (m *DefaultManager) statefulSetsReady(ctx context.Context, listOpts []client.ListOption) (bool, error) {
	list := &appsv1.StatefulSetList{}
	return workloadsReady(ctx, m.k8sClient, list, listOpts, "list statefulsets for readiness check", func() []appsv1.StatefulSet {
		return list.Items
	}, func(statefulSet *appsv1.StatefulSet) workloadReadiness {
		return workloadReadiness{
			labels: statefulSet.Labels, desiredReplicas: statefulSet.Spec.Replicas,
			resourceKind: "statefulset", resourceName: statefulSet.Name,
			generation: statefulSet.Generation, observedGeneration: statefulSet.Status.ObservedGeneration,
			replicas: statefulSet.Status.Replicas, readyReplicas: statefulSet.Status.ReadyReplicas,
			availableReplicas: statefulSet.Status.AvailableReplicas,
		}
	})
}

func (m *DefaultManager) replicaSetsReady(ctx context.Context, listOpts []client.ListOption) (bool, error) {
	list := &appsv1.ReplicaSetList{}
	return workloadsReady(ctx, m.k8sClient, list, listOpts, "list replicasets for readiness check", func() []appsv1.ReplicaSet {
		return list.Items
	}, func(replicaSet *appsv1.ReplicaSet) workloadReadiness {
		return workloadReadiness{
			labels: replicaSet.Labels, desiredReplicas: replicaSet.Spec.Replicas,
			resourceKind: "replicaset", resourceName: replicaSet.Name,
			generation: replicaSet.Generation, observedGeneration: replicaSet.Status.ObservedGeneration,
			replicas: replicaSet.Status.Replicas, readyReplicas: replicaSet.Status.ReadyReplicas,
			availableReplicas: replicaSet.Status.AvailableReplicas,
		}
	})
}

func (m *DefaultManager) replicationControllersReady(ctx context.Context, listOpts []client.ListOption) (bool, error) {
	list := &corev1.ReplicationControllerList{}
	return workloadsReady(ctx, m.k8sClient, list, listOpts, "list replicationcontrollers for readiness check", func() []corev1.ReplicationController {
		return list.Items
	}, func(replicationController *corev1.ReplicationController) workloadReadiness {
		return workloadReadiness{
			labels: replicationController.Labels, desiredReplicas: replicationController.Spec.Replicas,
			resourceKind: "replicationcontroller", resourceName: replicationController.Name,
			generation: replicationController.Generation, observedGeneration: replicationController.Status.ObservedGeneration,
			replicas: replicationController.Status.Replicas, readyReplicas: replicationController.Status.ReadyReplicas,
			availableReplicas: replicationController.Status.AvailableReplicas,
		}
	})
}

type workloadReadiness struct {
	labels             map[string]string
	desiredReplicas    *int32
	resourceKind       string
	resourceName       string
	generation         int64
	observedGeneration int64
	replicas           int32
	readyReplicas      int32
	availableReplicas  int32
}

func workloadsReady[T any](
	ctx context.Context,
	k8sClient k8sClient,
	list client.ObjectList,
	listOpts []client.ListOption,
	listError string,
	items func() []T,
	readiness func(*T) workloadReadiness,
) (bool, error) {
	allReady := true
	err := forEachWorkload(ctx, k8sClient, list, listOpts, listError, items, func(workload *T) error {
		state := readiness(workload)
		target, err := targetReplicasForReadiness(state.labels, state.desiredReplicas, state.resourceKind, state.resourceName)
		if err != nil {
			return err
		}
		allReady = allReady && state.desiredReplicas != nil && *state.desiredReplicas == target &&
			state.observedGeneration >= state.generation && state.replicas == target &&
			state.readyReplicas == target && state.availableReplicas == target
		return nil
	})
	return allReady, err
}

func targetReplicasForReadiness(
	labels map[string]string,
	desiredReplicas *int32,
	resourceKind string,
	resourceName string,
) (int32, error) {
	if _, exists := labels[labelScaledownReplicas]; exists {
		return storedTargetReplicas(labels, resourceKind, resourceName)
	}
	if desiredReplicas == nil {
		return 0, fmt.Errorf("%s %s has neither a stored nor a desired replica count", resourceKind, resourceName)
	}

	return *desiredReplicas, nil
}

func storedTargetReplicas(labels map[string]string, resourceKind string, resourceName string) (int32, error) {
	replicas, exists, err := storedTargetReplicasIfPresent(labels, resourceKind, resourceName)
	if err != nil {
		return 0, err
	}
	if !exists {
		return 0, fmt.Errorf("%s %s has no stored replica count", resourceKind, resourceName)
	}

	return replicas, nil
}

func storedTargetReplicasIfPresent(
	labels map[string]string,
	resourceKind string,
	resourceName string,
) (int32, bool, error) {
	replicaValue, exists := labels[labelScaledownReplicas]
	if !exists {
		return 0, false, nil
	}

	replicas, err := strconv.ParseInt(replicaValue, 10, 32)
	if err != nil {
		return 0, true, fmt.Errorf("parse stored replica count for %s %s: %w", resourceKind, resourceName, err)
	}

	return int32(replicas), true, nil
}

func (m *DefaultManager) scaleUpWorkload(
	ctx context.Context,
	workload client.Object,
	replicas **int32,
	resourceKind string,
) error {
	targetReplicas, exists, err := storedTargetReplicasIfPresent(
		workload.GetLabels(),
		resourceKind,
		workload.GetName(),
	)
	if err != nil || !exists {
		return err
	}
	if *replicas != nil && **replicas == targetReplicas {
		return nil
	}

	*replicas = new(targetReplicas)
	if err := m.k8sClient.Update(ctx, workload); err != nil {
		return fmt.Errorf("scale up %s %s: %w", resourceKind, workload.GetName(), err)
	}

	return nil
}

func (m *DefaultManager) scaleDownWorkload(
	ctx context.Context,
	workload client.Object,
	replicas **int32,
	resourceKind string,
) error {
	if _, alreadyScaled := workload.GetLabels()[labelScaledownReplicas]; alreadyScaled {
		return nil
	}

	var currentReplicas int32
	if *replicas != nil {
		currentReplicas = **replicas
	}

	workload.GetLabels()[labelScaledownReplicas] = strconv.FormatInt(int64(currentReplicas), 10)
	*replicas = zeroReplicas()

	if err := m.k8sClient.Update(ctx, workload); err != nil {
		return fmt.Errorf("scale down %s %s: %w", resourceKind, workload.GetName(), err)
	}

	return nil
}

func zeroReplicas() *int32 {
	return new(int32(0))
}
