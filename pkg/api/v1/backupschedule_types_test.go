package v1

import (
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
	"time"
)

const testNamespace = "test-ns"

func TestBackupScheduleStatus_GetRequeueTimeNanos(t *testing.T) {
	// given
	sut := BackupScheduleStatus{RequeueTimeNanos: 1234}

	// when
	actual := sut.GetRequeueTimeNanos()

	// then
	assert.Equal(t, time.Duration(1234), actual)
}

func TestBackupScheduleStatus_GetStatus(t *testing.T) {
	// given
	sut := BackupScheduleStatus{Status: BackupScheduleStatusCreating}

	// when
	actual := sut.GetStatus()

	// then
	assert.Equal(t, "creating", actual)
}

func TestBackupSchedule_GetStatus(t *testing.T) {
	// given
	expectedStatus := BackupScheduleStatus{
		Status:           BackupScheduleStatusCreated,
		RequeueTimeNanos: 54321,
	}
	sut := &BackupSchedule{Status: expectedStatus}

	// when
	actual := sut.GetStatus()

	// then
	assert.Equal(t, expectedStatus, actual)
}

func TestBackupSchedule_CronJobPodTemplate(t *testing.T) {
	// given
	mode := int32(0550)
	expected := corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "scheduled-backup-creator",
			Namespace: testNamespace,
			Labels: map[string]string{
				"app":                          "ces",
				"k8s.cloudogu.com/part-of":     "backup",
				"app.kubernetes.io/created-by": "k8s-backup-operator",
				"app.kubernetes.io/part-of":    "k8s-backup-operator",
			},
		},
		Spec: corev1.PodSpec{
			Volumes: []corev1.Volume{{
				Name: "k8s-backup-operator-create-backup-script",
				VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: "k8s-create-backup-script"},
					DefaultMode:          &mode,
				}},
			}},
			Containers: []corev1.Container{{
				Name:            "backup-schedule-my-schedule",
				Image:           "bitnami/kubectl:1.27.7",
				ImagePullPolicy: corev1.PullIfNotPresent,
				Command:         []string{"/bin/entrypoint.sh"},
				Env: []corev1.EnvVar{
					{Name: "NAMESPACE", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.namespace"}}},
					{Name: "SCHEDULED_BACKUP_NAME", Value: "my-schedule"},
					{Name: "PROVIDER", Value: "velero"},
				},
				VolumeMounts: []corev1.VolumeMount{{
					Name:      "k8s-backup-operator-create-backup-script",
					ReadOnly:  true,
					MountPath: "/bin/entrypoint.sh",
					SubPath:   "entrypoint.sh",
				}},
			}},
			RestartPolicy:      corev1.RestartPolicyOnFailure,
			ServiceAccountName: "k8s-backup-operator-scheduled-backup-creator-manager",
		},
	}
	sut := &BackupSchedule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-schedule",
			Namespace: testNamespace,
		},
		Spec: BackupScheduleSpec{
			Schedule: "* * * * *",
			Provider: "velero",
		},
	}

	// when
	actual := sut.CronJobPodTemplate("bitnami/kubectl:1.27.7")

	// then
	assert.Equal(t, expected, actual)
}
