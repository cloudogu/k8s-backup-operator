package backup

import (
	"slices"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/k8s-backup-operator/internal/controller/backup/state"
)

// populateUnitStatus adds the metadata of the backup state and the provider references of the started units to the
// status. It only fills fields which are not set yet and never touches the state, reason and message of a unit.
// These are set by later steps and must not be reset when this step runs again.
// startedUnits contains the names of the provider backups which exist.
func populateUnitStatus(status *backupv1.BackupStatus, backupName string, backupState state.BackupState, startedUnits map[string]struct{}) {
	if status.BlueprintID == "" {
		status.BlueprintID = backupState.BlueprintId
	}
	if status.BackupCoverage == "" {
		status.BackupCoverage = backupState.Coverage
	}

	for _, unit := range backupState.Units {
		var providerRef *backupv1.ProviderRef
		providerBackupName := unitBackupName(backupName, unit)
		if _, started := startedUnits[providerBackupName]; started {
			providerRef = &backupv1.ProviderRef{Name: providerBackupName}
		}

		switch unit.Type {
		case backupv1.UnitTypeGlobalConfig:
			populateGlobalConfigStatus(status, backupState, providerRef)
		case backupv1.UnitTypeDoguV2:
			populateV2DogusStatus(status, backupState, providerRef)
		case backupv1.UnitTypeDogu:
			populateDoguStatus(status, backupState, unit.Name, providerRef)
		case backupv1.UnitTypeMisc:
			populateMiscStatus(status, backupState, providerRef)
		}
	}
}

func populateGlobalConfigStatus(status *backupv1.BackupStatus, backupState state.BackupState, providerRef *backupv1.ProviderRef) {
	if status.GlobalConfig == nil {
		status.GlobalConfig = &backupv1.GlobalConfigStatus{}
	}
	if status.GlobalConfig.FQDN == "" {
		status.GlobalConfig.FQDN = backupState.Fqdn
	}
	if status.GlobalConfig.Domain == "" {
		status.GlobalConfig.Domain = backupState.Domain
	}
	setProviderRefIfMissing(&status.GlobalConfig.UnitStatus, providerRef)
}

func populateV2DogusStatus(status *backupv1.BackupStatus, backupState state.BackupState, providerRef *backupv1.ProviderRef) {
	if status.V2Dogus == nil {
		status.V2Dogus = &backupv1.V2DogusUnitStatus{}
	}
	if len(status.V2Dogus.Dogus) == 0 && len(backupState.InstalledDogusV2) > 0 {
		status.V2Dogus.Dogus = slices.Clone(backupState.InstalledDogusV2)
	}
	setProviderRefIfMissing(&status.V2Dogus.UnitStatus, providerRef)
}

func populateDoguStatus(status *backupv1.BackupStatus, backupState state.BackupState, doguName string, providerRef *backupv1.ProviderRef) {
	index := slices.IndexFunc(status.Dogus, func(dogu backupv1.DoguUnitStatus) bool { return dogu.Name == doguName })
	if index < 0 {
		status.Dogus = append(status.Dogus, backupv1.DoguUnitStatus{Name: doguName})
		index = len(status.Dogus) - 1
	}

	dogu := &status.Dogus[index]
	if dogu.Version == "" {
		dogu.Version = installedDoguVersion(backupState.InstalledDogusV3, doguName)
	}
	setProviderRefIfMissing(&dogu.UnitStatus, providerRef)
}

func populateMiscStatus(status *backupv1.BackupStatus, backupState state.BackupState, providerRef *backupv1.ProviderRef) {
	if status.Misc == nil {
		status.Misc = &backupv1.UnitStatus{}
	}
	if len(status.Components) == 0 && len(backupState.Components) > 0 {
		status.Components = slices.Clone(backupState.Components)
	}
	setProviderRefIfMissing(status.Misc, providerRef)
}

func setProviderRefIfMissing(unitStatus *backupv1.UnitStatus, providerRef *backupv1.ProviderRef) {
	if unitStatus.ProviderRef == nil && providerRef != nil {
		unitStatus.ProviderRef = providerRef
	}
}

func installedDoguVersion(dogus []backupv1.DoguVersion, doguName string) string {
	for _, dogu := range dogus {
		if dogu.Name == doguName {
			return dogu.Version
		}
	}
	return ""
}
