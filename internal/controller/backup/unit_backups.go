package backup

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"maps"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/k8s-backup-operator/internal/controller/backup/state"
	veleroprovider "github.com/cloudogu/k8s-backup-operator/internal/provider/velero"
	componentv1 "github.com/cloudogu/k8s-component-lib/api/v1"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// maxObjectNameLength is the maximum length of metadata.name.
	maxObjectNameLength = 253
	// unitNameHashLength is the number of hex characters of the hash which makes truncated unit names unique.
	unitNameHashLength = 8

	doguNameLabelKey           = "k8s.cloudogu.com/dogu.name"
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
		typeAnnotations, typeErr := unitTypeAnnotations(backupState, unit)
		if typeErr != nil {
			return nil, typeErr
		}
		maps.Copy(annotations, typeAnnotations)

		selector, selectorErr := unitSelector(backup, backupState, unit)
		if selectorErr != nil {
			return nil, selectorErr
		}

		resources, resourcesErr := includedResources(unit)
		if resourcesErr != nil {
			return nil, resourcesErr
		}

		unitBackups = append(unitBackups, veleroprovider.CreateVeleroUnitBackupResource(
			backup.Namespace,
			unitBackupName(backup.Name, unit),
			backupStorageName,
			labels,
			annotations,
			selector,
			resources,
		))
	}

	return unitBackups, nil
}

// unitBackupName concatenates the backup name, the unit type and, for dogu units, the dogu name.
// Names exceeding the maximum object name length are truncated and suffixed with a hash of the full
// name, so that two long names which only differ at the end stay unique.
func unitBackupName(backupName string, unit backupv1.UnitIdentifier) string {
	name := fmt.Sprintf("%s-%s", backupName, unit.Type)
	if unit.Name != "" {
		name = fmt.Sprintf("%s-%s", name, unit.Name)
	}

	if len(name) <= maxObjectNameLength {
		return name
	}

	hash := sha256.Sum256([]byte(name))
	suffix := hex.EncodeToString(hash[:])[:unitNameHashLength]
	return fmt.Sprintf("%s-%s", name[:maxObjectNameLength-unitNameHashLength-1], suffix)
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

func unitTypeAnnotations(backupState state.BackupState, unit backupv1.UnitIdentifier) (map[string]string, error) {
	switch unit.Type {
	case backupv1.UnitTypeGlobalConfig:
		return backupState.GlobalConfigUnitAnnotations(), nil
	case backupv1.UnitTypeMisc:
		return backupState.MiscUnitAnnotations()
	default:
		return nil, nil
	}
}

func unitSelector(backup *backupv1.Backup, backupState state.BackupState, unit backupv1.UnitIdentifier) (*metav1.LabelSelector, error) {
	switch unit.Type {
	case backupv1.UnitTypeDogu:
		return &metav1.LabelSelector{MatchLabels: map[string]string{doguNameLabelKey: unit.Name}}, nil
	case backupv1.UnitTypeDoguV2:
		names := make([]string, 0, len(backupState.InstalledDogusV2))
		for _, dogu := range backupState.InstalledDogusV2 {
			names = append(names, dogu.Name)
		}
		return &metav1.LabelSelector{MatchExpressions: []metav1.LabelSelectorRequirement{
			{Key: doguNameLabelKey, Operator: metav1.LabelSelectorOpIn, Values: names},
		}}, nil
	case backupv1.UnitTypeGlobalConfig:
		return &metav1.LabelSelector{MatchLabels: map[string]string{typeLabelKey: globalConfigTypeLabelValue}}, nil
	case backupv1.UnitTypeMisc:
		// Resources with the backup-scope label belong to a dogu unit if they are labeled with a dogu.
		requirements := []metav1.LabelSelectorRequirement{
			{Key: backupScopeLabelKey, Operator: metav1.LabelSelectorOpExists},
			{Key: doguNameLabelKey, Operator: metav1.LabelSelectorOpDoesNotExist},
		}
		if len(backup.Spec.ExcludedComponents) > 0 {
			requirements = append(requirements, metav1.LabelSelectorRequirement{
				Key: componentv1.ComponentNameLabelKey, Operator: metav1.LabelSelectorOpNotIn, Values: backup.Spec.ExcludedComponents,
			})
		}
		return &metav1.LabelSelector{MatchExpressions: requirements}, nil
	default:
		return nil, fmt.Errorf("unknown unit type %q", unit.Type)
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
	case backupv1.UnitTypeMisc:
		return []string{"configmaps", "secrets", "persistentvolumeclaims"}, nil
	default:
		return nil, fmt.Errorf("failed to determine included resources for unit type %q", unit.Type)
	}
}
