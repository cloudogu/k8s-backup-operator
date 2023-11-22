package main

import (
	"context"
	"flag"
	"github.com/cloudogu/k8s-backup-operator/pkg/additionalimages"
	"github.com/cloudogu/k8s-backup-operator/pkg/api/ecosystem"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	v1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
)

var testCtx = context.Background()

func Test_parseFlags(t *testing.T) {
	t.Run("should use defaults for flags", func(t *testing.T) {
		// given
		flags := flag.NewFlagSet("operator", flag.ContinueOnError)
		ctrlOpts := ctrl.Options{
			Scheme: scheme,
			Cache: cache.Options{DefaultNamespaces: map[string]cache.Config{
				"ecosystem": {},
			}},
			WebhookServer:        webhook.NewServer(webhook.Options{Port: 9443}),
			LeaderElectionID:     "e3f6c1a7.cloudogu.com",
			LivenessEndpointName: "/lifez",
		}

		// when
		newCtrlOpts, _ := parseManagerFlags(flags, []string{}, ctrlOpts)

		// then
		require.NotEmpty(t, newCtrlOpts)
		assert.Equal(t, ":8080", newCtrlOpts.Metrics.BindAddress)
		assert.Equal(t, ":8081", newCtrlOpts.HealthProbeBindAddress)
		assert.False(t, newCtrlOpts.LeaderElection)
		assert.Same(t, ctrlOpts.Scheme, newCtrlOpts.Scheme)
		assert.Equal(t, ctrlOpts.Cache, newCtrlOpts.Cache)
		assert.Equal(t, ctrlOpts.WebhookServer, newCtrlOpts.WebhookServer)
		assert.Equal(t, ctrlOpts.LeaderElectionID, newCtrlOpts.LeaderElectionID)
		assert.Equal(t, ctrlOpts.LivenessEndpointName, newCtrlOpts.LivenessEndpointName)
	})
	t.Run("should use values from flags", func(t *testing.T) {
		// given
		flags := flag.NewFlagSet("operator", flag.ContinueOnError)
		args := []string{
			"--metrics-bind-address", ":9090",
			"--health-probe-bind-address", ":9091",
			"--leader-elect",
		}
		ctrlOpts := ctrl.Options{
			Scheme: scheme,
			Cache: cache.Options{DefaultNamespaces: map[string]cache.Config{
				"ecosystem": {},
			}},
			WebhookServer:        webhook.NewServer(webhook.Options{Port: 9443}),
			LeaderElectionID:     "e3f6c1a7.cloudogu.com",
			LivenessEndpointName: "/lifez",
		}

		// when
		newCtrlOpts, _ := parseManagerFlags(flags, args, ctrlOpts)

		// then
		require.NotEmpty(t, newCtrlOpts)
		assert.Equal(t, ":9090", newCtrlOpts.Metrics.BindAddress)
		assert.Equal(t, ":9091", newCtrlOpts.HealthProbeBindAddress)
		assert.True(t, newCtrlOpts.LeaderElection)
		assert.Same(t, ctrlOpts.Scheme, newCtrlOpts.Scheme)
		assert.Equal(t, ctrlOpts.Cache, newCtrlOpts.Cache)
		assert.Equal(t, ctrlOpts.WebhookServer, newCtrlOpts.WebhookServer)
		assert.Equal(t, ctrlOpts.LeaderElectionID, newCtrlOpts.LeaderElectionID)
		assert.Equal(t, ctrlOpts.LivenessEndpointName, newCtrlOpts.LivenessEndpointName)
	})
}

func Test_startOperator(t *testing.T) {
	t.Run("should fail to create operator config", func(t *testing.T) {
		// given
		oldVersion := Version
		Version = "invalid"
		defer func() { Version = oldVersion }()

		flags := flag.NewFlagSet("operator", flag.ContinueOnError)

		// when
		err := startOperator(testCtx, flags, []string{})

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "unable to create operator config")
	})
	t.Run("should fail to create controller manager", func(t *testing.T) {
		// given
		t.Setenv("NAMESPACE", "ecosystem")

		oldNewManagerFunc := ctrl.NewManager
		oldGetConfigFunc := ctrl.GetConfigOrDie
		defer func() {
			ctrl.NewManager = oldNewManagerFunc
			ctrl.GetConfigOrDie = oldGetConfigFunc
		}()

		ctrl.NewManager = func(config *rest.Config, options manager.Options) (manager.Manager, error) {
			return nil, assert.AnError
		}
		ctrl.GetConfigOrDie = func() *rest.Config {
			return &rest.Config{}
		}

		flags := flag.NewFlagSet("operator", flag.ContinueOnError)

		// when
		err := startOperator(testCtx, flags, []string{})

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "unable to start manager")
	})
	t.Run("should fail to get kubectl image", func(t *testing.T) {
		// given
		t.Setenv("NAMESPACE", "ecosystem")
		t.Setenv("STAGE", "development")

		oldNewManagerFunc := ctrl.NewManager
		oldGetConfigFunc := ctrl.GetConfigOrDie
		oldNewAdditionalImageGetterFunc := newAdditionalImageGetter
		oldNewAdditionalImageUpdaterFunc := newAdditionalImageUpdater
		defer func() {
			ctrl.NewManager = oldNewManagerFunc
			ctrl.GetConfigOrDie = oldGetConfigFunc
			newAdditionalImageGetter = oldNewAdditionalImageGetterFunc
			newAdditionalImageUpdater = oldNewAdditionalImageUpdaterFunc
		}()

		restConfig := &rest.Config{}
		recorderMock := newMockEventRecorder(t)
		ctrlManMock := newMockControllerManager(t)
		ctrlManMock.EXPECT().GetEventRecorderFor("k8s-backup-operator").Return(recorderMock)
		ctrlManMock.EXPECT().GetConfig().Return(restConfig)

		ctrl.NewManager = func(config *rest.Config, options manager.Options) (manager.Manager, error) {
			return ctrlManMock, nil
		}
		ctrl.GetConfigOrDie = func() *rest.Config {
			return restConfig
		}

		additionalImageGetterMock := newMockAdditionalImageGetter(t)
		additionalImageGetterMock.EXPECT().ImageForKey(testCtx, "kubectlImage").Return("", assert.AnError)
		newAdditionalImageGetter = func(_ kubernetes.Interface, _ string) additionalimages.Getter {
			return additionalImageGetterMock
		}
		additionalImageUpdaterMock := newMockAdditionalImageUpdater(t)
		newAdditionalImageUpdater = func(_ ecosystem.Interface, _ string, _ string, _ record.EventRecorder) additionalimages.Updater {
			return additionalImageUpdaterMock
		}

		flags := flag.NewFlagSet("operator", flag.ContinueOnError)

		// when
		err := startOperator(testCtx, flags, []string{})

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to get kubectl image")
		assert.ErrorContains(t, err, "unable to configure manager: unable to configure reconciler")
	})
	t.Run("should fail update additional images", func(t *testing.T) {
		// given
		t.Setenv("NAMESPACE", "ecosystem")
		t.Setenv("STAGE", "development")

		oldNewManagerFunc := ctrl.NewManager
		oldGetConfigFunc := ctrl.GetConfigOrDie
		oldNewAdditionalImageGetterFunc := newAdditionalImageGetter
		oldNewAdditionalImageUpdaterFunc := newAdditionalImageUpdater
		defer func() {
			ctrl.NewManager = oldNewManagerFunc
			ctrl.GetConfigOrDie = oldGetConfigFunc
			newAdditionalImageGetter = oldNewAdditionalImageGetterFunc
			newAdditionalImageUpdater = oldNewAdditionalImageUpdaterFunc
		}()

		restConfig := &rest.Config{}
		recorderMock := newMockEventRecorder(t)
		ctrlManMock := newMockControllerManager(t)
		ctrlManMock.EXPECT().GetEventRecorderFor("k8s-backup-operator").Return(recorderMock)
		ctrlManMock.EXPECT().GetConfig().Return(restConfig)

		ctrl.NewManager = func(config *rest.Config, options manager.Options) (manager.Manager, error) {
			return ctrlManMock, nil
		}
		ctrl.GetConfigOrDie = func() *rest.Config {
			return restConfig
		}

		additionalImageGetterMock := newMockAdditionalImageGetter(t)
		additionalImageGetterMock.EXPECT().ImageForKey(testCtx, "kubectlImage").Return("bitnami/kubectl:1.27.7", nil)
		newAdditionalImageGetter = func(_ kubernetes.Interface, _ string) additionalimages.Getter {
			return additionalImageGetterMock
		}
		additionalImageUpdaterMock := newMockAdditionalImageUpdater(t)
		additionalImageUpdaterMock.EXPECT().Update(testCtx).Return(assert.AnError)
		newAdditionalImageUpdater = func(_ ecosystem.Interface, _ string, _ string, _ record.EventRecorder) additionalimages.Updater {
			return additionalImageUpdaterMock
		}

		flags := flag.NewFlagSet("operator", flag.ContinueOnError)

		// when
		err := startOperator(testCtx, flags, []string{})

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to update additional images in existing resources")
		assert.ErrorContains(t, err, "unable to configure manager: unable to configure reconciler")
	})
	t.Run("should fail to configure reconciler", func(t *testing.T) {
		// given
		t.Setenv("NAMESPACE", "ecosystem")
		t.Setenv("STAGE", "development")

		oldNewManagerFunc := ctrl.NewManager
		oldGetConfigFunc := ctrl.GetConfigOrDie
		oldNewAdditionalImageGetterFunc := newAdditionalImageGetter
		oldNewAdditionalImageUpdaterFunc := newAdditionalImageUpdater
		defer func() {
			ctrl.NewManager = oldNewManagerFunc
			ctrl.GetConfigOrDie = oldGetConfigFunc
			newAdditionalImageGetter = oldNewAdditionalImageGetterFunc
			newAdditionalImageUpdater = oldNewAdditionalImageUpdaterFunc
		}()

		restConfig := &rest.Config{}
		recorderMock := newMockEventRecorder(t)
		k8sClientMock := newMockK8sClient(t)
		ctrlManMock := newMockControllerManager(t)
		ctrlManMock.EXPECT().GetEventRecorderFor("k8s-backup-operator").Return(recorderMock)
		ctrlManMock.EXPECT().GetConfig().Return(restConfig)
		ctrlManMock.EXPECT().GetClient().Return(k8sClientMock)
		ctrlManMock.EXPECT().GetControllerOptions().Return(config.Controller{})
		ctrlManMock.EXPECT().GetScheme().Return(runtime.NewScheme())

		ctrl.NewManager = func(config *rest.Config, options manager.Options) (manager.Manager, error) {
			return ctrlManMock, nil
		}
		ctrl.GetConfigOrDie = func() *rest.Config {
			return restConfig
		}

		additionalImageGetterMock := newMockAdditionalImageGetter(t)
		additionalImageGetterMock.EXPECT().ImageForKey(testCtx, "kubectlImage").Return("bitnami/kubectl:1.27.7", nil)
		newAdditionalImageGetter = func(_ kubernetes.Interface, _ string) additionalimages.Getter {
			return additionalImageGetterMock
		}
		additionalImageUpdaterMock := newMockAdditionalImageUpdater(t)
		additionalImageUpdaterMock.EXPECT().Update(testCtx).Return(nil)
		newAdditionalImageUpdater = func(_ ecosystem.Interface, _ string, _ string, _ record.EventRecorder) additionalimages.Updater {
			return additionalImageUpdaterMock
		}

		flags := flag.NewFlagSet("operator", flag.ContinueOnError)

		// when
		err := startOperator(testCtx, flags, []string{})

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "unable to configure manager: unable to configure reconciler")
	})
	t.Run("should fail to add health check to controller manager", func(t *testing.T) {
		// given
		t.Setenv("NAMESPACE", "ecosystem")
		t.Setenv("STAGE", "development")

		oldNewManagerFunc := ctrl.NewManager
		oldGetConfigFunc := ctrl.GetConfigOrDie
		oldNewAdditionalImageGetterFunc := newAdditionalImageGetter
		oldNewAdditionalImageUpdaterFunc := newAdditionalImageUpdater
		defer func() {
			ctrl.NewManager = oldNewManagerFunc
			ctrl.GetConfigOrDie = oldGetConfigFunc
			newAdditionalImageGetter = oldNewAdditionalImageGetterFunc
			newAdditionalImageUpdater = oldNewAdditionalImageUpdaterFunc
		}()

		logMock := newMockLogSink(t)
		logMock.EXPECT().Init(mock.Anything).Return()
		logMock.EXPECT().WithValues(mock.Anything, mock.Anything).Return(logMock)
		logMock.EXPECT().WithValues(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(logMock)

		restConfig := &rest.Config{}
		recorderMock := newMockEventRecorder(t)
		k8sClientMock := newMockK8sClient(t)
		ctrlManMock := newMockControllerManager(t)
		ctrlManMock.EXPECT().GetEventRecorderFor("k8s-backup-operator").Return(recorderMock)
		ctrlManMock.EXPECT().GetConfig().Return(restConfig)
		ctrlManMock.EXPECT().GetClient().Return(k8sClientMock)
		ctrlManMock.EXPECT().GetControllerOptions().Return(config.Controller{})
		ctrlManMock.EXPECT().GetScheme().Return(createScheme(t))
		ctrlManMock.EXPECT().GetLogger().Return(logr.New(logMock))
		ctrlManMock.EXPECT().Add(mock.Anything).Return(nil)
		ctrlManMock.EXPECT().GetCache().Return(nil)
		ctrlManMock.EXPECT().AddHealthzCheck("healthz", mock.Anything).Return(assert.AnError)

		ctrl.NewManager = func(config *rest.Config, options manager.Options) (manager.Manager, error) {
			return ctrlManMock, nil
		}
		ctrl.GetConfigOrDie = func() *rest.Config {
			return restConfig
		}

		additionalImageGetterMock := newMockAdditionalImageGetter(t)
		additionalImageGetterMock.EXPECT().ImageForKey(testCtx, "kubectlImage").Return("bitnami/kubectl:1.27.7", nil)
		newAdditionalImageGetter = func(_ kubernetes.Interface, _ string) additionalimages.Getter {
			return additionalImageGetterMock
		}
		additionalImageUpdaterMock := newMockAdditionalImageUpdater(t)
		additionalImageUpdaterMock.EXPECT().Update(testCtx).Return(nil)
		newAdditionalImageUpdater = func(_ ecosystem.Interface, _ string, _ string, _ record.EventRecorder) additionalimages.Updater {
			return additionalImageUpdaterMock
		}

		flags := flag.NewFlagSet("operator", flag.ContinueOnError)

		// when
		err := startOperator(testCtx, flags, []string{})

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "unable to configure manager: unable to add checks to the manager: unable to set up health check")
	})
	t.Run("should fail to add readiness check to controller manager", func(t *testing.T) {
		// given
		t.Setenv("NAMESPACE", "ecosystem")
		t.Setenv("STAGE", "development")

		oldNewManagerFunc := ctrl.NewManager
		oldGetConfigFunc := ctrl.GetConfigOrDie
		oldNewAdditionalImageGetterFunc := newAdditionalImageGetter
		oldNewAdditionalImageUpdaterFunc := newAdditionalImageUpdater
		defer func() {
			ctrl.NewManager = oldNewManagerFunc
			ctrl.GetConfigOrDie = oldGetConfigFunc
			newAdditionalImageGetter = oldNewAdditionalImageGetterFunc
			newAdditionalImageUpdater = oldNewAdditionalImageUpdaterFunc
		}()

		logMock := newMockLogSink(t)
		logMock.EXPECT().Init(mock.Anything).Return()
		logMock.EXPECT().WithValues(mock.Anything, mock.Anything).Return(logMock)
		logMock.EXPECT().WithValues(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(logMock)

		restConfig := &rest.Config{}
		recorderMock := newMockEventRecorder(t)
		k8sClientMock := newMockK8sClient(t)
		ctrlManMock := newMockControllerManager(t)
		ctrlManMock.EXPECT().GetEventRecorderFor("k8s-backup-operator").Return(recorderMock)
		ctrlManMock.EXPECT().GetConfig().Return(restConfig)
		ctrlManMock.EXPECT().GetClient().Return(k8sClientMock)
		ctrlManMock.EXPECT().GetControllerOptions().Return(config.Controller{})
		ctrlManMock.EXPECT().GetScheme().Return(createScheme(t))
		ctrlManMock.EXPECT().GetLogger().Return(logr.New(logMock))
		ctrlManMock.EXPECT().Add(mock.Anything).Return(nil)
		ctrlManMock.EXPECT().GetCache().Return(nil)
		ctrlManMock.EXPECT().AddHealthzCheck("healthz", mock.Anything).Return(nil)
		ctrlManMock.EXPECT().AddReadyzCheck("readyz", mock.Anything).Return(assert.AnError)

		ctrl.NewManager = func(config *rest.Config, options manager.Options) (manager.Manager, error) {
			return ctrlManMock, nil
		}
		ctrl.GetConfigOrDie = func() *rest.Config {
			return restConfig
		}

		additionalImageGetterMock := newMockAdditionalImageGetter(t)
		additionalImageGetterMock.EXPECT().ImageForKey(testCtx, "kubectlImage").Return("bitnami/kubectl:1.27.7", nil)
		newAdditionalImageGetter = func(_ kubernetes.Interface, _ string) additionalimages.Getter {
			return additionalImageGetterMock
		}
		additionalImageUpdaterMock := newMockAdditionalImageUpdater(t)
		additionalImageUpdaterMock.EXPECT().Update(testCtx).Return(nil)
		newAdditionalImageUpdater = func(_ ecosystem.Interface, _ string, _ string, _ record.EventRecorder) additionalimages.Updater {
			return additionalImageUpdaterMock
		}

		flags := flag.NewFlagSet("operator", flag.ContinueOnError)

		// when
		err := startOperator(testCtx, flags, []string{})

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "unable to configure manager: unable to add checks to the manager: unable to set up ready check")
	})
	t.Run("should fail to start controller manager", func(t *testing.T) {
		// given
		t.Setenv("NAMESPACE", "ecosystem")
		t.Setenv("STAGE", "development")

		oldNewManagerFunc := ctrl.NewManager
		oldGetConfigFunc := ctrl.GetConfigOrDie
		oldSignalHandlerFunc := ctrl.SetupSignalHandler
		oldNewAdditionalImageGetterFunc := newAdditionalImageGetter
		oldNewAdditionalImageUpdaterFunc := newAdditionalImageUpdater
		defer func() {
			ctrl.NewManager = oldNewManagerFunc
			ctrl.GetConfigOrDie = oldGetConfigFunc
			ctrl.SetupSignalHandler = oldSignalHandlerFunc
			newAdditionalImageGetter = oldNewAdditionalImageGetterFunc
			newAdditionalImageUpdater = oldNewAdditionalImageUpdaterFunc
		}()

		logMock := newMockLogSink(t)
		logMock.EXPECT().Init(mock.Anything).Return()
		logMock.EXPECT().WithValues(mock.Anything, mock.Anything).Return(logMock)
		logMock.EXPECT().WithValues(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(logMock)

		restConfig := &rest.Config{}
		recorderMock := newMockEventRecorder(t)
		k8sClientMock := newMockK8sClient(t)
		ctrlManMock := newMockControllerManager(t)
		ctrlManMock.EXPECT().GetEventRecorderFor("k8s-backup-operator").Return(recorderMock)
		ctrlManMock.EXPECT().GetConfig().Return(restConfig)
		ctrlManMock.EXPECT().GetClient().Return(k8sClientMock)
		ctrlManMock.EXPECT().GetControllerOptions().Return(config.Controller{})
		ctrlManMock.EXPECT().GetScheme().Return(createScheme(t))
		ctrlManMock.EXPECT().GetLogger().Return(logr.New(logMock))
		ctrlManMock.EXPECT().Add(mock.Anything).Return(nil)
		ctrlManMock.EXPECT().GetCache().Return(nil)
		ctrlManMock.EXPECT().AddHealthzCheck("healthz", mock.Anything).Return(nil)
		ctrlManMock.EXPECT().AddReadyzCheck("readyz", mock.Anything).Return(nil)
		ctrlManMock.EXPECT().Start(mock.Anything).Return(assert.AnError)

		ctrl.NewManager = func(config *rest.Config, options manager.Options) (manager.Manager, error) {
			return ctrlManMock, nil
		}
		ctrl.GetConfigOrDie = func() *rest.Config {
			return restConfig
		}
		ctrl.SetupSignalHandler = func() context.Context {
			return testCtx
		}

		additionalImageGetterMock := newMockAdditionalImageGetter(t)
		additionalImageGetterMock.EXPECT().ImageForKey(testCtx, "kubectlImage").Return("bitnami/kubectl:1.27.7", nil)
		newAdditionalImageGetter = func(_ kubernetes.Interface, _ string) additionalimages.Getter {
			return additionalImageGetterMock
		}
		additionalImageUpdaterMock := newMockAdditionalImageUpdater(t)
		additionalImageUpdaterMock.EXPECT().Update(testCtx).Return(nil)
		newAdditionalImageUpdater = func(_ ecosystem.Interface, _ string, _ string, _ record.EventRecorder) additionalimages.Updater {
			return additionalImageUpdaterMock
		}

		flags := flag.NewFlagSet("operator", flag.ContinueOnError)

		// when
		err := startOperator(testCtx, flags, []string{})

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "problem running manager")
	})
	t.Run("should succeed to start controller manager", func(t *testing.T) {
		// given
		t.Setenv("NAMESPACE", "ecosystem")
		t.Setenv("STAGE", "development")

		oldNewManagerFunc := ctrl.NewManager
		oldGetConfigFunc := ctrl.GetConfigOrDie
		oldSignalHandlerFunc := ctrl.SetupSignalHandler
		oldNewAdditionalImageGetterFunc := newAdditionalImageGetter
		oldNewAdditionalImageUpdaterFunc := newAdditionalImageUpdater
		defer func() {
			ctrl.NewManager = oldNewManagerFunc
			ctrl.GetConfigOrDie = oldGetConfigFunc
			ctrl.SetupSignalHandler = oldSignalHandlerFunc
			newAdditionalImageGetter = oldNewAdditionalImageGetterFunc
			newAdditionalImageUpdater = oldNewAdditionalImageUpdaterFunc
		}()

		logMock := newMockLogSink(t)
		logMock.EXPECT().Init(mock.Anything).Return()
		logMock.EXPECT().WithValues(mock.Anything, mock.Anything).Return(logMock)
		logMock.EXPECT().WithValues(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(logMock)

		restConfig := &rest.Config{}
		recorderMock := newMockEventRecorder(t)
		k8sClientMock := newMockK8sClient(t)
		ctrlManMock := newMockControllerManager(t)
		ctrlManMock.EXPECT().GetEventRecorderFor("k8s-backup-operator").Return(recorderMock)
		ctrlManMock.EXPECT().GetConfig().Return(restConfig)
		ctrlManMock.EXPECT().GetClient().Return(k8sClientMock)
		ctrlManMock.EXPECT().GetControllerOptions().Return(config.Controller{})
		ctrlManMock.EXPECT().GetScheme().Return(createScheme(t))
		ctrlManMock.EXPECT().GetLogger().Return(logr.New(logMock))
		ctrlManMock.EXPECT().Add(mock.Anything).Return(nil)
		ctrlManMock.EXPECT().GetCache().Return(nil)
		ctrlManMock.EXPECT().AddHealthzCheck("healthz", mock.Anything).Return(nil)
		ctrlManMock.EXPECT().AddReadyzCheck("readyz", mock.Anything).Return(nil)
		ctrlManMock.EXPECT().Start(mock.Anything).Return(nil)

		ctrl.NewManager = func(config *rest.Config, options manager.Options) (manager.Manager, error) {
			return ctrlManMock, nil
		}
		ctrl.GetConfigOrDie = func() *rest.Config {
			return restConfig
		}
		ctrl.SetupSignalHandler = func() context.Context {
			return testCtx
		}

		additionalImageGetterMock := newMockAdditionalImageGetter(t)
		additionalImageGetterMock.EXPECT().ImageForKey(testCtx, "kubectlImage").Return("bitnami/kubectl:1.27.7", nil)
		newAdditionalImageGetter = func(_ kubernetes.Interface, _ string) additionalimages.Getter {
			return additionalImageGetterMock
		}
		additionalImageUpdaterMock := newMockAdditionalImageUpdater(t)
		additionalImageUpdaterMock.EXPECT().Update(testCtx).Return(nil)
		newAdditionalImageUpdater = func(_ ecosystem.Interface, _ string, _ string, _ record.EventRecorder) additionalimages.Updater {
			return additionalImageUpdaterMock
		}

		flags := flag.NewFlagSet("operator", flag.ContinueOnError)

		// when
		err := startOperator(testCtx, flags, []string{})

		// then
		require.NoError(t, err)
	})
}

func createScheme(t *testing.T) *runtime.Scheme {
	t.Helper()

	scheme := runtime.NewScheme()
	gv, err := schema.ParseGroupVersion("k8s.cloudogu.com/v1")
	assert.NoError(t, err)

	scheme.AddKnownTypes(gv, &v1.Backup{}, &v1.Restore{}, &v1.BackupSchedule{})
	return scheme
}
