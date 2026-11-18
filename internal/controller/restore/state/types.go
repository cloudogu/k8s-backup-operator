package state

import (
	"slices"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"

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
	// Components are the components of the misc unit.
	Components []string
}

// UnitBackup is a unit of the backup with its provider backup.
type UnitBackup struct {
	backupv1.UnitIdentifier
	// ProviderBackup is nil if the unit's provider backup is missing.
	ProviderBackup *velerov1.Backup
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

func (s RestoreState) isSkipped(spec backupv1.RestoreSpec, unit UnitBackup) bool {
	switch unit.Type {
	case backupv1.UnitTypeGlobalConfig:
		return spec.Options.IgnoreGlobalConfig
	case backupv1.UnitTypeMisc:
		// the misc unit holds only components, so without one left there is nothing to restore
		return spec.Options.IgnoreMisc || (unit.ProviderBackup != nil && len(s.RestoredComponents(spec)) == 0)
	case backupv1.UnitTypeDoguV2:
		return spec.Options.IgnoreV2Dogus
	case backupv1.UnitTypeDogu:
		return slices.Contains(spec.ExcludedDogus, unit.Name)
	default:
		return false
	}
}
