package scheduled_backup_creator

import (
	"context"
	"fmt"
	"github.com/cloudogu/k8s-backup-operator/pkg/api/ecosystem"
	v1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"strconv"
)

func main() {
	ctx := context.Background()
	logger := log.FromContext(ctx)

	backupScheduleName := os.Args[0]
	logger.Info(fmt.Sprintf("start schedule backup creator from backup schedule: %s", backupScheduleName))

	restConfig := ctrl.GetConfigOrDie()
	k8sClientSet, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		handleError(ctx, fmt.Errorf("unable to create k8s clientset: %w", err))
	}

	ecosystemClientSet, err := ecosystem.NewClientSet(restConfig, k8sClientSet)
	if err != nil {
		handleError(ctx, fmt.Errorf("unable to create ecosystem clientset: %w", err))
	}

	namespace := os.Getenv("NAMESPACE")
	scheduleResource, err := ecosystemClientSet.EcosystemV1Alpha1().BackupSchedules(namespace).Get(ctx, backupScheduleName, metav1.GetOptions{})
	if err != nil {
		handleError(ctx, fmt.Errorf("unable to get backub schedule resource with name %s: %w", backupScheduleName, err))
	}

	backupName := backupScheduleName + strconv.Itoa(scheduleResource.Status.BackupNumber)
	scheduleResource.Status.BackupNumber += 1

	_, err = ecosystemClientSet.EcosystemV1Alpha1().BackupSchedules(namespace).Update(ctx, scheduleResource, metav1.UpdateOptions{})
	if err != nil {
		handleError(ctx, fmt.Errorf("failed to update backup number in backup schedule %s: %w", backupScheduleName, err))
	}

	backupResource := &v1.Backup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      backupName,
			Namespace: namespace,
		},
		Spec: v1.BackupSpec{Provider: scheduleResource.Spec.Provider},
	}

	_, err = ecosystemClientSet.EcosystemV1Alpha1().Backups(namespace).Create(ctx, backupResource, metav1.CreateOptions{})
	if err != nil {
		handleError(ctx, fmt.Errorf("failed to create backup resource %s: %w", backupName, err))
	}
}

func handleError(ctx context.Context, err error) {
	logger := log.FromContext(ctx)
	logger.Error(err, "scheduled backup creator failed")
	os.Exit(1)
}
