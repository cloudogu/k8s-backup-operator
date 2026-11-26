package backup

import (
	"fmt"
	"maps"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/k8s-backup-operator/internal/controller/backup/state"
	veleroprovider "github.com/cloudogu/k8s-backup-operator/internal/provider/velero"
	componentv1 "github.com/cloudogu/k8s-component-lib/api/v1"
	doguv2 "github.com/cloudogu/k8s-dogu-lib/v3/api/v2"
	doguv3 "github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	backupScopeLabelKey        = "k8s.cloudogu.com/backup-scope"
	typeLabelKey               = "k8s.cloudogu.com/type"
	globalConfigTypeLabelValue = "global-config"
)

// desiredUnitBackups returns a provider backup for each unit of the backup state.
func desiredUnitBackups(backup *backupv1.Backup, backupState state.BackupState, backupStorageName string) ([]*velerov1.Backup, error) {
	commonAnnotations, err := backupState.CommonUnitAnnotations()
	if err != nil {
		return nil, err
	}

	unitBackups := make([]*velerov1.Backup, 0, len(backupState.Units))
	for _, unit := range backupState.Units {
		labels := unitLabels(backupState, unit)

		annotations := maps.Clone(commonAnnotations)
		typeAnnotations := unitTypeAnnotations(backupState, unit)
		maps.Copy(annotations, typeAnnotations)

		selector, orSelectors, selectorErr := unitSelector(backupState, unit)
		if selectorErr != nil {
			return nil, selectorErr
		}

		resources, resourcesErr := includedResources(unit)
		if resourcesErr != nil {
			return nil, resourcesErr
		}

		resource, backupErr := veleroprovider.CreateVeleroUnitBackupResource(
			backup.Namespace,
			veleroprovider.UnitObjectName(backup.Name, unit),
			backupStorageName,
			labels,
			annotations,
			selector,
			orSelectors,
			resources,
		)
		if backupErr != nil {
			return nil, fmt.Errorf("create provider backup for unit %q: %w", unit.Type, backupErr)
		}
		unitBackups = append(unitBackups, resource)
	}

	return unitBackups, nil
}

func unitLabels(backupState state.BackupState, unit backupv1.UnitIdentifier) map[string]string {
	labels := maps.Clone(defaultLabels)
	maps.Copy(labels, backupState.CommonLabels())
	labels[backupv1.BackupTypeLabelKey] = string(unit.Type)

	if unit.Type == backupv1.UnitTypeDogu {
		labels[backupv1.DoguNameLabelKey] = unit.Name
		if version := installedDoguVersion(backupState.InstalledDogusV3, unit.Name); version != "" {
			labels[backupv1.DoguVersionLabelKey] = version
		}
	}

	return labels
}

func unitTypeAnnotations(backupState state.BackupState, unit backupv1.UnitIdentifier) map[string]string {
	switch unit.Type {
	case backupv1.UnitTypeGlobalConfig:
		return backupState.GlobalConfigUnitAnnotations()
	default:
		return nil
	}
}

func unitSelector(backupState state.BackupState, unit backupv1.UnitIdentifier) (*metav1.LabelSelector, []*metav1.LabelSelector, error) {
	switch unit.Type {
	case backupv1.UnitTypeDogu:
		return &metav1.LabelSelector{MatchLabels: map[string]string{doguv3.DoguLabelName: unit.Name}}, nil, nil
	case backupv1.UnitTypeDoguV2:
		// The dogu-operator does not add the k8s.cloudogu.com/dogu.name label to resources of v2 dogus; they only
		// carry the legacy dogu.name label.
		names := make([]string, 0, len(backupState.InstalledDogusV2))
		for _, dogu := range backupState.InstalledDogusV2 {
			names = append(names, dogu.Name)
		}
		doguLabelSelector := &metav1.LabelSelector{MatchExpressions: []metav1.LabelSelectorRequirement{
			{Key: doguv3.DoguLabelName, Operator: metav1.LabelSelectorOpIn, Values: names},
		}}
		v2DoguLabelSelector := &metav1.LabelSelector{MatchExpressions: []metav1.LabelSelectorRequirement{
			{Key: doguv2.DoguLabelName, Operator: metav1.LabelSelectorOpIn, Values: names},
		}}
		return nil, []*metav1.LabelSelector{doguLabelSelector, v2DoguLabelSelector}, nil
	case backupv1.UnitTypeGlobalConfig:
		return &metav1.LabelSelector{MatchLabels: map[string]string{typeLabelKey: globalConfigTypeLabelValue}}, nil, nil
	case backupv1.UnitTypeComponents:
		// The components unit only contains resources of components labeled with the backup-scope label.
		if len(backupState.Components) == 0 {
			return nil, nil, fmt.Errorf("components unit requires at least one component")
		}
		names := make([]string, 0, len(backupState.Components))
		for _, component := range backupState.Components {
			names = append(names, component.Name)
		}
		return &metav1.LabelSelector{MatchExpressions: []metav1.LabelSelectorRequirement{
			{Key: backupScopeLabelKey, Operator: metav1.LabelSelectorOpExists},
			{Key: componentv1.ComponentNameLabelKey, Operator: metav1.LabelSelectorOpIn, Values: names},
		}}, nil, nil
	default:
		return nil, nil, fmt.Errorf("unknown unit type %q", unit.Type)
	}
}

// includedResources returns the resources which are backed up for the unit.
// Persistent volumes and CSI snapshot resources (volumesnapshots, volumesnapshotcontents, volumesnapshotclasses) are
// not listed on purpose: Velero adds them as additional items of the persistent volume claims. Cluster-scoped
// additional items are only skipped if they are excluded explicitly, and the CSI actions mark their additional items
// with "backup.velero.io/must-include-additional-items", which bypasses the resource filter.
func includedResources(unit backupv1.UnitIdentifier) ([]string, error) {
	switch unit.Type {
	case backupv1.UnitTypeGlobalConfig:
		return []string{"configmaps"}, nil
	case backupv1.UnitTypeDogu, backupv1.UnitTypeDoguV2:
		return []string{"configmaps", "secrets", "persistentvolumeclaims", "dogus.k8s.cloudogu.com"}, nil
	case backupv1.UnitTypeComponents:
		// Service account producers are backed up so that the restore of a producer component deletes and recreates
		// its producer. This makes the service-account-operator delete and recreate the secrets of the requests.
		return []string{"configmaps", "secrets", "persistentvolumeclaims", "serviceaccountproducers.k8s.cloudogu.com"}, nil
	default:
		return nil, fmt.Errorf("failed to determine included resources for unit type %q", unit.Type)
	}
}
