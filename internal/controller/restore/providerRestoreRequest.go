package restore

import (
	componentv1 "github.com/cloudogu/k8s-component-lib/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	k8sv1 "github.com/cloudogu/k8s-backup-lib/api/v1"

	"github.com/cloudogu/k8s-backup-operator/internal/controller/restore/state"
	"github.com/cloudogu/k8s-backup-operator/internal/provider/velero"
)

// providerRestoreRequest returns the provider restore of the unit; a legacy restore restores the whole backup.
func providerRestoreRequest(restore *k8sv1.Restore, unit k8sv1.UnitIdentifier) velero.RestoreRequest {
	request := velero.RestoreRequest{
		Name:       state.ProviderRestoreName(restore.Name, unit),
		BackupName: state.ProviderBackupName(restore.Spec.BackupName, unit),
	}
	if unit.Type == state.UnitTypeLegacy {
		return request
	}

	request.Labels = unitLabels(restore.Spec.BackupName, unit)
	request.LabelSelector = restoreSelector(restore.Spec, unit)
	return request
}

// unitLabels returns the labels that let a provider restore be found by backup and unit.
func unitLabels(backupName string, unit k8sv1.UnitIdentifier) map[string]string {
	labels := map[string]string{
		k8sv1.BackupLabelKey:     k8sv1.GetTruncatedName(backupName),
		k8sv1.BackupTypeLabelKey: string(unit.Type),
	}
	if unit.Type == k8sv1.UnitTypeDogu {
		labels[k8sv1.DoguNameLabelKey] = unit.Name
	}
	return labels
}

// restoreSelector restricts the provider restore; nil restores the whole unit (the usual case).
func restoreSelector(spec k8sv1.RestoreSpec, unit k8sv1.UnitIdentifier) *metav1.LabelSelector {
	if unit.Type == k8sv1.UnitTypeComponents && len(spec.ExcludedComponents) > 0 {
		return &metav1.LabelSelector{MatchExpressions: []metav1.LabelSelectorRequirement{
			{Key: componentv1.ComponentNameLabelKey, Operator: metav1.LabelSelectorOpNotIn, Values: spec.ExcludedComponents},
		}}
	}
	return nil
}
