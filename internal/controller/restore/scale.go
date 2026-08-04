package restore

import (
	"context"
	"fmt"
	"strconv"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
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
func (m *DefaultManager) ScaleDown(ctx context.Context) error {
	logger := log.FromContext(ctx)
	logger.Info("scaling down workloads labeled for restore scaledown...")

	listOpts := []client.ListOption{
		client.InNamespace(m.namespace),
		client.HasLabels{labelScaledownScope},
	}

	if err := m.scaleDownDeployments(ctx, listOpts); err != nil {
		return fmt.Errorf("failed to scale down Deployments: %w", err)
	}

	if err := m.scaleDownStatefulSets(ctx, listOpts); err != nil {
		return fmt.Errorf("failed to scale down StatefulSets: %w", err)
	}

	if err := m.scaleDownReplicaSets(ctx, listOpts); err != nil {
		return fmt.Errorf("failed to scale down ReplicaSets: %w", err)
	}

	if err := m.scaleDownReplicationControllers(ctx, listOpts); err != nil {
		return fmt.Errorf("failed to scale down ReplicationControllers: %w", err)
	}

	logger.Info("workload scaledown complete...")

	return nil
}

// ScaleUp finds all Deployments, StatefulSets, ReplicaSets, and ReplicationControllers
// labeled with the scaledown scope label and restores the stored replica count.
// The replicas label is retained so later recovery stages can identify and observe the workloads.
func (m *DefaultManager) ScaleUp(ctx context.Context) error {
	logger := log.FromContext(ctx)
	logger.Info("scaling up workloads after restore...")

	listOpts := []client.ListOption{
		client.InNamespace(m.namespace),
		client.HasLabels{labelScaledownScope},
	}

	if err := m.scaleUpDeployments(ctx, listOpts); err != nil {
		return fmt.Errorf("failed to scale up Deployments: %w", err)
	}

	if err := m.scaleUpStatefulSets(ctx, listOpts); err != nil {
		return fmt.Errorf("failed to scale up StatefulSets: %w", err)
	}

	if err := m.scaleUpReplicaSets(ctx, listOpts); err != nil {
		return fmt.Errorf("failed to scale up ReplicaSets: %w", err)
	}

	if err := m.scaleUpReplicationControllers(ctx, listOpts); err != nil {
		return fmt.Errorf("failed to scale up ReplicationControllers: %w", err)
	}

	logger.Info("workload scaleup complete...")

	return nil
}

func (m *DefaultManager) scaleDownDeployments(ctx context.Context, listOpts []client.ListOption) error {
	list := &appsv1.DeploymentList{}
	if err := m.k8sClient.List(ctx, list, listOpts...); err != nil {
		return fmt.Errorf("failed to list deployments for scaledown: %w", err)
	}

	for _, deploy := range list.Items {
		if _, alreadyScaled := deploy.Labels[labelScaledownReplicas]; alreadyScaled {
			continue
		}

		var replicas int32
		if deploy.Spec.Replicas != nil {
			replicas = *deploy.Spec.Replicas
		}

		deploy.Labels[labelScaledownReplicas] = strconv.FormatInt(int64(replicas), 10)
		deploy.Spec.Replicas = zeroReplicas()

		if err := m.k8sClient.Update(ctx, &deploy); err != nil {
			return fmt.Errorf("failed to scale down deployment %s: %w", deploy.Name, err)
		}
	}

	return nil
}

func (m *DefaultManager) scaleDownStatefulSets(ctx context.Context, listOpts []client.ListOption) error {
	list := &appsv1.StatefulSetList{}
	if err := m.k8sClient.List(ctx, list, listOpts...); err != nil {
		return fmt.Errorf("failed to list statefulsets for scaledown: %w", err)
	}

	for _, sts := range list.Items {
		if _, alreadyScaled := sts.Labels[labelScaledownReplicas]; alreadyScaled {
			continue
		}

		var replicas int32
		if sts.Spec.Replicas != nil {
			replicas = *sts.Spec.Replicas
		}

		sts.Labels[labelScaledownReplicas] = strconv.FormatInt(int64(replicas), 10)
		sts.Spec.Replicas = zeroReplicas()

		if err := m.k8sClient.Update(ctx, &sts); err != nil {
			return fmt.Errorf("failed to scale down statefulset %s: %w", sts.Name, err)
		}
	}

	return nil
}

func (m *DefaultManager) scaleDownReplicaSets(ctx context.Context, listOpts []client.ListOption) error {
	logger := log.FromContext(ctx)

	list := &appsv1.ReplicaSetList{}
	if err := m.k8sClient.List(ctx, list, listOpts...); err != nil {
		return fmt.Errorf("failed to list replicasets for scaledown: %w", err)
	}

	for _, rs := range list.Items {
		if len(rs.OwnerReferences) > 0 {
			logger.Info("skipping replicaset with owner references for scaledown", "name", rs.Name)
			continue
		}

		if _, alreadyScaled := rs.Labels[labelScaledownReplicas]; alreadyScaled {
			continue
		}

		var replicas int32
		if rs.Spec.Replicas != nil {
			replicas = *rs.Spec.Replicas
		}

		rs.Labels[labelScaledownReplicas] = strconv.FormatInt(int64(replicas), 10)
		rs.Spec.Replicas = zeroReplicas()

		if err := m.k8sClient.Update(ctx, &rs); err != nil {
			return fmt.Errorf("failed to scale down replicaset %s: %w", rs.Name, err)
		}
	}

	return nil
}

func (m *DefaultManager) scaleDownReplicationControllers(ctx context.Context, listOpts []client.ListOption) error {
	list := &corev1.ReplicationControllerList{}
	if err := m.k8sClient.List(ctx, list, listOpts...); err != nil {
		return fmt.Errorf("failed to list replicationcontrollers for scaledown: %w", err)
	}

	for _, rc := range list.Items {
		if _, alreadyScaled := rc.Labels[labelScaledownReplicas]; alreadyScaled {
			continue
		}

		var replicas int32
		if rc.Spec.Replicas != nil {
			replicas = *rc.Spec.Replicas
		}

		rc.Labels[labelScaledownReplicas] = strconv.FormatInt(int64(replicas), 10)
		rc.Spec.Replicas = zeroReplicas()

		if err := m.k8sClient.Update(ctx, &rc); err != nil {
			return fmt.Errorf("failed to scale down replicationcontroller %s: %w", rc.Name, err)
		}
	}

	return nil
}

func (m *DefaultManager) scaleUpDeployments(ctx context.Context, listOpts []client.ListOption) error {
	list := &appsv1.DeploymentList{}
	if err := m.k8sClient.List(ctx, list, listOpts...); err != nil {
		return fmt.Errorf("failed to list deployments for scaleup: %w", err)
	}

	for _, deploy := range list.Items {
		replicaStr, exists := deploy.Labels[labelScaledownReplicas]
		if !exists {
			continue
		}

		replicas, err := strconv.ParseInt(replicaStr, 10, 32)
		if err != nil {
			return fmt.Errorf("failed to parse stored replica count for deployment %s: %w", deploy.Name, err)
		}

		targetReplicas := int32(replicas)
		if deploy.Spec.Replicas != nil && *deploy.Spec.Replicas == targetReplicas {
			continue
		}

		deploy.Spec.Replicas = new(targetReplicas)

		if lErr := m.k8sClient.Update(ctx, &deploy); lErr != nil {
			return fmt.Errorf("failed to scale up deployment %s: %w", deploy.Name, lErr)
		}
	}

	return nil
}

func (m *DefaultManager) scaleUpStatefulSets(ctx context.Context, listOpts []client.ListOption) error {
	list := &appsv1.StatefulSetList{}
	if err := m.k8sClient.List(ctx, list, listOpts...); err != nil {
		return fmt.Errorf("failed to list statefulsets for scaleup: %w", err)
	}

	for _, sts := range list.Items {
		replicaStr, exists := sts.Labels[labelScaledownReplicas]
		if !exists {
			continue
		}

		replicas, err := strconv.ParseInt(replicaStr, 10, 32)
		if err != nil {
			return fmt.Errorf("failed to parse stored replica count for statefulset %s: %w", sts.Name, err)
		}

		targetReplicas := int32(replicas)
		if sts.Spec.Replicas != nil && *sts.Spec.Replicas == targetReplicas {
			continue
		}

		sts.Spec.Replicas = new(targetReplicas)

		if lErr := m.k8sClient.Update(ctx, &sts); lErr != nil {
			return fmt.Errorf("failed to scale up statefulset %s: %w", sts.Name, lErr)
		}
	}

	return nil
}

func (m *DefaultManager) scaleUpReplicaSets(ctx context.Context, listOpts []client.ListOption) error {
	list := &appsv1.ReplicaSetList{}
	if err := m.k8sClient.List(ctx, list, listOpts...); err != nil {
		return fmt.Errorf("failed to list replicasets for scaleup: %w", err)
	}

	for _, rs := range list.Items {
		replicaStr, exists := rs.Labels[labelScaledownReplicas]
		if !exists {
			continue
		}

		replicas, err := strconv.ParseInt(replicaStr, 10, 32)
		if err != nil {
			return fmt.Errorf("failed to parse stored replica count for replicaset %s: %w", rs.Name, err)
		}

		targetReplicas := int32(replicas)
		if rs.Spec.Replicas != nil && *rs.Spec.Replicas == targetReplicas {
			continue
		}

		rs.Spec.Replicas = new(targetReplicas)

		if lErr := m.k8sClient.Update(ctx, &rs); lErr != nil {
			return fmt.Errorf("failed to scale up replicaset %s: %w", rs.Name, lErr)
		}
	}

	return nil
}

func (m *DefaultManager) scaleUpReplicationControllers(ctx context.Context, listOpts []client.ListOption) error {
	list := &corev1.ReplicationControllerList{}
	if err := m.k8sClient.List(ctx, list, listOpts...); err != nil {
		return fmt.Errorf("failed to list replicationcontrollers for scaleup: %w", err)
	}

	for _, rc := range list.Items {
		replicaStr, exists := rc.Labels[labelScaledownReplicas]
		if !exists {
			continue
		}

		replicas, err := strconv.ParseInt(replicaStr, 10, 32)
		if err != nil {
			return fmt.Errorf("failed to parse stored replica count for replicationcontroller %s: %w", rc.Name, err)
		}

		targetReplicas := int32(replicas)
		if rc.Spec.Replicas != nil && *rc.Spec.Replicas == targetReplicas {
			continue
		}

		rc.Spec.Replicas = new(targetReplicas)

		if lErr := m.k8sClient.Update(ctx, &rc); lErr != nil {
			return fmt.Errorf("failed to scale up replicationcontroller %s: %w", rc.Name, lErr)
		}
	}

	return nil
}

// FinalizeScaleUp removes the temporary replica labels after every workload became ready.
// Repeated calls are safe and finish a partially completed label cleanup.
func (m *DefaultManager) FinalizeScaleUp(ctx context.Context) error {
	listOpts := []client.ListOption{
		client.InNamespace(m.namespace),
		client.HasLabels{labelScaledownScope, labelScaledownReplicas},
	}

	deployments := &appsv1.DeploymentList{}
	if err := m.k8sClient.List(ctx, deployments, listOpts...); err != nil {
		return fmt.Errorf("failed to list deployments for scale-up finalization: %w", err)
	}
	for i := range deployments.Items {
		if err := m.removeStoredReplicasLabel(ctx, &deployments.Items[i], "deployment"); err != nil {
			return err
		}
	}

	statefulSets := &appsv1.StatefulSetList{}
	if err := m.k8sClient.List(ctx, statefulSets, listOpts...); err != nil {
		return fmt.Errorf("failed to list statefulsets for scale-up finalization: %w", err)
	}
	for i := range statefulSets.Items {
		if err := m.removeStoredReplicasLabel(ctx, &statefulSets.Items[i], "statefulset"); err != nil {
			return err
		}
	}

	replicaSets := &appsv1.ReplicaSetList{}
	if err := m.k8sClient.List(ctx, replicaSets, listOpts...); err != nil {
		return fmt.Errorf("failed to list replicasets for scale-up finalization: %w", err)
	}
	for i := range replicaSets.Items {
		if err := m.removeStoredReplicasLabel(ctx, &replicaSets.Items[i], "replicaset"); err != nil {
			return err
		}
	}

	replicationControllers := &corev1.ReplicationControllerList{}
	if err := m.k8sClient.List(ctx, replicationControllers, listOpts...); err != nil {
		return fmt.Errorf("failed to list replicationcontrollers for scale-up finalization: %w", err)
	}
	for i := range replicationControllers.Items {
		if err := m.removeStoredReplicasLabel(ctx, &replicationControllers.Items[i], "replicationcontroller"); err != nil {
			return err
		}
	}

	return nil
}

func (m *DefaultManager) removeStoredReplicasLabel(ctx context.Context, workload client.Object, resourceKind string) error {
	delete(workload.GetLabels(), labelScaledownReplicas)
	if err := m.k8sClient.Update(ctx, workload); err != nil {
		return fmt.Errorf("failed to finalize scale-up for %s %s: %w", resourceKind, workload.GetName(), err)
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
		return false, fmt.Errorf("failed to check Deployment readiness: %w", err)
	}

	statefulSetsReady, err := m.statefulSetsReady(ctx, listOpts)
	if err != nil {
		return false, fmt.Errorf("failed to check StatefulSet readiness: %w", err)
	}

	replicaSetsReady, err := m.replicaSetsReady(ctx, listOpts)
	if err != nil {
		return false, fmt.Errorf("failed to check ReplicaSet readiness: %w", err)
	}

	replicationControllersReady, err := m.replicationControllersReady(ctx, listOpts)
	if err != nil {
		return false, fmt.Errorf("failed to check ReplicationController readiness: %w", err)
	}

	return deploymentsReady && statefulSetsReady && replicaSetsReady && replicationControllersReady, nil
}

func (m *DefaultManager) deploymentsReady(ctx context.Context, listOpts []client.ListOption) (bool, error) {
	list := &appsv1.DeploymentList{}
	if err := m.k8sClient.List(ctx, list, listOpts...); err != nil {
		return false, fmt.Errorf("failed to list deployments for readiness check: %w", err)
	}

	ready := true
	for i := range list.Items {
		deployment := &list.Items[i]
		target, err := targetReplicasForReadiness(deployment.Labels, deployment.Spec.Replicas, "deployment", deployment.Name)
		if err != nil {
			return false, err
		}

		ready = ready && deployment.Spec.Replicas != nil && *deployment.Spec.Replicas == target &&
			deployment.Status.ObservedGeneration >= deployment.Generation &&
			deployment.Status.Replicas == target &&
			deployment.Status.UpdatedReplicas == target &&
			deployment.Status.ReadyReplicas == target &&
			deployment.Status.AvailableReplicas == target &&
			deployment.Status.UnavailableReplicas == 0
	}

	return ready, nil
}

func (m *DefaultManager) statefulSetsReady(ctx context.Context, listOpts []client.ListOption) (bool, error) {
	list := &appsv1.StatefulSetList{}
	if err := m.k8sClient.List(ctx, list, listOpts...); err != nil {
		return false, fmt.Errorf("failed to list statefulsets for readiness check: %w", err)
	}

	ready := true
	for i := range list.Items {
		statefulSet := &list.Items[i]
		target, err := targetReplicasForReadiness(statefulSet.Labels, statefulSet.Spec.Replicas, "statefulset", statefulSet.Name)
		if err != nil {
			return false, err
		}

		ready = ready && statefulSet.Spec.Replicas != nil && *statefulSet.Spec.Replicas == target &&
			statefulSet.Status.ObservedGeneration >= statefulSet.Generation &&
			statefulSet.Status.Replicas == target &&
			statefulSet.Status.ReadyReplicas == target &&
			statefulSet.Status.AvailableReplicas == target
	}

	return ready, nil
}

func (m *DefaultManager) replicaSetsReady(ctx context.Context, listOpts []client.ListOption) (bool, error) {
	list := &appsv1.ReplicaSetList{}
	if err := m.k8sClient.List(ctx, list, listOpts...); err != nil {
		return false, fmt.Errorf("failed to list replicasets for readiness check: %w", err)
	}

	ready := true
	for i := range list.Items {
		replicaSet := &list.Items[i]
		target, err := targetReplicasForReadiness(replicaSet.Labels, replicaSet.Spec.Replicas, "replicaset", replicaSet.Name)
		if err != nil {
			return false, err
		}

		ready = ready && replicaSet.Spec.Replicas != nil && *replicaSet.Spec.Replicas == target &&
			replicaSet.Status.ObservedGeneration >= replicaSet.Generation &&
			replicaSet.Status.Replicas == target &&
			replicaSet.Status.ReadyReplicas == target &&
			replicaSet.Status.AvailableReplicas == target
	}

	return ready, nil
}

func (m *DefaultManager) replicationControllersReady(ctx context.Context, listOpts []client.ListOption) (bool, error) {
	list := &corev1.ReplicationControllerList{}
	if err := m.k8sClient.List(ctx, list, listOpts...); err != nil {
		return false, fmt.Errorf("failed to list replicationcontrollers for readiness check: %w", err)
	}

	ready := true
	for i := range list.Items {
		replicationController := &list.Items[i]
		target, err := targetReplicasForReadiness(replicationController.Labels, replicationController.Spec.Replicas, "replicationcontroller", replicationController.Name)
		if err != nil {
			return false, err
		}

		ready = ready && replicationController.Spec.Replicas != nil && *replicationController.Spec.Replicas == target &&
			replicationController.Status.ObservedGeneration >= replicationController.Generation &&
			replicationController.Status.Replicas == target &&
			replicationController.Status.ReadyReplicas == target &&
			replicationController.Status.AvailableReplicas == target
	}

	return ready, nil
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
	replicaValue, exists := labels[labelScaledownReplicas]
	if !exists {
		return 0, fmt.Errorf("%s %s has no stored replica count", resourceKind, resourceName)
	}

	replicas, err := strconv.ParseInt(replicaValue, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("failed to parse stored replica count for %s %s: %w", resourceKind, resourceName, err)
	}

	return int32(replicas), nil
}

func zeroReplicas() *int32 {
	return new(int32(0))
}
