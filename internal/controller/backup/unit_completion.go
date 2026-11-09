package backup

import (
	"context"
	"encoding/json/v2"
	"fmt"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/k8s-backup-operator/internal/controller/backup/state"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type UnitCompletionState int

func (u UnitCompletionState) String() string {
	switch u {
	case UnitsMissing:
		return "some units are missing and might not have been created yet"
	case UnitsInProgress:
		return "some units are still in progress"
	case UnitsFailed:
		return "some units have failed"
	case UnitsSucceeded:
		return "all units have succeeded"
	case UnknownStatus:
		return "some units have an unknown provider status phase"
	}
	return fmt.Sprintf("unknown completion state: %d", u)
}

const (
	UnitsMissing UnitCompletionState = iota
	UnitsInProgress
	UnitsFailed
	UnitsSucceeded
	UnknownStatus
)

func setCompletionState(completionState *UnitCompletionState, value UnitCompletionState) {
	if *completionState == UnitsSucceeded {
		*completionState = value
	}
}

func determineUnitCompletion(
	ctx context.Context,
	backup *backupv1.Backup,
	actualUnits []velerov1.Backup,
	expectedState state.BackupState,
) (
	completionState UnitCompletionState,
	backupStatusPatch statusUpdate,
	err error,
) {
	doguUnitMap, v2DogusBackup, globalConfigBackup, miscBackup := sortUnitTypes(ctx, backup, actualUnits)

	completionState = UnitsSucceeded

	globalConfigStatus, configCompletionState := determineGlobalConfigUnitStatus(expectedState, globalConfigBackup)
	setCompletionState(&completionState, configCompletionState)

	doguUnitsStatus, v3CompletionState := determineV3DoguUnitStatus(expectedState, doguUnitMap)
	setCompletionState(&completionState, v3CompletionState)

	doguV2UnitStatus, v2CompletionState, err := determineV2DoguUnitStatus(expectedState, v2DogusBackup)
	if err != nil {
		return 0, nil, err
	}

	setCompletionState(&completionState, v2CompletionState)

	miscUnitStatus, miscCompletionState := determineMiscUnitStatus(expectedState, miscBackup)
	setCompletionState(&completionState, miscCompletionState)

	return completionState, func(status *backupv1.BackupStatus) {
		status.Dogus = doguUnitsStatus
		status.GlobalConfig = globalConfigStatus
		status.Misc = miscUnitStatus
		status.V2Dogus = doguV2UnitStatus
	}, nil
}

func sortUnitTypes(ctx context.Context, backup *backupv1.Backup, actualUnits []velerov1.Backup) (map[string]*velerov1.Backup, *velerov1.Backup, *velerov1.Backup, *velerov1.Backup) {
	logger := log.FromContext(ctx).
		WithName("sortUnitTypes").
		WithValues("backup", backup.Name)

	doguUnitMap := make(map[string]*velerov1.Backup)
	var v2DogusBackup *velerov1.Backup
	var globalConfigBackup *velerov1.Backup
	var miscBackup *velerov1.Backup
	for _, unit := range actualUnits {
		switch backupv1.UnitType(unit.Labels[backupv1.BackupTypeLabelKey]) {
		case backupv1.UnitTypeDogu:
			name := unit.Labels[backupv1.DoguNameLabelKey]
			doguUnitMap[name] = &unit
		case backupv1.UnitTypeDoguV2:
			v2DogusBackup = &unit
		case backupv1.UnitTypeGlobalConfig:
			globalConfigBackup = &unit
		case backupv1.UnitTypeMisc:
			miscBackup = &unit
		default:
			logger.Info("Unknown backup type", "type", unit.Labels[backupv1.BackupTypeLabelKey])
			continue
		}
	}
	return doguUnitMap, v2DogusBackup, globalConfigBackup, miscBackup
}

func unitStatusFromProviderBackup(backup *velerov1.Backup) (unitStatus backupv1.UnitStatus, completionState UnitCompletionState) {
	if backup == nil {
		completionState = UnitsMissing
		unitStatus.State = backupv1.UnitStateMissing
		unitStatus.Reason = reasonProviderBackupMissing
		unitStatus.Message = "the provider backup for this unit is missing or might not have been created yet"
		return
	}

	if isProviderBackupInProgress(backup) {
		completionState = UnitsInProgress
		unitStatus.State = ""
		unitStatus.Reason = reasonProviderBackupInProgress
		unitStatus.Message = fmt.Sprintf("provider backup is in progress with status phase: %s", backup.Status.Phase)
	} else if hasProviderBackupFailed(backup) {
		completionState = UnitsFailed
		unitStatus.State = backupv1.UnitStateFailed
		unitStatus.Reason = reasonProviderBackupFailed
		unitStatus.Message = fmt.Sprintf("provider backup failed with status phase: %s", backup.Status.Phase)
	} else if hasProviderBackupSucceeded(backup) {
		completionState = UnitsSucceeded
		unitStatus.State = backupv1.UnitStateSucceeded
		unitStatus.Reason = reasonProviderBackupSucceeded
		unitStatus.Message = ""
	}

	completionState = UnknownStatus
	unitStatus.State = ""
	unitStatus.Reason = reasonProviderBackupUnknownStatus
	unitStatus.Message = fmt.Sprintf("provider backup has unknown status phase %s", backup.Status.Phase)

	unitStatus.ProviderRef = &backupv1.ProviderRef{Name: backup.Name}
	return
}

func determineMiscUnitStatus(expectedState state.BackupState, miscBackup *velerov1.Backup) (*backupv1.UnitStatus, UnitCompletionState) {
	if expectedState.ContainsUnitType(backupv1.UnitTypeMisc) {
		unitStatus, unitCompletionState := unitStatusFromProviderBackup(miscBackup)
		return &unitStatus, unitCompletionState
	}
	return nil, UnitsSucceeded
}

func determineV2DoguUnitStatus(expectedState state.BackupState, v2DogusBackup *velerov1.Backup) (*backupv1.V2DogusUnitStatus, UnitCompletionState, error) {
	if expectedState.ContainsUnitType(backupv1.UnitTypeDoguV2) {
		unitStatus, unitCompletionState := unitStatusFromProviderBackup(v2DogusBackup)
		installedDogusV2String := v2DogusBackup.Annotations[backupv1.InstalledDogusV2AnnotationKey]
		var installedDogusV2 []backupv1.DoguVersion
		err := json.Unmarshal([]byte(installedDogusV2String), &installedDogusV2)
		if err != nil {
			return nil, 0, fmt.Errorf("unmarshal installed dogus v2 from unit: %w", err)
		}

		return &backupv1.V2DogusUnitStatus{
			UnitStatus: unitStatus,
			Dogus:      installedDogusV2,
		}, unitCompletionState, nil
	}
	return nil, UnitsSucceeded, nil
}

func determineGlobalConfigUnitStatus(expectedState state.BackupState, globalConfigBackup *velerov1.Backup) (*backupv1.GlobalConfigStatus, UnitCompletionState) {
	if expectedState.ContainsUnitType(backupv1.UnitTypeGlobalConfig) {
		unitStatus, unitCompletionState := unitStatusFromProviderBackup(globalConfigBackup)
		globalConfigStatus := &backupv1.GlobalConfigStatus{
			UnitStatus: unitStatus,
			FQDN:       globalConfigBackup.Annotations[backupv1.FQDNAnnotationKey],
			Domain:     globalConfigBackup.Annotations[backupv1.DomainAnnotationKey],
		}
		return globalConfigStatus, unitCompletionState
	}
	return nil, UnitsSucceeded
}

func determineV3DoguUnitStatus(expectedState state.BackupState, doguUnitMap map[string]*velerov1.Backup) ([]backupv1.DoguUnitStatus, UnitCompletionState) {
	doguUnitsStatus := make([]backupv1.DoguUnitStatus, 0, len(expectedState.InstalledDogusV3))
	completionState := UnitsSucceeded
	for _, expectedDogu := range expectedState.InstalledDogusV3 {
		doguUnit := doguUnitMap[expectedDogu.Name]
		unitStatus, unitCompletionState := unitStatusFromProviderBackup(doguUnit)
		setCompletionState(&completionState, unitCompletionState)

		version := doguUnit.Labels[backupv1.DoguVersionLabelKey]
		doguUnitsStatus = append(doguUnitsStatus, backupv1.DoguUnitStatus{
			Name:       expectedDogu.Name,
			Version:    version,
			UnitStatus: unitStatus,
		})
	}
	return doguUnitsStatus, completionState
}
