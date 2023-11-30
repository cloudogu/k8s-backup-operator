package provider

import (
	"context"
	"fmt"
	"github.com/cloudogu/k8s-backup-operator/pkg/requeue"
	"github.com/cloudogu/k8s-backup-operator/pkg/velero"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	k8sv1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
)

var NewVeleroProvider = func(ecosystemClientSet EcosystemClientSet, recorder EventRecorder, namespace string) (Provider, error) {
	return velero.NewDefaultProvider(ecosystemClientSet, namespace, recorder)
}

// GetProvider returns the provider by the given name and calls a function on the provider object to check if it is ready.
func GetProvider(ctx context.Context, object runtime.Object, name k8sv1.Provider, namespace string, recorder EventRecorder, ecosystemClientSet EcosystemClientSet) (Provider, error) {
	var provider Provider
	var err error
	switch name {
	case "":
		recorder.Event(object, v1.EventTypeNormal, k8sv1.ProviderSelectEventReason, "No provider given. Select velero as default provider.")
		fallthrough
	case k8sv1.ProviderVelero:
		provider, err = NewVeleroProvider(ecosystemClientSet, recorder, namespace)
		if err != nil {
			return nil, fmt.Errorf("failed to create velero provider: %w", err)
		}
	default:
		return nil, fmt.Errorf("unknown provider %s", name)
	}

	err = provider.CheckReady(ctx)
	if err != nil {
		return nil, &requeue.GenericRequeueableError{
			Err:    err,
			ErrMsg: fmt.Sprintf("provider %s is not ready", name),
		}
	}

	return provider, nil
}
