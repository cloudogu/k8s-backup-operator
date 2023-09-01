package main

import (
	"flag"
	"fmt"
	"os"

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
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/cloudogu/k8s-backup-operator/pkg/api/ecosystem"
	k8sv1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
	"github.com/cloudogu/k8s-backup-operator/pkg/config"
	"github.com/cloudogu/k8s-backup-operator/pkg/controllers"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

var (
	// Version of the application
	Version = "0.0.0"
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(k8sv1.AddToScheme(scheme))

	//+kubebuilder:scaffold:scheme
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

func configureManager(k8sManager manager.Manager, operatorConfig *config.OperatorConfig) error {
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
	}
	zapOpts := zap.Options{}
	controllerOpts, zapOpts = parseFlags(controllerOpts)

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&zapOpts)))

	return controllerOpts
}

var parseFlags = func(options ctrl.Options) (ctrl.Options, zap.Options) {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	opts := zap.Options{
		Development: config.IsStageDevelopment(),
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	options.Metrics = server.Options{BindAddress: metricsAddr}
	options.HealthProbeBindAddress = probeAddr
	options.LeaderElection = enableLeaderElection

	return options, opts
}

func configureReconcilers(k8sManager manager.Manager, operatorConfig *config.OperatorConfig) error {
	eventRecorder := k8sManager.GetEventRecorderFor("k8s-backup-operator")

	k8sClientSet, err := kubernetes.NewForConfig(k8sManager.GetConfig())
	if err != nil {
		return fmt.Errorf("unable to create k8s clientset: %w", err)
	}

	ecosystemClientSet, err := ecosystem.NewClientSet(k8sManager.GetConfig(), k8sClientSet)
	if err != nil {
		return fmt.Errorf("unable to create ecosystem clientset: %w", err)
	}

	if err = (controller.NewRestoreReconciler(ecosystemClientSet, eventRecorder, operatorConfig.Namespace)).SetupWithManager(k8sManager); err != nil {
		return fmt.Errorf("unable to create restore controller: %w", err)
	}
	if err = (controller.NewBackupReconciler(ecosystemClientSet, eventRecorder, operatorConfig.Namespace)).SetupWithManager(k8sManager); err != nil {
		return fmt.Errorf("unable to create backup controller: %w", err)
	}
	// +kubebuilder:scaffold:builder

	return nil
}

func addChecks(k8sManager manager.Manager) error {
	if err := k8sManager.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		return fmt.Errorf("unable to set up health check: %w", err)
	}
	if err := k8sManager.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		return fmt.Errorf("unable to set up ready check: %w", err)
	}

	return nil
}

func startK8sManager(k8sManager manager.Manager) error {
	setupLog.Info("starting manager")
	if err := k8sManager.Start(ctrl.SetupSignalHandler()); err != nil {
		return fmt.Errorf("problem running manager: %w", err)
	}

	return nil
}
