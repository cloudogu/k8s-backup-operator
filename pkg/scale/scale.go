package scale

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

// NewManager creates a new instance of DefaultManager.
func NewManager(k8sClient k8sClient, namespace string) *DefaultManager {
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
// labeled with the scaledown scope label, reads the stored replica count, restores it,
// and removes the replicas label.
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

		deploy.Spec.Replicas = new(int32(replicas))
		delete(deploy.Labels, labelScaledownReplicas)

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

		sts.Spec.Replicas = new(int32(replicas))
		delete(sts.Labels, labelScaledownReplicas)

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

		rs.Spec.Replicas = new(int32(replicas))
		delete(rs.Labels, labelScaledownReplicas)

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

		rc.Spec.Replicas = new(int32(replicas))
		delete(rc.Labels, labelScaledownReplicas)

		if lErr := m.k8sClient.Update(ctx, &rc); lErr != nil {
			return fmt.Errorf("failed to scale up replicationcontroller %s: %w", rc.Name, lErr)
		}
	}

	return nil
}

func zeroReplicas() *int32 {
	return new(int32(0))
}
