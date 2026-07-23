package schedule

import (
	"fmt"
	"testing"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	testNamespace    = "test-namespace"
	testScheduleName = "test-schedule"
	testCronJobName  = testScheduleName
	testScheduleCron = "*/5 * * * *"
	testProvider     = "velero"
)

func TestNewReconciler(t *testing.T) {
	t.Run("should create new reconciler", func(t *testing.T) {
		// given
		fakeClient := fake.NewClientBuilder().Build()

		// when
		reconciler := NewReconciler(fakeClient)

		// then
		require.NotNil(t, reconciler)
		assert.IsType(t, &defaultReconciler{}, reconciler)
	})
}

// Helper functions

func createTestBackupSchedule() *backupv1.BackupSchedule {
	return &backupv1.BackupSchedule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testScheduleName,
			Namespace: testNamespace,
		},
		Spec: backupv1.BackupScheduleSpec{
			Schedule: testScheduleCron,
			Provider: testProvider,
		},
		Status: backupv1.BackupScheduleStatus{},
	}
}

func createTestCronJob(name, schedule string) *batchv1.CronJob {
	return &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			// TODO: get prefix from lib
			Name:      fmt.Sprintf("backup-schedule-%s", name),
			Namespace: testNamespace,
		},
		Spec: batchv1.CronJobSpec{
			Schedule: schedule,
			JobTemplate: batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "backup-container",
									Image: "backup-operator:latest",
								},
							},
							RestartPolicy: corev1.RestartPolicyOnFailure,
						},
					},
				},
			},
		},
	}
}

func createScheme(t *testing.T) *runtime.Scheme {
	t.Helper()

	scheme := runtime.NewScheme()
	err := batchv1.AddToScheme(scheme)
	require.NoError(t, err)
	err = backupv1.AddToScheme(scheme)
	require.NoError(t, err)
	err = corev1.AddToScheme(scheme)
	require.NoError(t, err)

	return scheme
}
