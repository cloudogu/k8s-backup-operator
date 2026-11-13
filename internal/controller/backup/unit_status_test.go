package backup

import (
	"testing"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func allStartedUnitsForTest() map[string]struct{} {
	return map[string]struct{}{
		"backup-doguv2":        {},
		"backup-dogu-ldap":     {},
		"backup-global-config": {},
		"backup-misc":          {},
	}
}

func TestPopulateUnitStatus(t *testing.T) {
	t.Run("Fill an empty status with the metadata and provider references of all units", func(t *testing.T) {
		backupState := newBackupStateForTest("backup")
		backupState.BlueprintId = "blueprint-id"
		backupState.Components = []backupv1.ComponentStatus{{Name: "k8s-loki", Version: "3.0.0-1"}}
		status := &backupv1.BackupStatus{}

		populateUnitStatus(status, "backup", backupState, allStartedUnitsForTest())

		assert.Equal(t, &backupv1.BackupStatus{
			BackupCoverage: backupv1.BackupCoverageFull,
			BlueprintID:    "blueprint-id",
			GlobalConfig: &backupv1.GlobalConfigStatus{
				UnitStatus: backupv1.UnitStatus{ProviderRef: &backupv1.ProviderRef{Name: "backup-global-config"}},
				FQDN:       "ces.example.com",
				Domain:     "example.com",
			},
			V2Dogus: &backupv1.V2DogusUnitStatus{
				UnitStatus: backupv1.UnitStatus{ProviderRef: &backupv1.ProviderRef{Name: "backup-doguv2"}},
				Dogus:      []backupv1.DoguVersion{{Name: "cas", Version: "7.0.0-1"}},
			},
			Dogus: []backupv1.DoguUnitStatus{{
				DoguVersion: backupv1.DoguVersion{Name: "ldap", Version: "2.6.8-1"},
				UnitStatus:  backupv1.UnitStatus{ProviderRef: &backupv1.ProviderRef{Name: "backup-dogu-ldap"}},
			}},
			Components: []backupv1.ComponentStatus{{Name: "k8s-loki", Version: "3.0.0-1"}},
			Misc:       &backupv1.UnitStatus{ProviderRef: &backupv1.ProviderRef{Name: "backup-misc"}},
		}, status)
	})

	t.Run("Do not set a provider reference for units which were not started", func(t *testing.T) {
		status := &backupv1.BackupStatus{}

		populateUnitStatus(status, "backup", newBackupStateForTest("backup"), map[string]struct{}{"backup-misc": {}})

		require.NotNil(t, status.GlobalConfig)
		assert.Nil(t, status.GlobalConfig.ProviderRef)
		assert.Equal(t, "ces.example.com", status.GlobalConfig.FQDN)
		require.NotNil(t, status.V2Dogus)
		assert.Nil(t, status.V2Dogus.ProviderRef)
		require.Len(t, status.Dogus, 1)
		assert.Nil(t, status.Dogus[0].ProviderRef)
		assert.Equal(t, "2.6.8-1", status.Dogus[0].Version)
		require.NotNil(t, status.Misc)
		assert.Equal(t, &backupv1.ProviderRef{Name: "backup-misc"}, status.Misc.ProviderRef)
	})

	t.Run("Add the provider reference when the unit is started in a later run", func(t *testing.T) {
		status := &backupv1.BackupStatus{}
		populateUnitStatus(status, "backup", newBackupStateForTest("backup"), map[string]struct{}{})
		require.Len(t, status.Dogus, 1)
		require.Nil(t, status.Dogus[0].ProviderRef)

		populateUnitStatus(status, "backup", newBackupStateForTest("backup"), allStartedUnitsForTest())

		require.Len(t, status.Dogus, 1)
		assert.Equal(t, &backupv1.ProviderRef{Name: "backup-dogu-ldap"}, status.Dogus[0].ProviderRef)
	})

	t.Run("Keep the state, reason, message and existing values of the units", func(t *testing.T) {
		finished := backupv1.UnitStatus{
			State:       backupv1.UnitStateFailed,
			Reason:      "ProviderBackupFailed",
			Message:     "failed",
			ProviderRef: &backupv1.ProviderRef{Name: "existing-ref"},
		}
		status := &backupv1.BackupStatus{
			BackupCoverage: backupv1.BackupCoveragePartial,
			BlueprintID:    "existing-blueprint",
			GlobalConfig:   &backupv1.GlobalConfigStatus{UnitStatus: finished, FQDN: "old.example.com", Domain: "old.com"},
			V2Dogus:        &backupv1.V2DogusUnitStatus{UnitStatus: finished, Dogus: []backupv1.DoguVersion{{Name: "redmine", Version: "5.0.0-1"}}},
			Dogus: []backupv1.DoguUnitStatus{
				{DoguVersion: backupv1.DoguVersion{Name: "other", Version: "1.0.0-1"}, UnitStatus: finished},
				{DoguVersion: backupv1.DoguVersion{Name: "ldap", Version: "2.6.7-1"}, UnitStatus: finished},
			},
			Components: []backupv1.ComponentStatus{{Name: "k8s-loki", Version: "2.0.0-1"}},
			Misc:       finished.DeepCopy(),
		}
		expected := status.DeepCopy()

		populateUnitStatus(status, "backup", newBackupStateForTest("backup"), allStartedUnitsForTest())

		assert.Equal(t, expected, status)
	})

	t.Run("Return the same status when running twice", func(t *testing.T) {
		status := &backupv1.BackupStatus{}
		populateUnitStatus(status, "backup", newBackupStateForTest("backup"), allStartedUnitsForTest())
		expected := status.DeepCopy()

		populateUnitStatus(status, "backup", newBackupStateForTest("backup"), allStartedUnitsForTest())

		assert.Equal(t, expected, status)
	})

	t.Run("Only add the status of units contained in the backup", func(t *testing.T) {
		backupState := newBackupStateForTest("backup")
		backupState.Units = []backupv1.UnitIdentifier{{Type: backupv1.UnitTypeDogu, Name: "ldap"}}
		status := &backupv1.BackupStatus{}

		populateUnitStatus(status, "backup", backupState, allStartedUnitsForTest())

		assert.Nil(t, status.GlobalConfig)
		assert.Nil(t, status.V2Dogus)
		assert.Nil(t, status.Misc)
		assert.Empty(t, status.Components)
		require.Len(t, status.Dogus, 1)
		assert.Equal(t, "ldap", status.Dogus[0].Name)
	})
}
