package state

import (
	"fmt"
	"slices"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"k8s.io/utils/ptr"

	"github.com/cloudogu/k8s-backup-operator/internal/provider/velero"
)

// UnitTypeLegacy is the operator-internal unit that restores a legacy backup as a whole.
const UnitTypeLegacy backupv1.UnitType = "legacy"

// ProviderRestoreName returns the provider restore name of the unit; a legacy restore keeps the restore name.
func ProviderRestoreName(restoreName string, unit backupv1.UnitIdentifier) string {
	if unit.Type == UnitTypeLegacy {
		return restoreName
	}
	return velero.UnitObjectName(restoreName, unit)
}

// ProviderBackupName returns the provider backup name of the unit; a legacy backup keeps the backup name.
func ProviderBackupName(backupName string, unit backupv1.UnitIdentifier) string {
	if unit.Type == UnitTypeLegacy {
		return backupName
	}
	return velero.UnitObjectName(backupName, unit)
}

// Kind tells how the backup of a restore was resolved.
type Kind string

const (
	KindNotFound Kind = "NotFound"
	KindLegacy   Kind = "Legacy"
	KindUnits    Kind = "Units"
)

// RestoreState captures the backup of a restore the restore steps decide on.
type RestoreState struct {
	Kind Kind
	// Units are the units of the backup in restore order; a legacy backup has the single unit legacy.
	Units []UnitBackup
	// V2Dogus are the dogus of the V2 blob.
	V2Dogus []string
	// V3Dogus are the dogus of all v3 dogu units.
	V3Dogus []string
	// Components are the components of the components unit.
	Components []string
}

// UnitBackup is a unit of the backup with its provider backup.
type UnitBackup struct {
	backupv1.UnitIdentifier
	// ProviderBackup is nil if the unit's provider backup is missing.
	ProviderBackup *ProviderBackup
}

// ProviderBackup is the provider backup of a unit as far as the restore needs it.
type ProviderBackup struct {
	Name  string
	Phase velerov1.BackupPhase
}

// UnitsToRestore returns the units without the ignored or excluded ones, in restore order.
func (s RestoreState) UnitsToRestore(spec backupv1.RestoreSpec) []UnitBackup {
	return slices.DeleteFunc(slices.Clone(s.Units), func(unit UnitBackup) bool { return s.isSkipped(spec, unit) })
}

// RestoredComponents returns the components without the excluded ones.
func (s RestoreState) RestoredComponents(spec backupv1.RestoreSpec) []string {
	return slices.DeleteFunc(slices.Clone(s.Components), func(component string) bool {
		return slices.Contains(spec.ExcludedComponents, component)
	})
}

// RestoredDogus returns the V2 and V3 dogus without the ignored or excluded ones.
func (s RestoreState) RestoredDogus(spec backupv1.RestoreSpec) (v2Dogus, v3Dogus []string) {
	if !ptr.Deref(spec.Options, backupv1.UnitOptions{}).IgnoreV2Dogus {
		v2Dogus = slices.Clone(s.V2Dogus)
	}
	v3Dogus = slices.DeleteFunc(slices.Clone(s.V3Dogus), func(dogu string) bool {
		return slices.Contains(spec.ExcludedDogus, dogu)
	})
	return v2Dogus, v3Dogus
}

// InvalidExclusions describes each exclusion or option not matching the backup; empty if all match.
func (s RestoreState) InvalidExclusions(spec backupv1.RestoreSpec) []string {
	if s.Kind == KindLegacy {
		if len(spec.ExcludedDogus) > 0 || len(spec.ExcludedComponents) > 0 || ptr.Deref(spec.Options, backupv1.UnitOptions{}) != (backupv1.UnitOptions{}) {
			return []string{"exclusions and options are not supported for a legacy backup"}
		}
		return nil
	}

	var invalid []string
	for _, dogu := range spec.ExcludedDogus {
		switch {
		case slices.Contains(s.V2Dogus, dogu):
			invalid = append(invalid, fmt.Sprintf("excludedDogus: %s is a V2 dogu", dogu))
		case !slices.ContainsFunc(s.Units, func(unit UnitBackup) bool {
			return unit.Type == backupv1.UnitTypeDogu && unit.Name == dogu
		}):
			invalid = append(invalid, fmt.Sprintf("excludedDogus: %s is not in the backup", dogu))
		}
	}
	for _, component := range spec.ExcludedComponents {
		if !slices.Contains(s.Components, component) {
			invalid = append(invalid, fmt.Sprintf("excludedComponents: %s is not in the backup", component))
		}
	}
	return invalid
}

// IsUsable reports whether the unit's provider backup can be restored.
func (u UnitBackup) IsUsable() bool {
	return u.ProviderBackup != nil && u.ProviderBackup.Phase == velerov1.BackupPhaseCompleted
}

// SkipOption names the exclusion or option that skips the unit; empty for the legacy unit.
func (u UnitBackup) SkipOption() string {
	switch u.Type {
	case backupv1.UnitTypeGlobalConfig:
		return "options.ignoreGlobalConfig"
	case backupv1.UnitTypeComponents:
		return "excludedComponents"
	case backupv1.UnitTypeDoguV2:
		return "options.ignoreV2Dogus"
	case backupv1.UnitTypeDogu:
		return "excludedDogus"
	default:
		return ""
	}
}

func (s RestoreState) isSkipped(spec backupv1.RestoreSpec, unit UnitBackup) bool {
	options := ptr.Deref(spec.Options, backupv1.UnitOptions{})
	switch unit.Type {
	case backupv1.UnitTypeGlobalConfig:
		return options.IgnoreGlobalConfig
	case backupv1.UnitTypeComponents:
		return len(s.RestoredComponents(spec)) == 0
	case backupv1.UnitTypeDoguV2:
		return options.IgnoreV2Dogus
	case backupv1.UnitTypeDogu:
		return slices.Contains(spec.ExcludedDogus, unit.Name)
	default:
		return false
	}
}
