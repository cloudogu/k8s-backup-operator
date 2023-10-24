//go:build k8s_integration
// +build k8s_integration

package main

import (
	"context"
	"github.com/cloudogu/k8s-backup-operator/pkg/cleanup"
	"os"
	"path/filepath"
	"testing"
	"time"

	"k8s.io/client-go/kubernetes"
	k8sScheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"

	"github.com/cloudogu/k8s-backup-operator/pkg/api/ecosystem"
	k8sv1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
	"github.com/cloudogu/k8s-backup-operator/pkg/backup"
	"github.com/cloudogu/k8s-backup-operator/pkg/config"
	"github.com/cloudogu/k8s-backup-operator/pkg/restore"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.
var testEnv *envtest.Environment
var cfg *rest.Config
var cancel context.CancelFunc

// Used in other integration tests
var (
	ecosystemClientSet ecosystem.Interface
	recorderMock       *mockEventRecorder
	namespace          = "default"
)

const TimeoutInterval = time.Second * 10
const PollingInterval = time.Second * 1

var oldGetConfig func() (*rest.Config, error)
var oldGetConfigOrDie func() *rest.Config

func TestControllers(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)

	ginkgo.RunSpecs(t, "Controller Suite")
}

var _ = ginkgo.BeforeSuite(func() {
	// We need to ensure that the development stage flag is not passed by our makefiles to prevent the component operator
	// from running in the developing mode. The developing mode changes some operator behaviour. Our integration test
	// aim to test the production functionality of the operator.
	err := os.Unsetenv(config.StageEnvVar)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	err = os.Setenv(config.StageEnvVar, config.StageProduction)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	config.Stage = config.StageProduction

	logf.SetLogger(zap.New(zap.WriteTo(ginkgo.GinkgoWriter), zap.UseDevMode(true)))

	var ctx context.Context
	ctx, cancel = context.WithCancel(context.TODO())

	ginkgo.By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(cfg).NotTo(gomega.BeNil())

	oldGetConfig = ctrl.GetConfig
	ctrl.GetConfig = func() (*rest.Config, error) {
		return cfg, nil
	}

	oldGetConfigOrDie = ctrl.GetConfigOrDie
	ctrl.GetConfigOrDie = func() *rest.Config {
		return cfg
	}

	err = k8sv1.AddToScheme(k8sScheme.Scheme)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	//+kubebuilder:scaffold:scheme
	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: k8sScheme.Scheme,
	})
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	gomega.Expect(k8sManager).NotTo(gomega.BeNil())
	t := &testing.T{}
	recorderMock = newMockEventRecorder(t)

	clientSet, err := kubernetes.NewForConfig(cfg)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	ecosystemClientSet, err = ecosystem.NewClientSet(k8sManager.GetConfig(), clientSet)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	mockRegistry := newMockEtcdRegistry(t)
	globalConfigMock := newMockEtcdContext(t)
	mockRegistry.EXPECT().GlobalConfig().Return(globalConfigMock)

	backupManager := backup.NewBackupManager(ecosystemClientSet.EcosystemV1Alpha1().Backups(namespace), recorderMock, mockRegistry)
	gomega.Expect(backupManager).NotTo(gomega.BeNil())
	backupRequeueHandler := backup.NewBackupRequeueHandler(ecosystemClientSet, recorderMock, namespace)
	gomega.Expect(backupRequeueHandler).NotTo(gomega.BeNil())

	backupReconciler := backup.NewBackupReconciler(ecosystemClientSet, recorderMock, namespace, backupManager, backupRequeueHandler)
	gomega.Expect(backupReconciler).NotTo(gomega.BeNil())

	err = backupReconciler.SetupWithManager(k8sManager)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	cleanupMock := cleanup.NewManager(namespace, k8sManager.GetClient(), clientSet)
	restoreRequeueHandler := restore.NewRequeueHandler(ecosystemClientSet, recorderMock, namespace)
	gomega.Expect(restoreRequeueHandler).NotTo(gomega.BeNil())
	restoreManager := restore.NewRestoreManager(
		ecosystemClientSet.EcosystemV1Alpha1().Restores(namespace),
		recorderMock,
		mockRegistry,
		ecosystemClientSet.AppsV1().StatefulSets(namespace),
		ecosystemClientSet.CoreV1().Services(namespace),
		cleanupMock,
	)
	gomega.Expect(restoreManager).NotTo(gomega.BeNil())
	restoreReconciler := restore.NewRestoreReconciler(ecosystemClientSet, recorderMock, namespace, restoreManager, restoreRequeueHandler)
	gomega.Expect(restoreReconciler).NotTo(gomega.BeNil())

	err = restoreReconciler.SetupWithManager(k8sManager)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	go func() {
		err = k8sManager.Start(ctx)
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
	}()
}, 60)

var _ = ginkgo.AfterSuite(func() {
	cancel()
	ginkgo.By("tearing down the test environment")
	err := testEnv.Stop()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	ctrl.GetConfig = oldGetConfig
	ctrl.GetConfigOrDie = oldGetConfigOrDie
})
