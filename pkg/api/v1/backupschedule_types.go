/*
This file was generated with "make generate".
*/

package v1

import (
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	BackupScheduleStatusNew      = ""
	BackupScheduleStatusFailed   = "failed"
	BackupScheduleStatusDeleting = "deleting"
	BackupScheduleStatusUpdating = "updating"
	BackupScheduleStatusCreating = "creating"
	BackupScheduleStatusCreated  = "created"
)

const BackupScheduleFinalizer = "cloudogu-backup-schedule-finalizer"

const ProviderEnvVar = "PROVIDER"

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// BackupScheduleSpec defines the desired state of BackupSchedule
type BackupScheduleSpec struct {
	// Important: Run "make" to regenerate code after modifying this file

	// Schedule is a cron expression defining when to run the backup.
	Schedule string `json:"schedule,omitempty"`
	// Provider defines the backup provider which should be used for the scheduled backups.
	Provider Provider `json:"provider,omitempty"`
}

// BackupScheduleStatus defines the observed state of BackupSchedule
// +kubebuilder:object:generate=false
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

func (bs *BackupSchedule) CronJobPodTemplate(kubectlImage string) corev1.PodTemplateSpec {
	return corev1.PodTemplateSpec{
		ObjectMeta: cronJobPodMeta(bs.Namespace),
		Spec:       bs.cronJobPodSpec(kubectlImage),
	}
}

func cronJobPodMeta(namespace string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      "scheduled-backup-creator",
		Namespace: namespace,
		Labels: map[string]string{
			"app":                          "ces",
			"k8s.cloudogu.com/part-of":     "backup",
			"app.kubernetes.io/created-by": "k8s-backup-operator",
			"app.kubernetes.io/part-of":    "k8s-backup-operator",
		},
	}
}

func (bs *BackupSchedule) cronJobPodSpec(kubectlImage string) corev1.PodSpec {
	mode := int32(0550)
	volumeName := "k8s-backup-operator-create-backup-script"
	scriptPath := "/bin/entrypoint.sh"
	return corev1.PodSpec{
		Containers: []corev1.Container{{
			Name:            bs.CronJobName(),
			Image:           kubectlImage,
			ImagePullPolicy: corev1.PullIfNotPresent,
			Command:         []string{scriptPath},
			Env:             bs.cronJobEnvVars(),
			VolumeMounts: []corev1.VolumeMount{{
				Name:      volumeName,
				ReadOnly:  true,
				MountPath: scriptPath,
				SubPath:   "entrypoint.sh",
			}},
		}},
		RestartPolicy:      corev1.RestartPolicyOnFailure,
		ServiceAccountName: "k8s-backup-operator-scheduled-backup-creator-manager",
		Volumes: []corev1.Volume{{
			Name: volumeName,
			VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: "k8s-create-backup-script"},
				DefaultMode:          &mode,
			}},
		}},
	}
}

func (bs *BackupSchedule) CronJobName() string {
	return fmt.Sprintf("backup-schedule-%s", bs.Name)
}

func (bs *BackupSchedule) cronJobEnvVars() []corev1.EnvVar {
	return []corev1.EnvVar{
		{Name: "NAMESPACE",
			ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.namespace"}}},
		{Name: "SCHEDULED_BACKUP_NAME", Value: bs.Name},
		{Name: ProviderEnvVar, Value: string(bs.Spec.Provider)}}
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
