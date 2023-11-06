package v1

import (
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"time"
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
	DeleteEventReason        = "Delete"
	UpdateEventReason        = "Update"
	ErrorOnCreateEventReason = "ErrCreation"
)

const (
	ProviderSelectEventReason        = "Provider selection"
	ProviderDeleteEventReason        = "Provider delete"
	ErrorOnProviderDeleteEventReason = "Error provider delete"
)

const BackupFinalizer = "cloudogu-backup-finalizer"

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// BackupSpec defines the desired state of Backup
type BackupSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Provider defines the backup provider which should be used for the backup.
	Provider Provider `json:"provider,omitempty"`
}

// BackupStatus defines the observed state of Backup
// +kubebuilder:object:generate=false
type BackupStatus struct {
	// Status represents the state of the backup.
	Status string `json:"status,omitempty"`
	// RequeueTimeNanos contains the time in nanoseconds to wait until the next requeue.
	RequeueTimeNanos time.Duration `json:"requeueTimeNanos,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:metadata:labels=app=ces;app.kubernetes.io/name=k8s-backup-operator;k8s.cloudogu.com/part-of=backup

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

// GetFieldSelectorWithName returns the field selector with the metadata.name property.
func (b *Backup) GetFieldSelectorWithName() string {
	return fmt.Sprintf("metadata.name=%s", b.Name)
}

// RequeuableObject provides provides functionalities used for an abstract requeueHandler
// +kubebuilder:object:generate=false
type RequeuableObject interface {
	runtime.Object
	// GetStatus returns the status from the object.
	GetStatus() RequeueableStatus
	// GetName returns the name from the object.
	GetName() string
}

// RequeueableStatus provides functionalities used for an abstract requeueHandler
// +kubebuilder:object:generate=false
type RequeueableStatus interface {
	// GetRequeueTimeNanos returns the requeue time in nano seconds.
	GetRequeueTimeNanos() time.Duration
	// GetStatus return the status from the object.
	GetStatus() string
}

// GetStatus return the requeueable status.
func (b *Backup) GetStatus() RequeueableStatus {
	return b.Status
}

// GetStatus return the status from the status object.
func (bs BackupStatus) GetStatus() string {
	return bs.Status
}

// GetRequeueTimeNanos returns the requeue time in nano seconds.
func (bs BackupStatus) GetRequeueTimeNanos() time.Duration {
	return bs.RequeueTimeNanos
}
