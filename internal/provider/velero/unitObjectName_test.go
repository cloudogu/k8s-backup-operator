package velero

import (
	"strings"
	"testing"

	backupv1 "github.com/cloudogu/k8s-backup-lib/api/v1"
	"github.com/stretchr/testify/assert"
)

func TestUnitObjectName(t *testing.T) {
	t.Run("Concatenate parent name and unit type", func(t *testing.T) {
		assert.Equal(t, "backup-misc", UnitObjectName("backup", backupv1.UnitIdentifier{Type: backupv1.UnitTypeMisc}))
	})

	t.Run("Concatenate parent name, unit type and dogu name", func(t *testing.T) {
		assert.Equal(t, "backup-dogu-ldap", UnitObjectName("backup", backupv1.UnitIdentifier{Type: backupv1.UnitTypeDogu, Name: "ldap"}))
	})

	t.Run("Truncate long names and keep them unique", func(t *testing.T) {
		parentName := strings.Repeat("b", 240)

		first := UnitObjectName(parentName, backupv1.UnitIdentifier{Type: backupv1.UnitTypeDogu, Name: "dogu-with-a-long-name-a"})
		second := UnitObjectName(parentName, backupv1.UnitIdentifier{Type: backupv1.UnitTypeDogu, Name: "dogu-with-a-long-name-b"})

		assert.Len(t, first, maxObjectNameLength)
		assert.Len(t, second, maxObjectNameLength)
		assert.NotEqual(t, first, second)
		assert.True(t, strings.HasPrefix(first, parentName))
	})

	t.Run("Suffix truncated names with the first 8 hex characters of the fnv-64a hash of the full name", func(t *testing.T) {
		got := UnitObjectName(strings.Repeat("a", 250), backupv1.UnitIdentifier{Type: backupv1.UnitTypeDogu, Name: "redmine"})

		assert.Equal(t, strings.Repeat("a", 244)+"-011ddf00", got)
	})
}
