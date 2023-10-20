package v1

import (
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

const (
	RestoreStatusNew        = ""
	RestoreStatusInProgress = "in progress"
	RestoreStatusFailed     = "failed"
	RestoreStatusCompleted  = "completed"
	RestoreStatusDeleting   = "deleting"
)

const RestoreFinalizer = "cloudogu-restore-finalizer"

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// RestoreSpec defines the desired state of Restore
type RestoreSpec struct {
	// BackupName references the backup that should be restored.
	BackupName string `json:"backupName,omitempty"`
	// Provider defines the backup provider which should be used for the restore.
	Provider Provider `json:"provider,omitempty"`
}

// RestoreStatus defines the observed state of Restore
type RestoreStatus struct {
	// Status represents the state of the backup.
	Status string `json:"status,omitempty"`
	// RequeueTimeNanos contains the time in nanoseconds to wait until the next requeue.
	RequeueTimeNanos time.Duration `json:"requeueTimeNanos,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:metadata:labels=app=ces;app.kubernetes.io/name=k8s-backup-operator;k8s.cloudogu.com/part-of=backup

// Restore is the Schema for the restores API
type Restore struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired state of Restore
	Spec RestoreSpec `json:"spec,omitempty"`
	// Status defines the observed state of Restore
	Status RestoreStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RestoreList contains a list of Restore
type RestoreList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Restore `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Restore{}, &RestoreList{})
}

func (r *Restore) GetFieldSelectorWithName() string {
	return fmt.Sprintf("metadata.name=%s", r.Name)
}
