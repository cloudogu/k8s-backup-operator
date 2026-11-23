package state

import (
	"reflect"
	"strings"
	"testing"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/stretchr/testify/assert"
)

func TestBackupState_GlobalConfigUnitAnnotations(t *testing.T) {
	want := map[string]string{
		"k8s.cloudogu.com/backup-fqdn":   "fqdn.example.com",
		"k8s.cloudogu.com/backup-domain": "domain.example.com",
	}
	s := BackupState{
		Fqdn:   "fqdn.example.com",
		Domain: "domain.example.com",
	}
	if got := s.GlobalConfigUnitAnnotations(); !reflect.DeepEqual(got, want) {
		t.Errorf("GlobalConfigUnitAnnotations() = %v, want %v", got, want)
	}
}

func TestBackupState_CommonUnitAnnotations(t *testing.T) {
	type fields struct {
		BlueprintId         string
		Units               []backupv1.UnitIdentifier
		InstalledDogusV2    []backupv1.DoguVersion
		InstalledDogusV3    []backupv1.DoguVersion
		InstalledComponents []backupv1.ComponentVersion
	}
	tests := []struct {
		name   string
		fields fields
		want   map[string]string
	}{
		{
			name: "everything included",
			fields: fields{
				BlueprintId: "myBlueprint",
				Units: []backupv1.UnitIdentifier{
					{Type: backupv1.UnitTypeDogu, Name: "baseline"},
					{Type: backupv1.UnitTypeDoguV2},
					{Type: backupv1.UnitTypeGlobalConfig},
					{Type: backupv1.UnitTypeComponents},
				},
				InstalledDogusV2:    []backupv1.DoguVersion{{Name: "cockpit", Version: "1.2.3"}},
				InstalledDogusV3:    []backupv1.DoguVersion{{Name: "baseline", Version: "4.5.6"}},
				InstalledComponents: []backupv1.ComponentVersion{{Name: "test1", Version: "1.2.3"}, {Name: "test2", Version: "4.5.6"}},
			},
			want: map[string]string{
				"k8s.cloudogu.com/backup":                    strings.Repeat("a", 64),
				"backup.cloudogu.com/blueprintId":            "myBlueprint",
				"k8s.cloudogu.com/backup-units":              `[{"type":"dogu","name":"baseline"},{"type":"doguv2"},{"type":"global-config"},{"type":"components"}]`,
				"k8s.cloudogu.com/backup-installed-dogus-v3": `[{"name":"baseline","version":"4.5.6"}]`,
				"k8s.cloudogu.com/backup-installed-dogus-v2": `[{"name":"cockpit","version":"1.2.3"}]`,
				"k8s.cloudogu.com/backup-components":         `[{"name":"test1","version":"1.2.3"},{"name":"test2","version":"4.5.6"}]`,
			},
		},
		{
			name: "should not add empty annotations for v3/v3 dogus, components and blueprint id",
			fields: fields{
				BlueprintId: "",
				Units: []backupv1.UnitIdentifier{
					{Type: backupv1.UnitTypeDogu, Name: "baseline"},
					{Type: backupv1.UnitTypeDoguV2},
					{Type: backupv1.UnitTypeGlobalConfig},
					{Type: backupv1.UnitTypeComponents},
				},
			},
			want: map[string]string{
				"k8s.cloudogu.com/backup":       strings.Repeat("a", 64),
				"k8s.cloudogu.com/backup-units": `[{"type":"dogu","name":"baseline"},{"type":"doguv2"},{"type":"global-config"},{"type":"components"}]`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := BackupState{
				Name:             testBackupName,
				BlueprintId:      tt.fields.BlueprintId,
				Units:            tt.fields.Units,
				InstalledDogusV2: tt.fields.InstalledDogusV2,
				InstalledDogusV3: tt.fields.InstalledDogusV3,
				Components:       tt.fields.InstalledComponents,
			}
			got, err := s.CommonUnitAnnotations()
			assert.NoError(t, err)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CommonUnitAnnotations() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBackupState_CommonLabels(t *testing.T) {
	want := map[string]string{
		"k8s.cloudogu.com/backup":          "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa-d96f0f85",
		"k8s.cloudogu.com/backup-coverage": "full",
	}
	s := BackupState{
		Name:     testBackupName,
		Coverage: backupv1.BackupCoverageFull,
	}
	assert.Equalf(t, want, s.CommonLabels(), "CommonLabels()")
}
