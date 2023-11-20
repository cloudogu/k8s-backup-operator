package main

import (
	"context"
	"flag"
	"fmt"
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

	"github.com/cloudogu/k8s-backup-operator/pkg/additionalimages"
	"github.com/cloudogu/k8s-backup-operator/pkg/api/ecosystem"
	k8sv1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
	"github.com/cloudogu/k8s-backup-operator/pkg/backup"
	"github.com/cloudogu/k8s-backup-operator/pkg/backupschedule"
	"github.com/cloudogu/k8s-backup-operator/pkg/cleanup"
	"github.com/cloudogu/k8s-backup-operator/pkg/config"
	"github.com/cloudogu/k8s-backup-operator/pkg/garbagecollection"
	"github.com/cloudogu/k8s-backup-operator/pkg/requeue"
	"github.com/cloudogu/k8s-backup-operator/pkg/restore"
	// +kubebuilder:scaffold:imports
)

var (
	operatorCmd         = flag.NewFlagSet("operator", flag.ExitOnError)
	garbageCollectorCmd = flag.NewFlagSet("gc", flag.ExitOnError)
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

var (
	newAdditionalImageGetter    = additionalimages.NewGetter
	newAdditionalImageUpdater   = additionalimages.NewUpdater
	newGarbageCollectionManager = garbagecollection.NewManager
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(k8sv1.AddToScheme(scheme))

	// +kubebuilder:scaffold:scheme
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if len(os.Args) < 2 {
		fmt.Printf("expected one of the following subcommands:\n"+
			"  %s - start in operator-mode, reconciling this operators custom resources\n"+
			"  %s - start in garbage-collection mode, deleting backups according to the configured retention strategy\n",
			operatorCmd.Name(),
			garbageCollectorCmd.Name())
		os.Exit(1)
	}

	switch os.Args[1] {
	case operatorCmd.Name():
		err := startOperator(ctx, operatorCmd, os.Args[2:])
		if err != nil {
			setupLog.Error(err, "failed to start operator")
			os.Exit(1)
		}
	case garbageCollectorCmd.Name():
		err := startGarbageCollector(ctx, garbageCollectorCmd, os.Args[2:])
		if err != nil {
			setupLog.Error(err, "failed to start garbage-collector")
			os.Exit(1)
		}
	}
}

func startGarbageCollector(ctx context.Context, flags *flag.FlagSet, args []string) error {
	restConfig := ctrl.GetConfigOrDie()
	namespace, err := config.GetNamespace()
	if err != nil {
		return fmt.Errorf("unable to get current namespace: %w", err)
	}

	k8sClientSet, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return fmt.Errorf("unable to create k8s clientset: %w", err)
	}

	ecosystemClientSet, err := ecosystem.NewClientSet(restConfig, k8sClientSet)
	if err != nil {
		return fmt.Errorf("unable to create ecosystem clientset: %w", err)
	}

	retentionStrategy := parseStrategyName(flags, args)

	gcManager := newGarbageCollectionManager(ecosystemClientSet, namespace, retentionStrategy)
	return gcManager.CollectGarbage(ctx)
}

func parseStrategyName(flags *flag.FlagSet, args []string) string {
	var strategyName string
	flags.StringVar(&strategyName, "strategy", "keepAll", "The retention strategy to decide which backups to delete and which to keep.")

	// Ignore errors; flags is set to exit on errors
	_ = flags.Parse(args)
	return strategyName
}

func startOperator(ctx context.Context, flags *flag.FlagSet, args []string) error {
	operatorConfig, err := config.NewOperatorConfig(Version)
	if err != nil {
		return fmt.Errorf("unable to create operator config: %w", err)
	}

	options := getK8sManagerOptions(flags, args, operatorConfig)
	restConfig := ctrl.GetConfigOrDie()

	k8sManager, err := ctrl.NewManager(restConfig, options)
	if err != nil {
		return fmt.Errorf("unable to start manager: %w", err)
	}

	err = configureManager(ctx, k8sManager, operatorConfig)
	if err != nil {
		return fmt.Errorf("unable to configure manager: %w", err)
	}

	return startK8sManager(k8sManager)
}

func configureManager(ctx context.Context, k8sManager controllerManager, operatorConfig *config.OperatorConfig) error {
	err := configureReconcilers(ctx, k8sManager, operatorConfig)
	if err != nil {
		return fmt.Errorf("unable to configure reconciler: %w", err)
	}

	err = addChecks(k8sManager)
	if err != nil {
		return fmt.Errorf("unable to add checks to the manager: %w", err)
	}

	return nil
}

func getK8sManagerOptions(flags *flag.FlagSet, args []string, operatorConfig *config.OperatorConfig) ctrl.Options {
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
	controllerOpts, zapOpts := parseManagerFlags(flags, args, controllerOpts)

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&zapOpts)))

	return controllerOpts
}

func parseManagerFlags(flags *flag.FlagSet, args []string, ctrlOpts ctrl.Options) (ctrl.Options, zap.Options) {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	flags.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flags.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flags.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	zapOpts := zap.Options{
		Development: config.IsStageDevelopment(),
	}
	zapOpts.BindFlags(flags)
	// Ignore errors; flags is set to exit on errors
	_ = flags.Parse(args)

	ctrlOpts.Metrics = server.Options{BindAddress: metricsAddr}
	ctrlOpts.HealthProbeBindAddress = probeAddr
	ctrlOpts.LeaderElection = enableLeaderElection

	return ctrlOpts, zapOpts
}

func configureReconcilers(ctx context.Context, k8sManager controllerManager, operatorConfig *config.OperatorConfig) error {
	var recorder eventRecorder = k8sManager.GetEventRecorderFor("k8s-backup-operator")

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

	imageGetter := newAdditionalImageGetter(k8sClientSet, operatorConfig.Namespace)
	kubectlImage, err := imageGetter.ImageForKey(ctx, config.KubectlImageConfigmapNameKey)
	if err != nil {
		return fmt.Errorf("failed to get kubectl image: %w", err)
	}

	additionalImageUpdater := newAdditionalImageUpdater(ecosystemClientSet, operatorConfig.Namespace, kubectlImage, recorder)
	err = additionalImageUpdater.Update(ctx)
	if err != nil {
		return fmt.Errorf("failed to update additional images in existing resources: %w", err)
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
