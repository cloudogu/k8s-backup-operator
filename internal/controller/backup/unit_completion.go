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

type unitCompletionState int

// completion states are sorted by priority, see setCompletionState
const (
	unitsInProgress unitCompletionState = iota
	unknownStatus
	unitsFailed
	unitsSucceeded
)

func setCompletionState(completionState *unitCompletionState, value unitCompletionState) {
	// completion states are sorted by priority
	// smaller is higher priority
	if *completionState > value {
		*completionState = value
	}
}

func determineUnitCompletion(
	ctx context.Context,
	backup *backupv1.Backup,
	actualUnits []velerov1.Backup,
	expectedState state.BackupState,
) (
	completionState unitCompletionState,
	backupStatusPatch statusUpdate,
	err error,
) {
	doguUnitMap, v2DogusBackup, globalConfigBackup, miscBackup := sortUnitTypes(ctx, backup, actualUnits)

	completionState = unitsSucceeded

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

func sortUnitTypes(ctx context.Context, backup *backupv1.Backup, actualUnits []velerov1.Backup) (
	doguUnitMap map[string]*velerov1.Backup,
	v2DogusBackup *velerov1.Backup,
	globalConfigBackup *velerov1.Backup,
	miscBackup *velerov1.Backup,
) {
	logger := log.FromContext(ctx).
		WithName("sortUnitTypes").
		WithValues("backup", backup.Name)

	doguUnitMap = make(map[string]*velerov1.Backup)
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

func unitStatusFromProviderBackup(backup *velerov1.Backup) (unitStatus backupv1.UnitStatus, completionState unitCompletionState) {
	if backup == nil {
		// treat missing same as in progress
		completionState = unitsInProgress
		unitStatus.State = backupv1.UnitStateMissing
		unitStatus.Reason = reasonProviderBackupMissing
		unitStatus.Message = "the provider backup for this unit is missing or might not have been created yet"
		return
	}

	if isProviderBackupInProgress(backup) {
		completionState = unitsInProgress
		unitStatus.State = ""
		unitStatus.Reason = reasonProviderBackupInProgress
		unitStatus.Message = fmt.Sprintf("provider backup is in progress with status phase: %s", backup.Status.Phase)
	} else if hasProviderBackupFailed(backup) {
		completionState = unitsFailed
		unitStatus.State = backupv1.UnitStateFailed
		unitStatus.Reason = reasonProviderBackupFailed
		unitStatus.Message = fmt.Sprintf("provider backup failed with status phase: %s", backup.Status.Phase)
	} else if hasProviderBackupSucceeded(backup) {
		completionState = unitsSucceeded
		unitStatus.State = backupv1.UnitStateSucceeded
		unitStatus.Reason = reasonProviderBackupSucceeded
		unitStatus.Message = ""
	} else {
		completionState = unknownStatus
		unitStatus.State = ""
		unitStatus.Reason = reasonProviderBackupUnknownStatus
		unitStatus.Message = fmt.Sprintf("provider backup has unknown status phase %s", backup.Status.Phase)
	}

	unitStatus.ProviderRef = &backupv1.ProviderRef{Name: backup.Name}
	return
}

func determineMiscUnitStatus(expectedState state.BackupState, miscBackup *velerov1.Backup) (*backupv1.UnitStatus, unitCompletionState) {
	if expectedState.ContainsUnitType(backupv1.UnitTypeMisc) {
		unitStatus, unitCompletionState := unitStatusFromProviderBackup(miscBackup)
		return &unitStatus, unitCompletionState
	}
	return nil, unitsSucceeded
}

func determineV2DoguUnitStatus(expectedState state.BackupState, v2DogusBackup *velerov1.Backup) (*backupv1.V2DogusUnitStatus, unitCompletionState, error) {
	if expectedState.ContainsUnitType(backupv1.UnitTypeDoguV2) {
		unitStatus, unitCompletionState := unitStatusFromProviderBackup(v2DogusBackup)

		var installedDogusV2 []backupv1.DoguVersion
		if v2DogusBackup != nil && v2DogusBackup.Annotations != nil {
			installedDogusV2String := v2DogusBackup.Annotations[backupv1.InstalledDogusV2AnnotationKey]
			err := json.Unmarshal([]byte(installedDogusV2String), &installedDogusV2)
			if err != nil {
				return nil, 0, fmt.Errorf("unmarshal installed dogus v2 from unit: %w", err)
			}
		}

		return &backupv1.V2DogusUnitStatus{
			UnitStatus: unitStatus,
			Dogus:      installedDogusV2,
		}, unitCompletionState, nil
	}
	return nil, unitsSucceeded, nil
}

func determineGlobalConfigUnitStatus(expectedState state.BackupState, globalConfigBackup *velerov1.Backup) (*backupv1.GlobalConfigStatus, unitCompletionState) {
	if expectedState.ContainsUnitType(backupv1.UnitTypeGlobalConfig) {
		unitStatus, unitCompletionState := unitStatusFromProviderBackup(globalConfigBackup)

		var fqdn, domain string
		if globalConfigBackup != nil && globalConfigBackup.Annotations != nil {
			fqdn = globalConfigBackup.Annotations[backupv1.FQDNAnnotationKey]
			domain = globalConfigBackup.Annotations[backupv1.DomainAnnotationKey]
		}

		return &backupv1.GlobalConfigStatus{
			UnitStatus: unitStatus,
			FQDN:       fqdn,
			Domain:     domain,
		}, unitCompletionState
	}
	return nil, unitsSucceeded
}

func determineV3DoguUnitStatus(expectedState state.BackupState, doguUnitMap map[string]*velerov1.Backup) ([]backupv1.DoguUnitStatus, unitCompletionState) {
	doguUnitsStatus := make([]backupv1.DoguUnitStatus, 0, len(expectedState.InstalledDogusV3))
	completionState := unitsSucceeded
	for _, expectedDogu := range expectedState.InstalledDogusV3 {
		doguUnit := doguUnitMap[expectedDogu.Name]
		unitStatus, unitCompletionState := unitStatusFromProviderBackup(doguUnit)
		setCompletionState(&completionState, unitCompletionState)

		var version string
		if doguUnit != nil && doguUnit.Labels != nil {
			version = doguUnit.Labels[backupv1.DoguVersionLabelKey]
		}

		doguUnitsStatus = append(doguUnitsStatus, backupv1.DoguUnitStatus{
			Name:       expectedDogu.Name,
			Version:    version,
			UnitStatus: unitStatus,
		})
	}

	return doguUnitsStatus, completionState
}
