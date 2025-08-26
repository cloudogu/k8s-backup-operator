package provider

import (
	"context"
	"fmt"
	"github.com/cloudogu/k8s-backup-operator/pkg/requeue"
	"github.com/cloudogu/k8s-backup-operator/pkg/velero"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"

	k8sv1 "github.com/cloudogu/k8s-backup-lib/pkg/api/v1"
)

var knownProviders = []k8sv1.Provider{k8sv1.ProviderVelero}

var NewVeleroProvider = func(k8sClient K8sClient, ecosystemClientSet EcosystemClientSet, recorder EventRecorder, namespace string) Provider {
	return velero.NewDefaultProvider(k8sClient, ecosystemClientSet.Discovery(), namespace, recorder)
}

// Get returns the provider by the given name and calls a function on the provider object to check if it is ready.
func Get(ctx context.Context, object runtime.Object, name k8sv1.Provider, namespace string, recorder EventRecorder, k8sClient K8sClient, ecosystemClientSet EcosystemClientSet) (Provider, error) {
	var provider Provider
	switch name {
	case "":
		recorder.Event(object, v1.EventTypeNormal, k8sv1.ProviderSelectEventReason, "No provider given. Select velero as default provider.")
		fallthrough
	case k8sv1.ProviderVelero:
		provider = NewVeleroProvider(k8sClient, ecosystemClientSet, recorder, namespace)
	default:
		return nil, fmt.Errorf("unknown provider %s", name)
	}

	err := provider.CheckReady(ctx)
	if err != nil {
		return nil, &requeue.GenericRequeueableError{
			Err:    err,
			ErrMsg: fmt.Sprintf("provider %s is not ready", name),
		}
	}

	return provider, nil
}

// GetAll returns all known providers that are functional and ready.
func GetAll(ctx context.Context, namespace string, recorder EventRecorder, k8sClient K8sClient, ecosystemClientSet EcosystemClientSet) []Provider {
	logger := log.FromContext(ctx).WithName("get all providers")

	logger.Info(fmt.Sprintf("getting all known providers: %q", knownProviders))
	var providers []Provider
	for _, providerName := range knownProviders {
		provider, err := Get(ctx, nil, providerName, namespace, recorder, k8sClient, ecosystemClientSet)
		if err != nil {
			logger.Info(fmt.Sprintf("skipping provider %q due to error: %s", providerName, err.Error()))
			continue
		}

		providers = append(providers, provider)
	}

	return providers
}
