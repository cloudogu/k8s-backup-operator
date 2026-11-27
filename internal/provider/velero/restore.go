package velero

import (
	"context"
	"fmt"
	"slices"

	"github.com/cloudogu/k8s-backup-operator/internal/logging"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	k8sv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
)

const (
	// RestoreSourceNameLabel names the Cloudogu Restore a Velero restore was created for.
	// It is secondary evidence for debugging and for listing children without a cache index
	RestoreSourceNameLabel = "k8s.cloudogu.com/restore-source-name"
	// RestoreSourceUIDLabel carries the UID of the Cloudogu Restore a Velero restore was
	// created for.
	RestoreSourceUIDLabel = "k8s.cloudogu.com/restore-source-uid"
	// restoreDoguModifierConfigMapName is the resource modifier applied to every Velero restore
	// this operator creates.
	restoreDoguModifierConfigMapName = "k8s-backup-operator-restore-dogu-modifier"
	// restoreKind is the kind of the Cloudogu Restore, used in the child's owner reference.
	restoreKind = "Restore"
)

// ConflictError reports an existing Velero restore at the expected name that this
// operator may not use.
type ConflictError struct {
	name   string
	reason string
}

func (e *ConflictError) Error() string {
	return fmt.Sprintf("velero restore [%s] conflicts with the expected child: %s", e.name, e.reason)
}

// BuildRestore returns the Velero restore that the given Cloudogu Restore expects. The child
// takes the parent's name and namespace, so two Cloudogu Restores can never collide on it.
func BuildRestore(cesRestore *k8sv1.Restore) *velerov1.Restore {
	coreAPIGroup := ""
	isController := true

	return &velerov1.Restore{
		Name:      cesRestore.Name,
		Namespace: cesRestore.Namespace,
		Labels: map[string]string{
			RestoreSourceNameLabel: cesRestore.Name,
			RestoreSourceUIDLabel:  string(cesRestore.UID),
		},
		OwnerReferences: []metav1.OwnerReference{{
			APIVersion:         k8sv1.GroupVersion.String(),
			Kind:               restoreKind,
			Name:               cesRestore.Name,
			UID:                cesRestore.UID,
			Controller:         &isController,
			BlockOwnerDeletion: &isController,
		}},
		Spec: velerov1.RestoreSpec{
			BackupName:             cesRestore.Spec.BackupName,
			ExistingResourcePolicy: velerov1.PolicyTypeUpdate,
			ResourceModifier: &corev1.TypedLocalObjectReference{
				APIGroup: &coreAPIGroup,
				Kind:     "ConfigMap",
				Name:     restoreDoguModifierConfigMapName,
			},
		},
	}
}

// GetLegacyRestore reads the Velero restore at the expected name of the given Cloudogu Restore. An
// absent child is not an error.
func GetLegacyRestore(ctx context.Context, k8sClient client.Client, cesRestore *k8sv1.Restore) (*velerov1.Restore, error) {
	return GetRestoreByName(ctx, k8sClient, cesRestore.Namespace, cesRestore.Name)
}

// GetRestoreByName reads the Velero restore with the given name. An absent restore is not an error.
func GetRestoreByName(ctx context.Context, k8sClient client.Client, namespace, name string) (*velerov1.Restore, error) {
	veleroRestore := &velerov1.Restore{}
	err := k8sClient.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, veleroRestore)
	if apierrors.IsNotFound(err) {
		return nil, nil // has yet to be created
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get velero restore [%s]: %w", name, err)
	}

	return veleroRestore, nil
}

// HasOwnedRestore reports whether any Velero restore with the UID label of the given Cloudogu Restore is controlled by it.
func HasOwnedRestore(ctx context.Context, k8sClient client.Client, cesRestore *k8sv1.Restore) (bool, error) {
	veleroRestores := &velerov1.RestoreList{}
	err := k8sClient.List(ctx, veleroRestores, client.InNamespace(cesRestore.Namespace),
		client.MatchingLabels{RestoreSourceUIDLabel: string(cesRestore.UID)})
	if err != nil {
		return false, fmt.Errorf("failed to list velero restores of [%s]: %w", cesRestore.Name, err)
	}

	return slices.ContainsFunc(veleroRestores.Items, func(veleroRestore velerov1.Restore) bool {
		return IsOwnedRestore(cesRestore, &veleroRestore)
	}), nil
}

// IsOwnedRestore reports whether the given Velero restore is controlled by the given Cloudogu Restore.
func IsOwnedRestore(cesRestore *k8sv1.Restore, veleroRestore *velerov1.Restore) bool {
	controller := metav1.GetControllerOf(veleroRestore)

	return controller != nil && controller.UID != "" && controller.UID == cesRestore.UID
}

// CheckRestoreForConflicts reports whether an existing velero restore is from a given Cloudogu Restore.
// It returns a *ConflictError when it is not.
func CheckRestoreForConflicts(cesRestore *k8sv1.Restore, veleroRestore *velerov1.Restore, expectedBackupName string) error {
	if !IsOwnedRestore(cesRestore, veleroRestore) {
		return &ConflictError{
			name:   veleroRestore.Name,
			reason: fmt.Sprintf("it is not controlled by restore [%s] and must not be adopted", cesRestore.Name),
		}
	}

	if veleroRestore.Spec.BackupName != expectedBackupName {
		return &ConflictError{
			name: veleroRestore.Name,
			reason: fmt.Sprintf("it restores backup [%s] but restore [%s] expects backup [%s]",
				veleroRestore.Spec.BackupName, cesRestore.Name, expectedBackupName),
		}
	}

	return nil
}

// EnsureRestore returns the Velero restore of the given Cloudogu Restore, creating it exactly
// once. An existing own child is returned without any write, which is what makes a repeated attempt
// after a crash between child creation and parent status persistence safe. An existing child that is
// not usable yields a *ConflictError and is neither deleted, mutated nor claimed.
func EnsureRestore(ctx context.Context, k8sClient client.Client, cesRestore *k8sv1.Restore) (*velerov1.Restore, error) {

	existing, err := GetLegacyRestore(ctx, k8sClient, cesRestore)
	if err != nil {
		return nil, err
	}

	if existing != nil {
		if conflictErr := CheckRestoreForConflicts(cesRestore, existing, cesRestore.Spec.BackupName); conflictErr != nil {
			return nil, conflictErr
		}

		logging.Info(ctx, fmt.Sprintf("velero restore [%s] already exists and is ours: reuse it", existing.Name))

		return existing, nil
	}

	veleroRestore := BuildRestore(cesRestore)
	err = k8sClient.Create(ctx, veleroRestore)
	if err != nil {
		return nil, fmt.Errorf("failed to create velero restore [%s]: %w", veleroRestore.Name, err)
	}

	return veleroRestore, nil
}

// DeleteRestore deletes a Velero restore that the caller already classified as safe to delete.
// The UID precondition prevents a namesake that replaced the classified child from being deleted.
// An already absent child is not an error.
func DeleteRestore(ctx context.Context, k8sClient client.Client, veleroRestore *velerov1.Restore) error {
	uid := veleroRestore.UID
	err := k8sClient.Delete(ctx, veleroRestore, client.Preconditions{UID: &uid})
	if apierrors.IsNotFound(err) {
		logging.Info(ctx, fmt.Sprintf("velero restore [%s] is already gone: ignore", veleroRestore.Name))

		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to delete velero restore [%s]: %w", veleroRestore.Name, err)
	}

	return nil
}
