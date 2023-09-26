package v1

import (
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	BackupStatusNew        = ""
	BackupStatusInProgress = "in progress"
	BackupStatusCompleted  = "completed"
	BackupStatusDeleting   = "deleting"
	BackupStatusFailed     = "failed"
)

type Provider string

const (
	ProviderVelero = "velero"
)

const (
	CreateEventReason        = "Creation"
	ErrorOnCreateEventReason = "ErrCreation"
)

const (
	ProviderDeleteEventReason        = "Provider delete"
	ErrorOnProviderDeleteEventReason = "Error provider delete"
)

const BackupFinalizer = "backup-finalizer"

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// BackupSpec defines the desired state of Backup
type BackupSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Provider defines the backup provider which should be used for the backup.
	Provider Provider `json:"provider,omitempty"`
}

// BackupStatus defines the observed state of Backup
type BackupStatus struct {
	Status string `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:metadata:labels=app=ces;app.kubernetes.io/name=k8s-backup-operator

// Backup is the Schema for the backups API
type Backup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired state of Backup
	Spec BackupSpec `json:"spec,omitempty"`
	// Status defines the observed state of Backup
	Status BackupStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// BackupList contains a list of Backup
type BackupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Backup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Backup{}, &BackupList{})
}

func (b *Backup) GetFieldSelectorWithName() string {
	return fmt.Sprintf("metadata.name=%s", b.Name)
}
