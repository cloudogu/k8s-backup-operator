package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/cloudogu/k8s-backup-operator/pkg/additionalimages"
	"github.com/cloudogu/k8s-backup-operator/pkg/backupschedule"
	"github.com/cloudogu/k8s-backup-operator/pkg/cleanup"
	"github.com/cloudogu/k8s-backup-operator/pkg/requeue"
	"os"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/cloudogu/cesapp-lib/core"
	reg "github.com/cloudogu/cesapp-lib/registry"
	"github.com/cloudogu/k8s-backup-operator/pkg/api/ecosystem"
	k8sv1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
	"github.com/cloudogu/k8s-backup-operator/pkg/backup"
	"github.com/cloudogu/k8s-backup-operator/pkg/config"
	"github.com/cloudogu/k8s-backup-operator/pkg/restore"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

var (
	// Version of the application
	Version = "0.0.0"
)

var (
	leaseDuration = time.Second * 60
	renewDeadline = time.Second * 40
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(k8sv1.AddToScheme(scheme))

	// +kubebuilder:scaffold:scheme
}

func main() {
	err := startOperator()
	if err != nil {
		setupLog.Error(err, "failed to start operator")
		os.Exit(1)
	}
}

func startOperator() error {
	operatorConfig, err := config.NewOperatorConfig(Version)
	if err != nil {
		return fmt.Errorf("unable to create operator config: %w", err)
	}

	options := getK8sManagerOptions(operatorConfig)
	k8sManager, err := ctrl.NewManager(ctrl.GetConfigOrDie(), options)
	if err != nil {
		return fmt.Errorf("unable to start manager: %w", err)
	}

	err = configureManager(k8sManager, operatorConfig)
	if err != nil {
		return fmt.Errorf("unable to configure manager: %w", err)
	}

	return startK8sManager(k8sManager)
}

func configureManager(k8sManager controllerManager, operatorConfig *config.OperatorConfig) error {
	err := configureReconcilers(k8sManager, operatorConfig)
	if err != nil {
		return fmt.Errorf("unable to configure reconciler: %w", err)
	}

	err = addChecks(k8sManager)
	if err != nil {
		return fmt.Errorf("unable to add checks to the manager: %w", err)
	}

	return nil
}

func getK8sManagerOptions(operatorConfig *config.OperatorConfig) ctrl.Options {
	controllerOpts := ctrl.Options{
		Scheme: scheme,
		Cache: cache.Options{DefaultNamespaces: map[string]cache.Config{
			operatorConfig.Namespace: {},
		}},
		WebhookServer:    webhook.NewServer(webhook.Options{Port: 9443}),
		LeaderElectionID: "e3f6c1a7.cloudogu.com",
		LeaseDuration:    &leaseDuration,
		RenewDeadline:    &renewDeadline,
	}
	var zapOpts zap.Options
	controllerOpts, zapOpts = parseFlags(controllerOpts)

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&zapOpts)))

	return controllerOpts
}

// parseFlags is a closure because it panics when its called twice in tests.
// Therefore, we must overwrite it for all but one single test.
var parseFlags = func(ctrlOpts ctrl.Options) (ctrl.Options, zap.Options) {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	zapOpts := zap.Options{
		Development: config.IsStageDevelopment(),
	}
	zapOpts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrlOpts.Metrics = server.Options{BindAddress: metricsAddr}
	ctrlOpts.HealthProbeBindAddress = probeAddr
	ctrlOpts.LeaderElection = enableLeaderElection

	return ctrlOpts, zapOpts
}

func configureReconcilers(k8sManager controllerManager, operatorConfig *config.OperatorConfig) error {
	var recorder eventRecorder = k8sManager.GetEventRecorderFor("k8s-backup-operator")

	ctx := context.Background()

	k8sClientSet, err := kubernetes.NewForConfig(k8sManager.GetConfig())
	if err != nil {
		return fmt.Errorf("unable to create k8s clientset: %w", err)
	}

	ecosystemClientSet, err := ecosystem.NewClientSet(k8sManager.GetConfig(), k8sClientSet)
	if err != nil {
		return fmt.Errorf("unable to create ecosystem clientset: %w", err)
	}

	registry, err := reg.New(core.Registry{
		Type:      "etcd",
		Endpoints: []string{fmt.Sprintf("http://etcd.%s.svc.cluster.local:4001", operatorConfig.Namespace)},
	})
	if err != nil {
		return fmt.Errorf("failed to create CES registry: %w", err)
	}

	imageGetter := additionalimages.NewGetter(k8sClientSet, operatorConfig.Namespace)
	kubectlImage, err := imageGetter.ImageForKey(ctx, config.KubectlImageConfigmapNameKey)
	if err != nil {
		return fmt.Errorf("failed to get kubectl image: %w", err)
	}

	requeueHandler := requeue.NewRequeueHandler(ecosystemClientSet, recorder, operatorConfig.Namespace)
	cleanupManager := cleanup.NewManager(operatorConfig.Namespace, k8sManager.GetClient(), k8sClientSet)
	restoreManager := restore.NewRestoreManager(
		ecosystemClientSet.EcosystemV1Alpha1().Restores(operatorConfig.Namespace),
		recorder,
		registry,
		ecosystemClientSet.AppsV1().StatefulSets(operatorConfig.Namespace),
		ecosystemClientSet.CoreV1().Services(operatorConfig.Namespace),
		cleanupManager,
	)
	if err = (restore.NewRestoreReconciler(ecosystemClientSet, recorder, operatorConfig.Namespace, restoreManager, requeueHandler)).SetupWithManager(k8sManager); err != nil {
		return fmt.Errorf("unable to create restore controller: %w", err)
	}

	backupManager := backup.NewBackupManager(ecosystemClientSet.EcosystemV1Alpha1().Backups(operatorConfig.Namespace), recorder, registry)
	if err = (backup.NewBackupReconciler(ecosystemClientSet, recorder, operatorConfig.Namespace, backupManager, requeueHandler)).SetupWithManager(k8sManager); err != nil {
		return fmt.Errorf("unable to create backup controller: %w", err)
	}

	if err = backupschedule.NewReconciler(ecosystemClientSet, recorder, operatorConfig.Namespace, requeueHandler, kubectlImage).SetupWithManager(k8sManager); err != nil {
		return fmt.Errorf("unable to create backupSchedule controller: %w", err)
	}
	// +kubebuilder:scaffold:builder

	return nil
}

func addChecks(k8sManager controllerManager) error {
	if err := k8sManager.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		return fmt.Errorf("unable to set up health check: %w", err)
	}
	if err := k8sManager.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		return fmt.Errorf("unable to set up ready check: %w", err)
	}

	return nil
}

func startK8sManager(k8sManager controllerManager) error {
	setupLog.Info("starting manager")
	if err := k8sManager.Start(ctrl.SetupSignalHandler()); err != nil {
		return fmt.Errorf("problem running manager: %w", err)
	}

	return nil
}
