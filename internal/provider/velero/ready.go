package velero

import (
	"context"
	"fmt"

	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// ReasonVeleroProviderReady reports a provider that can serve backup and restore requests
	ReasonVeleroProviderReady = "VeleroProviderReady"
	// ReasonVeleroBackupStorageLocationNotFound reports a missing backup storage location, which usually
	// means that Velero is not installed or not configured yet.
	ReasonVeleroBackupStorageLocationNotFound = "VeleroBackupStorageLocationNotFound"
	// ReasonVeleroBackupStorageLocationNotAvailable reports a backup storage location that Velero cannot reach.
	ReasonVeleroBackupStorageLocationNotAvailable = "VeleroBackupStorageLocationNotAvailable"
	// ReasonVeleroDeploymentNotFound reports a missing velero deployment, which usually means that Velero is
	// not installed or that it is deployed under a different name than configured.
	ReasonVeleroDeploymentNotFound = "VeleroDeploymentNotFound"
	// ReasonVeleroDeploymentNotReady reports a velero deployment without a single ready replica
	ReasonVeleroDeploymentNotReady = "VeleroDeploymentNotReady"
)

// Readiness reports whether the provider can serve backup and restore requests. Reason and Message
// are written to the conditions of the backup or restore that waits for the provider, so they
// describe the provider state rather than the resource that observed it.
type Readiness struct {
	Ready   bool
	Reason  string
	Message string
}

// CheckReady reports whether the provider is ready to serve backup and restore requests. A provider
// that is not ready yet is reported through the Readiness, API server errors are returned as an error.
func CheckReady(ctx context.Context, k8sClient client.Client, namespace string, backupStorageName string, deploymentName string) (Readiness, error) {
	backupStorageLocation := &velerov1.BackupStorageLocation{}
	err := k8sClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: backupStorageName}, backupStorageLocation)
	if apierrors.IsNotFound(err) {
		return Readiness{
			Reason:  ReasonVeleroBackupStorageLocationNotFound,
			Message: fmt.Sprintf("The velero backup storage location 'name=%s' was not found.", backupStorageName),
		}, nil
	}
	if err != nil {
		return Readiness{}, fmt.Errorf("get velero backup storage location 'name=%s': %w", backupStorageName, err)
	}

	if backupStorageLocation.Status.Phase != velerov1.BackupStorageLocationPhaseAvailable {
		return Readiness{
			Reason: ReasonVeleroBackupStorageLocationNotAvailable,
			Message: fmt.Sprintf("The velero backup storage location 'name=%s' is not available (phase: %s).",
				backupStorageName, backupStorageLocation.Status.Phase),
		}, nil
	}

	return checkDeploymentReady(ctx, k8sClient, namespace, backupStorageName, deploymentName)
}

// checkDeploymentReady reports whether the velero deployment has at least one ready replica.
func checkDeploymentReady(ctx context.Context, k8sClient client.Client, namespace string, backupStorageName string, deploymentName string) (Readiness, error) {
	deployment := &appsv1.Deployment{}
	err := k8sClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: deploymentName}, deployment)
	if apierrors.IsNotFound(err) {
		return Readiness{
			Reason:  ReasonVeleroDeploymentNotFound,
			Message: fmt.Sprintf("The velero deployment 'name=%s' was not found in namespace '%s'.", deploymentName, namespace),
		}, nil
	}
	if err != nil {
		return Readiness{}, fmt.Errorf("get velero deployment 'name=%s': %w", deploymentName, err)
	}

	if deployment.Status.ReadyReplicas < 1 {
		return Readiness{
			Reason: ReasonVeleroDeploymentNotReady,
			Message: fmt.Sprintf("The velero deployment 'name=%s' has no ready replica (readyReplicas: %d).",
				deploymentName, deployment.Status.ReadyReplicas),
		}, nil
	}

	return Readiness{
		Ready:   true,
		Reason:  ReasonVeleroProviderReady,
		Message: fmt.Sprintf("The velero backup storage location 'name=%s' is available and the velero deployment 'name=%s' is ready.", backupStorageName, deploymentName),
	}, nil
}
