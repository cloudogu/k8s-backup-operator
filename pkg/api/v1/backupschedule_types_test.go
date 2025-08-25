package v1

import (
	"github.com/cloudogu/k8s-backup-operator/pkg/config"
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
	t.Run("should use pullifpresent on production", func(t *testing.T) {
		// given
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
				Containers: []corev1.Container{{
					Name:            "backup-schedule-my-schedule",
					Image:           "bitnami/legacy/kubectl:1.27.7",
					ImagePullPolicy: corev1.PullIfNotPresent,
					Args:            []string{"scheduled-backup", "--name=my-schedule", "--provider=velero"},
					Env: []corev1.EnvVar{
						{Name: "NAMESPACE", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.namespace"}}},
					},
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
		actual := sut.CronJobPodTemplate("bitnami/legacy/kubectl:1.27.7")

		// then
		assert.Equal(t, expected, actual)
	})
	t.Run("should use pullalways in development", func(t *testing.T) {
		// given
		oldStage := config.Stage
		defer func() {
			config.Stage = oldStage
		}()

		config.Stage = config.StageDevelopment

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
				Containers: []corev1.Container{{
					Name:            "backup-schedule-my-schedule",
					Image:           "bitnami/legacy/kubectl:1.27.7",
					ImagePullPolicy: corev1.PullAlways,
					Args:            []string{"scheduled-backup", "--name=my-schedule", "--provider=velero"},
					Env: []corev1.EnvVar{
						{Name: "NAMESPACE", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.namespace"}}},
					},
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
		actual := sut.CronJobPodTemplate("bitnami/legacy/kubectl:1.27.7")

		// then
		assert.Equal(t, expected, actual)
	})
}
