package velero

import (
	"context"
	"fmt"

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
func BuildRestore(parent *k8sv1.Restore) *velerov1.Restore {
	coreAPIGroup := ""
	isController := true

	return &velerov1.Restore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      parent.Name,
			Namespace: parent.Namespace,
			Labels: map[string]string{
				RestoreSourceNameLabel: parent.Name,
				RestoreSourceUIDLabel:  string(parent.UID),
			},
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion:         k8sv1.GroupVersion.String(),
				Kind:               restoreKind,
				Name:               parent.Name,
				UID:                parent.UID,
				Controller:         &isController,
				BlockOwnerDeletion: &isController,
			}},
		},
		Spec: velerov1.RestoreSpec{
			BackupName:             parent.Spec.BackupName,
			ExistingResourcePolicy: velerov1.PolicyTypeUpdate,
			ResourceModifier: &corev1.TypedLocalObjectReference{
				APIGroup: &coreAPIGroup,
				Kind:     "ConfigMap",
				Name:     restoreDoguModifierConfigMapName,
			},
		},
	}
}

// GetRestore reads the Velero restore at the expected name of the given Cloudogu Restore. An
// absent child is not an error.
func GetRestore(ctx context.Context, k8sClient client.Client, parent *k8sv1.Restore) (*velerov1.Restore, error) {
	child := &velerov1.Restore{}
	err := k8sClient.Get(ctx, client.ObjectKeyFromObject(parent), child)
	if apierrors.IsNotFound(err) {
		return nil, nil // has yet to be created
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get velero restore [%s]: %w", parent.Name, err)
	}

	return child, nil
}

// IsOwnedRestore reports whether the given Velero restore is controlled by the given Cloudogu Restore.
func IsOwnedRestore(parent *k8sv1.Restore, child *velerov1.Restore) bool {
	controller := metav1.GetControllerOf(child)

	return controller != nil && controller.UID != "" && controller.UID == parent.UID
}

// CheckRestoreForConflicts reports whether an existing child may be used as the given Cloudogu
// Restore's child. It returns a *ConflictError when it may not, and never proposes a
// write: the child's spec is never mutated, because Velero has already acted on it.
func CheckRestoreForConflicts(parent *k8sv1.Restore, child *velerov1.Restore) error {
	if !IsOwnedRestore(parent, child) {
		return &ConflictError{
			name:   child.Name,
			reason: fmt.Sprintf("it is not controlled by restore [%s] and must not be adopted", parent.Name),
		}
	}

	if child.Spec.BackupName != parent.Spec.BackupName {
		return &ConflictError{
			name: child.Name,
			reason: fmt.Sprintf("it restores backup [%s] but restore [%s] expects backup [%s]",
				child.Spec.BackupName, parent.Name, parent.Spec.BackupName),
		}
	}

	return nil
}

// EnsureRestore returns the Velero restore of the given Cloudogu Restore, creating it exactly
// once. An existing own child is returned without any write, which is what makes a repeated attempt
// after a crash between child creation and parent status persistence safe. An existing child that is
// not usable yields a *ConflictError and is neither deleted, mutated nor claimed.
func EnsureRestore(ctx context.Context, k8sClient client.Client, parent *k8sv1.Restore) (*velerov1.Restore, error) {

	existing, err := GetRestore(ctx, k8sClient, parent)
	if err != nil {
		return nil, err
	}

	if existing != nil {
		if conflictErr := CheckRestoreForConflicts(parent, existing); conflictErr != nil {
			return nil, conflictErr
		}

		logging.Info(ctx, fmt.Sprintf("velero restore [%s] already exists and is ours: reuse it", existing.Name))

		return existing, nil
	}

	child := BuildRestore(parent)
	err = k8sClient.Create(ctx, child)
	if err != nil {
		return nil, fmt.Errorf("failed to create velero restore [%s]: %w", child.Name, err)
	}

	return child, nil
}

// DeleteRestore deletes a Velero restore that the caller already classified as safe to delete.
// The UID precondition prevents a namesake that replaced the classified child from being deleted.
// An already absent child is not an error.
func DeleteRestore(ctx context.Context, k8sClient client.Client, child *velerov1.Restore) error {

	uid := child.UID
	err := k8sClient.Delete(ctx, child, client.Preconditions{UID: &uid})
	if apierrors.IsNotFound(err) {
		logging.Info(ctx, fmt.Sprintf("velero restore [%s] is already gone: ignore", child.Name))

		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to delete velero restore [%s]: %w", child.Name, err)
	}

	return nil
}
