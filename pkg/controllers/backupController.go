package controllers

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// backupReconciler watches every Service object in the cluster and creates ingress objects accordingly.
type backupReconciler struct {
	client   client.Client
	recorder record.EventRecorder
}

// NewBackupReconciler creates a new backup reconciler.
func NewBackupReconciler(client client.Client, recorder record.EventRecorder) *backupReconciler {
	return &backupReconciler{
		client:   client,
		recorder: recorder,
	}
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *backupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconcile this backup manager")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *backupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Service{}).
		Complete(r)
}
