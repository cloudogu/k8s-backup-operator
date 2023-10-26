package backupschedule

import (
	"context"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"

	k8sv1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
)

// backupScheduleReconciler reconciles a BackupSchedule object
type backupScheduleReconciler struct {
	clientSet      ecosystemInterface
	recorder       eventRecorder
	requeueHandler requeueHandler
}

func NewBackupScheduleReconciler(clientSet ecosystemInterface, recorder eventRecorder, requeueHandler requeueHandler) *backupScheduleReconciler {
	return &backupScheduleReconciler{clientSet: clientSet, recorder: recorder, requeueHandler: requeueHandler}
}

//+kubebuilder:rbac:groups=k8s.cloudogu.com,resources=backupschedules,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=k8s.cloudogu.com,resources=backupschedules/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=k8s.cloudogu.com,resources=backupschedules/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the BackupSchedule object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.15.0/pkg/reconcile
func (r *backupScheduleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// TODO(user): your logic here

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *backupScheduleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8sv1.BackupSchedule{}).
		Complete(r)
}
