package backup

import (
	"context"
	"fmt"
	"github.com/cloudogu/k8s-backup-operator/pkg/api/ecosystem"
	"github.com/cloudogu/k8s-backup-operator/pkg/provider/velero"

	k8sv1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
)

var newVeleroProvider = func(client ecosystem.BackupInterface, recorder eventRecorder, namespace string) (Provider, error) {
	return velero.New(client, recorder, namespace)
}

func getBackupProvider(ctx context.Context, backup *k8sv1.Backup, client ecosystemBackupInterface, recorder eventRecorder) (Provider, error) {
	var provider Provider
	var err error
	switch backup.Spec.Provider {
	case k8sv1.ProviderVelero:
		provider, err = newVeleroProvider(client, recorder, backup.Namespace)
		if err != nil {
			return nil, fmt.Errorf("failed to create velero provider: %w", err)
		}
	default:
		return nil, fmt.Errorf("unknown backup provider %s", backup.Spec.Provider)
	}

	err = provider.CheckReady(ctx)
	if err != nil {
		return nil, fmt.Errorf("provider %s is not ready: %w", backup.Spec.Provider, err)
	}

	return provider, nil
}
