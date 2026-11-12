package backup

import (
	"strings"
	"testing"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/cloudogu/k8s-backup-operator/internal/controller/backup/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestUnitBackupName(t *testing.T) {
	t.Run("Concatenate backup name and unit type", func(t *testing.T) {
		assert.Equal(t, "backup-misc", unitBackupName("backup", backupv1.UnitIdentifier{Type: backupv1.UnitTypeMisc}))
	})

	t.Run("Concatenate backup name, unit type and dogu name", func(t *testing.T) {
		assert.Equal(t, "backup-dogu-ldap", unitBackupName("backup", backupv1.UnitIdentifier{Type: backupv1.UnitTypeDogu, Name: "ldap"}))
	})

	t.Run("Truncate long names and keep them unique", func(t *testing.T) {
		backupName := strings.Repeat("b", 240)

		first := unitBackupName(backupName, backupv1.UnitIdentifier{Type: backupv1.UnitTypeDogu, Name: "dogu-with-a-long-name-a"})
		second := unitBackupName(backupName, backupv1.UnitIdentifier{Type: backupv1.UnitTypeDogu, Name: "dogu-with-a-long-name-b"})

		assert.Len(t, first, maxObjectNameLength)
		assert.Len(t, second, maxObjectNameLength)
		assert.NotEqual(t, first, second)
		assert.True(t, strings.HasPrefix(first, backupName))
	})
}

func TestDesiredUnitBackups(t *testing.T) {
	backup := newBackupForTest("ns", "backup")
	backup.Spec.ExcludedComponents = []string{"k8s-loki"}
	backupState := newBackupStateForTest(backup.Name)
	backupState.BlueprintId = "blueprint-id"

	unitBackups, err := desiredUnitBackups(backup, backupState, "default")

	require.NoError(t, err)
	require.Len(t, unitBackups, 4)
	byName := map[string]*velerov1.Backup{}
	for _, unitBackup := range unitBackups {
		byName[unitBackup.Name] = unitBackup

		assert.Equal(t, "ns", unitBackup.Namespace)
		assert.Equal(t, "ces", unitBackup.Labels["app"])
		assert.Equal(t, "backup", unitBackup.Labels["k8s.cloudogu.com/part-of"])
		assert.Equal(t, backup.Name, unitBackup.Labels[backupv1.BackupLabelKey])
		assert.Equal(t, string(backupv1.BackupCoverageFull), unitBackup.Labels[backupv1.BackupCoverageLabelKey])
		assert.Equal(t, `[{"type":"doguv2"},{"type":"dogu","name":"ldap"},{"type":"global-config"},{"type":"misc"}]`, unitBackup.Annotations[backupv1.UnitsAnnotationKey])
		assert.Equal(t, `[{"name":"cas","version":"7.0.0-1"}]`, unitBackup.Annotations[backupv1.InstalledDogusV2AnnotationKey])
		assert.Equal(t, `[{"name":"ldap","version":"2.6.8-1"}]`, unitBackup.Annotations[backupv1.InstalledDogusV3AnnotationKey])
		assert.Equal(t, "blueprint-id", unitBackup.Annotations[backupv1.BlueprintIDAnnotationKey])
	}

	t.Run("dogu unit", func(t *testing.T) {
		unit := byName["backup-dogu-ldap"]
		require.NotNil(t, unit)
		assert.Equal(t, string(backupv1.UnitTypeDogu), unit.Labels[backupv1.BackupTypeLabelKey])
		assert.Equal(t, "ldap", unit.Labels[backupv1.DoguNameLabelKey])
		assert.Equal(t, "2.6.8-1", unit.Labels[backupv1.DoguVersionLabelKey])
		assert.Equal(t, &metav1.LabelSelector{MatchLabels: map[string]string{"k8s.cloudogu.com/dogu.name": "ldap"}}, unit.Spec.LabelSelector)
		assert.Equal(t, []string{"configmaps", "secrets", "persistentvolumeclaims", "dogus.k8s.cloudogu.com"}, unit.Spec.IncludedResources)
		assert.NotContains(t, unit.Annotations, backupv1.FQDNAnnotationKey)
		assert.NotContains(t, unit.Annotations, backupv1.ComponentsAnnotationKey)
	})

	t.Run("doguv2 unit", func(t *testing.T) {
		unit := byName["backup-doguv2"]
		require.NotNil(t, unit)
		assert.Equal(t, string(backupv1.UnitTypeDoguV2), unit.Labels[backupv1.BackupTypeLabelKey])
		assert.NotContains(t, unit.Labels, backupv1.DoguNameLabelKey)
		assert.Equal(t, &metav1.LabelSelector{MatchExpressions: []metav1.LabelSelectorRequirement{
			{Key: "k8s.cloudogu.com/dogu.name", Operator: metav1.LabelSelectorOpIn, Values: []string{"cas"}},
		}}, unit.Spec.LabelSelector)
		assert.Equal(t, []string{"configmaps", "secrets", "persistentvolumeclaims", "dogus.k8s.cloudogu.com"}, unit.Spec.IncludedResources)
	})

	t.Run("global-config unit", func(t *testing.T) {
		unit := byName["backup-global-config"]
		require.NotNil(t, unit)
		assert.Equal(t, string(backupv1.UnitTypeGlobalConfig), unit.Labels[backupv1.BackupTypeLabelKey])
		assert.Equal(t, "ces.example.com", unit.Annotations[backupv1.FQDNAnnotationKey])
		assert.Equal(t, "example.com", unit.Annotations[backupv1.DomainAnnotationKey])
		assert.Equal(t, &metav1.LabelSelector{MatchLabels: map[string]string{"k8s.cloudogu.com/type": "global-config"}}, unit.Spec.LabelSelector)
		assert.Equal(t, []string{"configmaps"}, unit.Spec.IncludedResources)
	})

	t.Run("misc unit", func(t *testing.T) {
		unit := byName["backup-misc"]
		require.NotNil(t, unit)
		assert.Equal(t, string(backupv1.UnitTypeMisc), unit.Labels[backupv1.BackupTypeLabelKey])
		assert.Contains(t, unit.Annotations, backupv1.ComponentsAnnotationKey)
		assert.Equal(t, &metav1.LabelSelector{MatchExpressions: []metav1.LabelSelectorRequirement{
			{Key: "k8s.cloudogu.com/backup-scope", Operator: metav1.LabelSelectorOpExists},
			{Key: "k8s.cloudogu.com/component.name", Operator: metav1.LabelSelectorOpIn, Values: []string{"k8s-lop-idp"}},
		}}, unit.Spec.LabelSelector)
		assert.Equal(t, []string{"configmaps", "secrets", "persistentvolumeclaims"}, unit.Spec.IncludedResources)
	})

	t.Run("misc unit selects all components in one requirement", func(t *testing.T) {
		miscOnly := state.BackupState{
			Name:  "backup",
			Units: []backupv1.UnitIdentifier{{Type: backupv1.UnitTypeMisc}},
			Components: []backupv1.ComponentStatus{
				{Name: "k8s-lop-idp", Version: "1.0.0"},
				{Name: "k8s-loki", Version: "2.0.0"},
			},
		}

		unitBackups, err := desiredUnitBackups(newBackupForTest("ns", "backup"), miscOnly, "default")

		require.NoError(t, err)
		require.Len(t, unitBackups, 1)
		assert.Equal(t, &metav1.LabelSelector{MatchExpressions: []metav1.LabelSelectorRequirement{
			{Key: "k8s.cloudogu.com/backup-scope", Operator: metav1.LabelSelectorOpExists},
			{Key: "k8s.cloudogu.com/component.name", Operator: metav1.LabelSelectorOpIn, Values: []string{"k8s-lop-idp", "k8s-loki"}},
		}}, unitBackups[0].Spec.LabelSelector)
	})

	t.Run("misc unit without components fails", func(t *testing.T) {
		miscOnly := state.BackupState{Name: "backup", Units: []backupv1.UnitIdentifier{{Type: backupv1.UnitTypeMisc}}}

		unitBackups, err := desiredUnitBackups(newBackupForTest("ns", "backup"), miscOnly, "default")

		require.Error(t, err)
		assert.ErrorContains(t, err, "misc unit requires at least one component")
		assert.Nil(t, unitBackups)
	})

	t.Run("Do not modify the default labels", func(t *testing.T) {
		assert.Equal(t, map[string]string{"app": "ces", "k8s.cloudogu.com/part-of": "backup"}, defaultLabels)
	})
}

func TestIncludedResources(t *testing.T) {
	tests := []struct {
		name     string
		unitType backupv1.UnitType
		expected []string
	}{
		{name: "dogu", unitType: backupv1.UnitTypeDogu, expected: []string{"configmaps", "secrets", "persistentvolumeclaims", "dogus.k8s.cloudogu.com"}},
		{name: "doguv2", unitType: backupv1.UnitTypeDoguV2, expected: []string{"configmaps", "secrets", "persistentvolumeclaims", "dogus.k8s.cloudogu.com"}},
		{name: "global-config", unitType: backupv1.UnitTypeGlobalConfig, expected: []string{"configmaps"}},
		{name: "misc", unitType: backupv1.UnitTypeMisc, expected: []string{"configmaps", "secrets", "persistentvolumeclaims"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := includedResources(backupv1.UnitIdentifier{Type: tt.unitType})

			require.NoError(t, err)
			assert.Equal(t, tt.expected, actual)
		})
	}

	t.Run("Fail for unknown unit type", func(t *testing.T) {
		actual, err := includedResources(backupv1.UnitIdentifier{Type: "unknown"})

		require.Error(t, err)
		assert.ErrorContains(t, err, `failed to determine included resources for unit type "unknown"`)
		assert.Nil(t, actual)
	})
}
