package state

import (
	"testing"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/stretchr/testify/assert"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
)

var (
	globalConfigUnit = UnitBackup{Type: backupv1.UnitTypeGlobalConfig, ProviderBackup: &ProviderBackup{}}
	componentsUnit   = UnitBackup{Type: backupv1.UnitTypeComponents, ProviderBackup: &ProviderBackup{}}
	doguV2Unit       = UnitBackup{Type: backupv1.UnitTypeDoguV2, ProviderBackup: &ProviderBackup{}}
	postfixUnit      = UnitBackup{Type: backupv1.UnitTypeDogu, Name: "postfix", ProviderBackup: &ProviderBackup{}}
	redmineUnit      = UnitBackup{Type: backupv1.UnitTypeDogu, Name: "redmine", ProviderBackup: &ProviderBackup{}}
)

func unitState() RestoreState {
	return RestoreState{
		Kind:       KindUnits,
		Units:      []UnitBackup{globalConfigUnit, componentsUnit, doguV2Unit, postfixUnit, redmineUnit},
		Components: []string{"k8s-loki", "k8s-velero"},
	}
}

func TestRestoreState_UnitsToRestore(t *testing.T) {
	// the components are known from the annotations of the other provider backups
	missingComponents := unitState()
	missingComponents.Units[1].ProviderBackup = nil

	withoutComponents := unitState()
	withoutComponents.Components = nil

	tests := []struct {
		name  string
		state RestoreState
		spec  backupv1.RestoreSpec
		want  []UnitBackup
	}{
		{name: "all units", state: unitState(), want: []UnitBackup{globalConfigUnit, componentsUnit, doguV2Unit, postfixUnit, redmineUnit}},
		{
			name:  "ignoreGlobalConfig drops the global-config unit",
			state: unitState(),
			spec:  backupv1.RestoreSpec{Options: &backupv1.UnitOptions{IgnoreGlobalConfig: true}},
			want:  []UnitBackup{componentsUnit, doguV2Unit, postfixUnit, redmineUnit},
		},
		{
			name:  "ignoreV2Dogus drops the doguv2 unit",
			state: unitState(),
			spec:  backupv1.RestoreSpec{Options: &backupv1.UnitOptions{IgnoreV2Dogus: true}},
			want:  []UnitBackup{globalConfigUnit, componentsUnit, postfixUnit, redmineUnit},
		},
		{
			name:  "excludedDogus drops V3 dogu units only",
			state: unitState(),
			spec:  backupv1.RestoreSpec{ExcludedDogus: []string{"redmine", "cockpit"}},
			want:  []UnitBackup{globalConfigUnit, componentsUnit, doguV2Unit, postfixUnit},
		},
		{
			name:  "some components excluded keeps the components unit",
			state: unitState(),
			spec:  backupv1.RestoreSpec{ExcludedComponents: []string{"k8s-loki"}},
			want:  []UnitBackup{globalConfigUnit, componentsUnit, doguV2Unit, postfixUnit, redmineUnit},
		},
		{
			name:  "all components excluded drops the components unit",
			state: unitState(),
			spec:  backupv1.RestoreSpec{ExcludedComponents: []string{"k8s-loki", "k8s-velero"}},
			want:  []UnitBackup{globalConfigUnit, doguV2Unit, postfixUnit, redmineUnit},
		},
		{
			name:  "backup without components drops the components unit",
			state: withoutComponents,
			want:  []UnitBackup{globalConfigUnit, doguV2Unit, postfixUnit, redmineUnit},
		},
		{
			name:  "missing components backup keeps the components unit",
			state: missingComponents,
			want:  []UnitBackup{globalConfigUnit, {Type: backupv1.UnitTypeComponents}, doguV2Unit, postfixUnit, redmineUnit},
		},
		{
			name:  "missing components backup is dropped if all components are excluded",
			state: missingComponents,
			spec:  backupv1.RestoreSpec{ExcludedComponents: []string{"k8s-loki", "k8s-velero"}},
			want:  []UnitBackup{globalConfigUnit, doguV2Unit, postfixUnit, redmineUnit},
		},
		{
			name:  "legacy unit is kept",
			state: RestoreState{Kind: KindLegacy, Units: []UnitBackup{{Type: UnitTypeLegacy}}},
			spec:  backupv1.RestoreSpec{ExcludedDogus: []string{"redmine"}, Options: &backupv1.UnitOptions{IgnoreGlobalConfig: true}},
			want:  []UnitBackup{{Type: UnitTypeLegacy}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.state.UnitsToRestore(tt.spec))
		})
	}
}

func TestRestoreState_UnitsToRestore_KeepsTheState(t *testing.T) {
	state := unitState()

	state.UnitsToRestore(backupv1.RestoreSpec{Options: &backupv1.UnitOptions{IgnoreGlobalConfig: true}})

	assert.Equal(t, unitState(), state)
}

func TestRestoreState_RestoredComponents(t *testing.T) {
	state := unitState()

	assert.Equal(t, []string{"k8s-loki", "k8s-velero"}, state.RestoredComponents(backupv1.RestoreSpec{}))
	assert.Equal(t, []string{"k8s-velero"}, state.RestoredComponents(backupv1.RestoreSpec{ExcludedComponents: []string{"k8s-loki"}}))
	assert.Empty(t, state.RestoredComponents(backupv1.RestoreSpec{ExcludedComponents: []string{"k8s-loki", "k8s-velero"}}))
}

func TestProviderRestoreName(t *testing.T) {
	tests := []struct {
		name        string
		restoreName string
		unit        backupv1.UnitIdentifier
		want        string
	}{
		{name: "legacy", restoreName: "myrestore", unit: backupv1.UnitIdentifier{Type: UnitTypeLegacy}, want: "myrestore"},
		{name: "global-config", restoreName: "myrestore", unit: backupv1.UnitIdentifier{Type: backupv1.UnitTypeGlobalConfig}, want: "myrestore-global-config"},
		{name: "components", restoreName: "myrestore", unit: backupv1.UnitIdentifier{Type: backupv1.UnitTypeComponents}, want: "myrestore-components"},
		{name: "doguv2", restoreName: "myrestore", unit: backupv1.UnitIdentifier{Type: backupv1.UnitTypeDoguV2}, want: "myrestore-doguv2"},
		{name: "dogu", restoreName: "myrestore", unit: backupv1.UnitIdentifier{Type: backupv1.UnitTypeDogu, Name: "redmine"}, want: "myrestore-dogu-redmine"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ProviderRestoreName(tt.restoreName, tt.unit))
		})
	}
}

func TestProviderBackupName(t *testing.T) {
	tests := []struct {
		name       string
		backupName string
		unit       backupv1.UnitIdentifier
		want       string
	}{
		{name: "legacy", backupName: "mybackup", unit: backupv1.UnitIdentifier{Type: UnitTypeLegacy}, want: "mybackup"},
		{name: "global-config", backupName: "mybackup", unit: backupv1.UnitIdentifier{Type: backupv1.UnitTypeGlobalConfig}, want: "mybackup-global-config"},
		{name: "dogu", backupName: "mybackup", unit: backupv1.UnitIdentifier{Type: backupv1.UnitTypeDogu, Name: "redmine"}, want: "mybackup-dogu-redmine"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ProviderBackupName(tt.backupName, tt.unit))
		})
	}
}

func TestRestoreState_InvalidExclusions(t *testing.T) {
	unitBased := RestoreState{
		Kind: KindUnits,
		Units: []UnitBackup{
			{Type: backupv1.UnitTypeComponents, ProviderBackup: &ProviderBackup{}},
			{Type: backupv1.UnitTypeDogu, Name: "redmine", ProviderBackup: &ProviderBackup{}},
		},
		V2Dogus:    []string{"cockpit"},
		Components: []string{"k8s-loki"},
	}
	withoutComponents := unitBased
	withoutComponents.Components = nil
	legacy := RestoreState{Kind: KindLegacy, Units: []UnitBackup{{Type: UnitTypeLegacy}}}
	legacyNotSupported := []string{"exclusions and options are not supported for a legacy backup"}

	tests := []struct {
		name  string
		spec  backupv1.RestoreSpec
		state RestoreState
		want  []string
	}{
		{name: "no exclusions", state: unitBased},
		{
			name:  "excludedDogus V2 dogu and dogu not in the backup",
			spec:  backupv1.RestoreSpec{ExcludedDogus: []string{"redmine", "cockpit", "ghost"}},
			state: unitBased,
			want:  []string{"excludedDogus: cockpit is a V2 dogu", "excludedDogus: ghost is not in the backup"},
		},
		{
			name:  "excludedComponents not in the backup",
			spec:  backupv1.RestoreSpec{ExcludedComponents: []string{"k8s-loki", "k8s-foo"}},
			state: unitBased,
			want:  []string{"excludedComponents: k8s-foo is not in the backup"},
		},
		{
			name:  "backup without components invalidates every excludedComponents entry",
			spec:  backupv1.RestoreSpec{ExcludedComponents: []string{"k8s-loki", "k8s-foo"}},
			state: withoutComponents,
			want:  []string{"excludedComponents: k8s-loki is not in the backup", "excludedComponents: k8s-foo is not in the backup"},
		},
		{
			name:  "ignoreV2Dogus without doguv2 unit is fine",
			spec:  backupv1.RestoreSpec{Options: &backupv1.UnitOptions{IgnoreV2Dogus: true}},
			state: unitBased,
		},
		{name: "legacy without exclusions", state: legacy},
		{name: "legacy with excludedDogus", spec: backupv1.RestoreSpec{ExcludedDogus: []string{"redmine"}}, state: legacy, want: legacyNotSupported},
		{name: "legacy with excludedComponents", spec: backupv1.RestoreSpec{ExcludedComponents: []string{"k8s-loki"}}, state: legacy, want: legacyNotSupported},
		{name: "legacy with option", spec: backupv1.RestoreSpec{Options: &backupv1.UnitOptions{IgnoreGlobalConfig: true}}, state: legacy, want: legacyNotSupported},
		{name: "legacy with empty options", spec: backupv1.RestoreSpec{Options: &backupv1.UnitOptions{}}, state: legacy},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.state.InvalidExclusions(tt.spec))
		})
	}
}

func TestUnitBackup_IsUsable(t *testing.T) {
	unitInPhase := func(unitType backupv1.UnitType, phase velerov1.BackupPhase) UnitBackup {
		return UnitBackup{Type: unitType, ProviderBackup: &ProviderBackup{Phase: phase}}
	}

	tests := []struct {
		name string
		unit UnitBackup
		want bool
	}{
		{name: "unit completed", unit: unitInPhase(backupv1.UnitTypeComponents, velerov1.BackupPhaseCompleted), want: true},
		{name: "unit partially failed", unit: unitInPhase(backupv1.UnitTypeComponents, velerov1.BackupPhasePartiallyFailed), want: false},
		{name: "unit failed", unit: unitInPhase(backupv1.UnitTypeComponents, velerov1.BackupPhaseFailed), want: false},
		{name: "unit in progress", unit: unitInPhase(backupv1.UnitTypeComponents, velerov1.BackupPhaseInProgress), want: false},
		{name: "unit missing", unit: UnitBackup{Type: backupv1.UnitTypeComponents}, want: false},
		{name: "legacy completed", unit: unitInPhase(UnitTypeLegacy, velerov1.BackupPhaseCompleted), want: true},
		{name: "legacy partially failed", unit: unitInPhase(UnitTypeLegacy, velerov1.BackupPhasePartiallyFailed), want: false},
		{name: "legacy failed", unit: unitInPhase(UnitTypeLegacy, velerov1.BackupPhaseFailed), want: false},
		{name: "legacy in progress", unit: unitInPhase(UnitTypeLegacy, velerov1.BackupPhaseInProgress), want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.unit.IsUsable())
		})
	}
}
