/*
This file was generated with "make generate".
*/

package v1

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	BackupScheduleStatusFailed   = "failed"
	BackupScheduleStatusDeleting = "deleting"
	BackupScheduleStatusUpdating = "updating"
	BackupScheduleStatusCreating = "creating"
	BackupScheduleStatusCreated  = "created"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// BackupScheduleSpec defines the desired state of BackupSchedule
type BackupScheduleSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of BackupSchedule. Edit backupschedule_types.go to remove/update
	Foo string `json:"foo,omitempty"`
}

// BackupScheduleStatus defines the observed state of BackupSchedule
type BackupScheduleStatus struct {
	// Status represents the state of the backup.
	Status string `json:"status,omitempty"`
	// RequeueTimeNanos contains the time in nanoseconds to wait until the next requeue.
	RequeueTimeNanos time.Duration `json:"requeueTimeNanos,omitempty"`
}

// GetRequeueTimeNanos returns the requeue time in nano seconds.
func (bss BackupScheduleStatus) GetRequeueTimeNanos() time.Duration {
	return bss.RequeueTimeNanos
}

// GetStatus return the status from the object.
func (bss BackupScheduleStatus) GetStatus() string {
	return bss.Status
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// BackupSchedule is the Schema for the backupschedules API
type BackupSchedule struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BackupScheduleSpec   `json:"spec,omitempty"`
	Status BackupScheduleStatus `json:"status,omitempty"`
}

// GetStatus return the requeueable status.
func (bs *BackupSchedule) GetStatus() RequeueableStatus {
	return bs.Status
}

//+kubebuilder:object:root=true

// BackupScheduleList contains a list of BackupSchedule
type BackupScheduleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BackupSchedule `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BackupSchedule{}, &BackupScheduleList{})
}