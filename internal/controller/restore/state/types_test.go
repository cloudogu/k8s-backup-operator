package state

import (
	"testing"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/stretchr/testify/assert"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
)

var (
	globalConfigUnit = UnitBackup{Type: backupv1.UnitTypeGlobalConfig, ProviderBackup: &velerov1.Backup{}}
	miscUnit         = UnitBackup{Type: backupv1.UnitTypeMisc, ProviderBackup: &velerov1.Backup{}}
	doguV2Unit       = UnitBackup{Type: backupv1.UnitTypeDoguV2, ProviderBackup: &velerov1.Backup{}}
	postfixUnit      = UnitBackup{Type: backupv1.UnitTypeDogu, Name: "postfix", ProviderBackup: &velerov1.Backup{}}
	redmineUnit      = UnitBackup{Type: backupv1.UnitTypeDogu, Name: "redmine", ProviderBackup: &velerov1.Backup{}}
)

func unitState() RestoreState {
	return RestoreState{
		Kind:       KindUnits,
		Units:      []UnitBackup{globalConfigUnit, miscUnit, doguV2Unit, postfixUnit, redmineUnit},
		Components: []string{"k8s-loki", "k8s-velero"},
	}
}

func TestRestoreState_UnitsToRestore(t *testing.T) {
	missingMisc := unitState()
	missingMisc.Units[1].ProviderBackup = nil
	missingMisc.Components = nil

	withoutComponents := unitState()
	withoutComponents.Components = nil

	tests := []struct {
		name  string
		state RestoreState
		spec  backupv1.RestoreSpec
		want  []UnitBackup
	}{
		{name: "all units", state: unitState(), want: []UnitBackup{globalConfigUnit, miscUnit, doguV2Unit, postfixUnit, redmineUnit}},
		{
			name:  "ignoreGlobalConfig drops the global-config unit",
			state: unitState(),
			spec:  backupv1.RestoreSpec{Options: backupv1.UnitOptions{IgnoreGlobalConfig: true}},
			want:  []UnitBackup{miscUnit, doguV2Unit, postfixUnit, redmineUnit},
		},
		{
			name:  "ignoreMisc drops the misc unit",
			state: unitState(),
			spec:  backupv1.RestoreSpec{Options: backupv1.UnitOptions{IgnoreMisc: true}},
			want:  []UnitBackup{globalConfigUnit, doguV2Unit, postfixUnit, redmineUnit},
		},
		{
			name:  "ignoreV2Dogus drops the doguv2 unit",
			state: unitState(),
			spec:  backupv1.RestoreSpec{Options: backupv1.UnitOptions{IgnoreV2Dogus: true}},
			want:  []UnitBackup{globalConfigUnit, miscUnit, postfixUnit, redmineUnit},
		},
		{
			name:  "excludedDogus drops V3 dogu units only",
			state: unitState(),
			spec:  backupv1.RestoreSpec{ExcludedDogus: []string{"redmine", "cockpit"}},
			want:  []UnitBackup{globalConfigUnit, miscUnit, doguV2Unit, postfixUnit},
		},
		{
			name:  "some components excluded keeps the misc unit",
			state: unitState(),
			spec:  backupv1.RestoreSpec{ExcludedComponents: []string{"k8s-loki"}},
			want:  []UnitBackup{globalConfigUnit, miscUnit, doguV2Unit, postfixUnit, redmineUnit},
		},
		{
			name:  "all components excluded drops the misc unit",
			state: unitState(),
			spec:  backupv1.RestoreSpec{ExcludedComponents: []string{"k8s-loki", "k8s-velero"}},
			want:  []UnitBackup{globalConfigUnit, doguV2Unit, postfixUnit, redmineUnit},
		},
		{
			name:  "misc backup without components drops the misc unit",
			state: withoutComponents,
			want:  []UnitBackup{globalConfigUnit, doguV2Unit, postfixUnit, redmineUnit},
		},
		{
			name:  "missing misc backup keeps the misc unit",
			state: missingMisc,
			want:  []UnitBackup{globalConfigUnit, {Type: backupv1.UnitTypeMisc}, doguV2Unit, postfixUnit, redmineUnit},
		},
		{
			name:  "legacy unit is kept",
			state: RestoreState{Kind: KindLegacy, Units: []UnitBackup{{Type: UnitTypeLegacy}}},
			spec:  backupv1.RestoreSpec{ExcludedDogus: []string{"redmine"}, Options: backupv1.UnitOptions{IgnoreMisc: true}},
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

	state.UnitsToRestore(backupv1.RestoreSpec{Options: backupv1.UnitOptions{IgnoreGlobalConfig: true}})

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
		{name: "misc", restoreName: "myrestore", unit: backupv1.UnitIdentifier{Type: backupv1.UnitTypeMisc}, want: "myrestore-misc"},
		{name: "doguv2", restoreName: "myrestore", unit: backupv1.UnitIdentifier{Type: backupv1.UnitTypeDoguV2}, want: "myrestore-doguv2"},
		{name: "dogu", restoreName: "myrestore", unit: backupv1.UnitIdentifier{Type: backupv1.UnitTypeDogu, Name: "redmine"}, want: "myrestore-dogu-redmine"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ProviderRestoreName(tt.restoreName, tt.unit))
		})
	}
}
